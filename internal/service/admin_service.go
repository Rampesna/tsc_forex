package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/presence"
)

type PresenceReader interface {
	Online(context.Context) ([]presence.Entry, error)
}

type AdminService struct {
	db       *gorm.DB
	presence PresenceReader
	notifier *Notifier
	adminCfg config.AdminConfig
	aiCfg    config.AIConfig
	log      *zap.Logger
}

func NewAdminService(db *gorm.DB, presenceReader PresenceReader, notifier *Notifier, adminCfg config.AdminConfig, aiCfg config.AIConfig, log *zap.Logger) *AdminService {
	if log == nil {
		log = zap.NewNop()
	}
	return &AdminService{
		db: db, presence: presenceReader, notifier: notifier,
		adminCfg: adminCfg, aiCfg: aiCfg, log: log,
	}
}

type AdminNamedValue struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type AdminTimeValue struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
	Count int     `json:"count,omitempty"`
}

type AdminRevenueWindow struct {
	Sales       int     `json:"sales"`
	GrossTRY    float64 `json:"gross_try"`
	Label       string  `json:"label"`
	IsEstimated bool    `json:"is_estimated"`
}

type AdminDashboard struct {
	GeneratedAt time.Time `json:"generated_at"`
	RangeDays   int       `json:"range_days"`
	Totals      struct {
		Users               int64 `json:"users"`
		NewUsers            int64 `json:"new_users"`
		OnlineUsers         int   `json:"online_users"`
		ActiveSubscriptions int64 `json:"active_subscriptions"`
		ActiveAlerts        int64 `json:"active_alerts"`
		Favorites           int64 `json:"favorites"`
		AIResearchUses      int64 `json:"ai_research_uses"`
	} `json:"totals"`
	Revenue struct {
		Daily   AdminRevenueWindow `json:"daily"`
		Weekly  AdminRevenueWindow `json:"weekly"`
		Monthly AdminRevenueWindow `json:"monthly"`
		Series  []AdminTimeValue   `json:"series"`
		Notice  string             `json:"notice"`
	} `json:"revenue"`
	UserSeries           []AdminTimeValue  `json:"user_series"`
	ActivitySeries       []AdminTimeValue  `json:"activity_series"`
	PlatformDistribution []AdminNamedValue `json:"platform_distribution"`
	PlanDistribution     []AdminNamedValue `json:"plan_distribution"`
	ProviderDistribution []AdminNamedValue `json:"provider_distribution"`
	SalesByPlan          []AdminNamedValue `json:"sales_by_plan"`
	TopFavoriteAssets    []AdminNamedValue `json:"top_favorite_assets"`
	RecentlyOnline       []AdminOnlineUser `json:"recently_online"`
}

type adminPurchaseEvent struct {
	PurchasedAt time.Time
	ProductID   string
	Provider    string
}

func (s *AdminService) Dashboard(ctx context.Context, days int) (AdminDashboard, error) {
	var out AdminDashboard
	if s == nil || s.db == nil {
		return out, errors.New("admin database unavailable")
	}
	if days != 7 && days != 30 && days != 90 && days != 365 {
		days = 30
	}
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -days+1).Truncate(24 * time.Hour)
	out.GeneratedAt = now
	out.RangeDays = days

	counts := []struct {
		dest  *int64
		model any
		query string
		args  []any
	}{
		{&out.Totals.Users, &models.User{}, "role = ?", []any{"user"}},
		{&out.Totals.NewUsers, &models.User{}, "role = ? AND created_at >= ?", []any{"user", from}},
		{&out.Totals.ActiveSubscriptions, &models.SubscriptionEntitlement{}, "status = ? AND expires_at > ? AND revoked_at IS NULL", []any{"active", now}},
		{&out.Totals.ActiveAlerts, &models.PriceAlert{}, "active = ?", []any{true}},
		{&out.Totals.Favorites, &models.WatchlistItem{}, "1 = 1", nil},
		{&out.Totals.AIResearchUses, &models.FeatureUsage{}, "feature = ? AND created_at >= ?", []any{FeatureAIResearch, from}},
	}
	for _, item := range counts {
		if err := s.db.WithContext(ctx).Model(item.model).Where(item.query, item.args...).Count(item.dest).Error; err != nil {
			return out, err
		}
	}

	online, err := s.OnlineUsers(ctx)
	if err != nil {
		s.log.Warn("admin online snapshot failed", zap.Error(err))
	} else {
		out.Totals.OnlineUsers = len(online)
		if len(online) > 8 {
			online = online[:8]
		}
		out.RecentlyOnline = online
	}

	out.UserSeries, err = s.dailyCountSeries(ctx, "users", "created_at", "role = ?", []any{"user"}, from, now)
	if err != nil {
		return out, err
	}
	out.ActivitySeries, err = s.activitySeries(ctx, from, now)
	if err != nil {
		return out, err
	}
	if out.PlatformDistribution, err = s.platformDistribution(ctx); err != nil {
		return out, err
	}
	if out.PlanDistribution, err = s.planDistribution(ctx, now); err != nil {
		return out, err
	}
	if out.ProviderDistribution, err = s.providerDistribution(ctx); err != nil {
		return out, err
	}
	if out.TopFavoriteAssets, err = s.topFavoriteAssets(ctx); err != nil {
		return out, err
	}

	var events []adminPurchaseEvent
	err = s.db.WithContext(ctx).Raw(`
		SELECT purchased_at, product_id, 'apple' AS provider
		FROM apple_transactions
		WHERE purchased_at >= ? AND revoked_at IS NULL AND LOWER(environment) = 'production'
		UNION ALL
		SELECT purchased_at, product_id, 'google' AS provider
		FROM google_play_transactions
		WHERE purchased_at >= ? AND revoked_at IS NULL AND LOWER(environment) = 'production'
		ORDER BY purchased_at ASC
	`, now.AddDate(-1, 0, 0), now.AddDate(-1, 0, 0)).Scan(&events).Error
	if err != nil {
		return out, err
	}
	priceByProduct := make(map[string]float64)
	nameByProduct := make(map[string]string)
	for _, plan := range SubscriptionPlans() {
		priceByProduct[plan.ProductID] = plan.PriceHintTRY
		nameByProduct[plan.ProductID] = plan.Name
	}
	window := func(since time.Time, label string) AdminRevenueWindow {
		value := AdminRevenueWindow{Label: label, IsEstimated: true}
		for _, event := range events {
			if !event.PurchasedAt.Before(since) {
				value.Sales++
				value.GrossTRY += priceByProduct[event.ProductID]
			}
		}
		value.GrossTRY = math.Round(value.GrossTRY*100) / 100
		return value
	}
	out.Revenue.Daily = window(now.Add(-24*time.Hour), "Son 24 saat")
	out.Revenue.Weekly = window(now.AddDate(0, 0, -7), "Son 7 gün")
	out.Revenue.Monthly = window(now.AddDate(0, 0, -30), "Son 30 gün")
	out.Revenue.Notice = "Tutarlar doğrulanmış mağaza işlemlerinin katalog fiyatlarıyla hesaplanan tahmini brüttür; vergi, mağaza komisyonu ve kur düzeltmeleri net gelire dahil değildir."

	dailyRevenue := make(map[string]*AdminTimeValue)
	salesPlans := make(map[string]float64)
	for _, event := range events {
		if event.PurchasedAt.Before(from) {
			continue
		}
		key := event.PurchasedAt.UTC().Format("2006-01-02")
		point := dailyRevenue[key]
		if point == nil {
			point = &AdminTimeValue{Time: key}
			dailyRevenue[key] = point
		}
		point.Count++
		point.Value += priceByProduct[event.ProductID]
		name := nameByProduct[event.ProductID]
		if name == "" {
			name = event.ProductID
		}
		salesPlans[name]++
	}
	for day := from; !day.After(now); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		point := dailyRevenue[key]
		if point == nil {
			point = &AdminTimeValue{Time: key}
		}
		point.Value = math.Round(point.Value*100) / 100
		out.Revenue.Series = append(out.Revenue.Series, *point)
	}
	out.SalesByPlan = namedValues(salesPlans)
	return out, nil
}

func (s *AdminService) dailyCountSeries(ctx context.Context, table, column, where string, args []any, from, to time.Time) ([]AdminTimeValue, error) {
	allowed := map[string]bool{"users": true}
	if !allowed[table] || column != "created_at" {
		return nil, errors.New("unsupported series")
	}
	type row struct {
		Day   time.Time
		Count int
	}
	var rows []row
	query := fmt.Sprintf("SELECT DATE_TRUNC('day', %s) AS day, COUNT(*) AS count FROM %s WHERE %s AND %s >= ? GROUP BY day ORDER BY day", column, table, where, column)
	allArgs := append(append([]any{}, args...), from)
	if err := s.db.WithContext(ctx).Raw(query, allArgs...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	byDay := make(map[string]int, len(rows))
	for _, item := range rows {
		byDay[item.Day.UTC().Format("2006-01-02")] = item.Count
	}
	out := make([]AdminTimeValue, 0, int(to.Sub(from).Hours()/24)+1)
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		out = append(out, AdminTimeValue{Time: key, Value: float64(byDay[key]), Count: byDay[key]})
	}
	return out, nil
}

func (s *AdminService) activitySeries(ctx context.Context, from, to time.Time) ([]AdminTimeValue, error) {
	type row struct {
		Day   time.Time
		Count int
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT day, SUM(count) AS count FROM (
			SELECT DATE_TRUNC('day', created_at) AS day, COUNT(*) AS count FROM gamification_events WHERE created_at >= ? GROUP BY day
			UNION ALL
			SELECT DATE_TRUNC('day', created_at) AS day, COUNT(*) AS count FROM feature_usages WHERE created_at >= ? GROUP BY day
			UNION ALL
			SELECT DATE_TRUNC('day', created_at) AS day, COUNT(*) AS count FROM portfolio_transactions WHERE created_at >= ? GROUP BY day
		) activity GROUP BY day ORDER BY day
	`, from, from, from).Scan(&rows).Error; err != nil {
		return nil, err
	}
	byDay := make(map[string]int, len(rows))
	for _, item := range rows {
		byDay[item.Day.UTC().Format("2006-01-02")] = item.Count
	}
	out := make([]AdminTimeValue, 0)
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		out = append(out, AdminTimeValue{Time: key, Value: float64(byDay[key]), Count: byDay[key]})
	}
	return out, nil
}

func (s *AdminService) platformDistribution(ctx context.Context) ([]AdminNamedValue, error) {
	type row struct {
		IOS     int
		Android int
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
			MAX(CASE WHEN d.platform = 'ios' THEN 1 ELSE 0 END) AS ios,
			MAX(CASE WHEN d.platform = 'android' THEN 1 ELSE 0 END) AS android
		FROM users u LEFT JOIN device_tokens d ON d.user_id = u.id
		WHERE u.role = 'user' GROUP BY u.id
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	values := map[string]float64{"iOS": 0, "Android": 0, "Çoklu cihaz": 0, "Bilinmiyor": 0}
	for _, row := range rows {
		switch {
		case row.IOS == 1 && row.Android == 1:
			values["Çoklu cihaz"]++
		case row.IOS == 1:
			values["iOS"]++
		case row.Android == 1:
			values["Android"]++
		default:
			values["Bilinmiyor"]++
		}
	}
	return namedValues(values), nil
}

func (s *AdminService) planDistribution(ctx context.Context, now time.Time) ([]AdminNamedValue, error) {
	type row struct {
		Name  string
		Count int
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(active.plan_code, 'free') AS name, COUNT(*) AS count
		FROM users u
		LEFT JOIN (
			SELECT user_id, plan_code FROM (
				SELECT user_id, plan_code, ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY updated_at DESC) AS rn
				FROM subscription_entitlements
				WHERE status = 'active' AND expires_at > ? AND revoked_at IS NULL
			) ranked WHERE rn = 1
		) active ON active.user_id = u.id
		WHERE u.role = 'user'
		GROUP BY COALESCE(active.plan_code, 'free')
	`, now).Scan(&rows).Error; err != nil {
		return nil, err
	}
	nameMap := map[string]string{PlanFree: "Ücretsiz", PlanPlus: "Başlangıç", PlanPro: "Pro", PlanExpert: "Uzman", PlanElite: "Elite"}
	out := make([]AdminNamedValue, 0, len(rows))
	for _, row := range rows {
		name := nameMap[row.Name]
		if name == "" {
			name = row.Name
		}
		out = append(out, AdminNamedValue{Name: name, Value: float64(row.Count)})
	}
	return out, nil
}

func (s *AdminService) providerDistribution(ctx context.Context) ([]AdminNamedValue, error) {
	type row struct {
		Name  string
		Count int
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(NULLIF(provider, ''), 'local') AS name, COUNT(*) AS count
		FROM users WHERE role = 'user' GROUP BY COALESCE(NULLIF(provider, ''), 'local')
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]AdminNamedValue, 0, len(rows))
	for _, row := range rows {
		out = append(out, AdminNamedValue{Name: strings.Title(row.Name), Value: float64(row.Count)})
	}
	return out, nil
}

func (s *AdminService) topFavoriteAssets(ctx context.Context) ([]AdminNamedValue, error) {
	type row struct {
		Name  string
		Count int
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT code AS name, COUNT(*) AS count FROM watchlist_items
		GROUP BY code ORDER BY count DESC, code ASC LIMIT 8
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]AdminNamedValue, 0, len(rows))
	for _, row := range rows {
		out = append(out, AdminNamedValue{Name: row.Name, Value: float64(row.Count)})
	}
	return out, nil
}

func namedValues(input map[string]float64) []AdminNamedValue {
	out := make([]AdminNamedValue, 0, len(input))
	for name, value := range input {
		out = append(out, AdminNamedValue{Name: name, Value: value})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Value == out[j].Value {
			return out[i].Name < out[j].Name
		}
		return out[i].Value > out[j].Value
	})
	return out
}

type AdminUserQuery struct {
	Search   string
	Plan     string
	Platform string
	UserID   *uuid.UUID
	Page     int
	PageSize int
}

type AdminUserSummary struct {
	ID                 uuid.UUID  `json:"id"`
	Email              string     `json:"email"`
	DisplayName        string     `json:"display_name"`
	AvatarURL          string     `json:"avatar_url"`
	Provider           string     `json:"provider"`
	PlanCode           string     `json:"plan_code"`
	PurchasePlatform   string     `json:"purchase_platform"`
	DevicePlatform     string     `json:"device_platform"`
	Favorites          int        `json:"favorites"`
	ActiveAlerts       int        `json:"active_alerts"`
	Badges             int        `json:"badges"`
	AIResearchUses     int        `json:"ai_research_uses"`
	CreatedAt          time.Time  `json:"created_at"`
	LastActiveAt       *time.Time `json:"last_active_at,omitempty"`
	SubscriptionExpiry *time.Time `json:"subscription_expiry,omitempty"`
	Online             bool       `json:"online"`
}

type AdminUsersPage struct {
	Items    []AdminUserSummary `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int64              `json:"total"`
}

func (s *AdminService) Users(ctx context.Context, query AdminUserQuery) (AdminUsersPage, error) {
	var out AdminUsersPage
	query.Search = strings.ToLower(strings.TrimSpace(query.Search))
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}
	base := s.db.WithContext(ctx).Table("users AS u").Where("u.role = ?", "user")
	if query.UserID != nil {
		base = base.Where("u.id = ?", *query.UserID)
	}
	if query.Search != "" {
		like := "%" + query.Search + "%"
		base = base.Where("LOWER(u.email) LIKE ? OR LOWER(u.display_name) LIKE ?", like, like)
	}
	if query.Plan != "" {
		if query.Plan == PlanFree {
			base = base.Where("NOT EXISTS (SELECT 1 FROM subscription_entitlements e WHERE e.user_id = u.id AND e.status = 'active' AND e.expires_at > ? AND e.revoked_at IS NULL)", time.Now().UTC())
		} else {
			base = base.Where("EXISTS (SELECT 1 FROM subscription_entitlements e WHERE e.user_id = u.id AND e.plan_code = ? AND e.status = 'active' AND e.expires_at > ? AND e.revoked_at IS NULL)", query.Plan, time.Now().UTC())
		}
	}
	if query.Platform == "ios" || query.Platform == "android" {
		base = base.Where("EXISTS (SELECT 1 FROM device_tokens d WHERE d.user_id = u.id AND d.platform = ?)", query.Platform)
	}
	if err := base.Count(&out.Total).Error; err != nil {
		return out, err
	}

	type row struct {
		ID                 uuid.UUID
		Email              string
		DisplayName        string
		AvatarURL          string
		Provider           string
		PlanCode           string
		PurchasePlatform   string
		HasIOS             int
		HasAndroid         int
		Favorites          int
		ActiveAlerts       int
		Badges             int
		AIResearchUses     int
		CreatedAt          time.Time
		LastActiveAt       *time.Time
		SubscriptionExpiry *time.Time
	}
	var rows []row
	selectSQL := `
		u.id, u.email, u.display_name, u.avatar_url, u.provider, u.created_at,
		COALESCE((SELECT e.plan_code FROM subscription_entitlements e WHERE e.user_id = u.id AND e.status = 'active' AND e.expires_at > CURRENT_TIMESTAMP AND e.revoked_at IS NULL ORDER BY e.updated_at DESC LIMIT 1), 'free') AS plan_code,
		COALESCE((SELECT e.provider FROM subscription_entitlements e WHERE e.user_id = u.id AND e.status = 'active' AND e.expires_at > CURRENT_TIMESTAMP AND e.revoked_at IS NULL ORDER BY e.updated_at DESC LIMIT 1), '') AS purchase_platform,
		(SELECT e.expires_at FROM subscription_entitlements e WHERE e.user_id = u.id AND e.status = 'active' AND e.expires_at > CURRENT_TIMESTAMP AND e.revoked_at IS NULL ORDER BY e.updated_at DESC LIMIT 1) AS subscription_expiry,
		COALESCE((SELECT MAX(CASE WHEN d.platform = 'ios' THEN 1 ELSE 0 END) FROM device_tokens d WHERE d.user_id = u.id), 0) AS has_ios,
		COALESCE((SELECT MAX(CASE WHEN d.platform = 'android' THEN 1 ELSE 0 END) FROM device_tokens d WHERE d.user_id = u.id), 0) AS has_android,
		(SELECT COUNT(*) FROM watchlist_items w WHERE w.user_id = u.id) AS favorites,
		(SELECT COUNT(*) FROM price_alerts a WHERE a.user_id = u.id AND a.active = true) AS active_alerts,
		(SELECT COUNT(*) FROM user_badges b WHERE b.user_id = u.id) AS badges,
		(SELECT COALESCE(SUM(f.quantity), 0) FROM feature_usages f WHERE f.user_id = u.id AND f.feature = 'ai_research') AS ai_research_uses,
		COALESCE((SELECT MAX(d.last_seen_at) FROM device_tokens d WHERE d.user_id = u.id), (SELECT MAX(r.created_at) FROM refresh_tokens r WHERE r.user_id = u.id), u.last_login_at, u.updated_at) AS last_active_at
	`
	if err := base.Select(selectSQL).Order("u.created_at DESC").
		Limit(query.PageSize).Offset((query.Page - 1) * query.PageSize).Scan(&rows).Error; err != nil {
		return out, err
	}
	onlineIDs := s.onlineIDSet(ctx)
	out.Items = make([]AdminUserSummary, 0, len(rows))
	for _, row := range rows {
		devicePlatform := "unknown"
		switch {
		case row.HasIOS == 1 && row.HasAndroid == 1:
			devicePlatform = "both"
		case row.HasIOS == 1:
			devicePlatform = "ios"
		case row.HasAndroid == 1:
			devicePlatform = "android"
		}
		out.Items = append(out.Items, AdminUserSummary{
			ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, AvatarURL: row.AvatarURL,
			Provider: row.Provider, PlanCode: row.PlanCode, PurchasePlatform: row.PurchasePlatform,
			DevicePlatform: devicePlatform, Favorites: row.Favorites, ActiveAlerts: row.ActiveAlerts,
			Badges: row.Badges, AIResearchUses: row.AIResearchUses, CreatedAt: row.CreatedAt,
			LastActiveAt: row.LastActiveAt, SubscriptionExpiry: row.SubscriptionExpiry,
			Online: onlineIDs[row.ID.String()],
		})
	}
	out.Page, out.PageSize = query.Page, query.PageSize
	return out, nil
}

type AdminActivityItem struct {
	Type      string    `json:"type"`
	Label     string    `json:"label"`
	Reference string    `json:"reference,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminDevice intentionally omits the APNs/FCM token. A device token is a
// delivery credential, not an operations-console display field.
type AdminDevice struct {
	ID          uuid.UUID `json:"id"`
	Platform    string    `json:"platform"`
	Locale      string    `json:"locale,omitempty"`
	AppVersion  string    `json:"app_version,omitempty"`
	DeviceModel string    `json:"device_model,omitempty"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AdminUserDetail struct {
	User         AdminUserSummary           `json:"user"`
	Devices      []AdminDevice              `json:"devices"`
	Favorites    []models.WatchlistItem     `json:"favorites"`
	Alerts       []models.PriceAlert        `json:"alerts"`
	Badges       []models.UserBadge         `json:"badges"`
	Gamification *models.UserGamification   `json:"gamification,omitempty"`
	DailyPlans   []models.DailyPlan         `json:"daily_plans"`
	Learning     []models.LearningProgress  `json:"learning_progress"`
	Reviews      []models.DisciplineReview  `json:"discipline_reviews"`
	FeatureUsage []models.FeatureUsage      `json:"feature_usage"`
	Portfolios   []models.Portfolio         `json:"portfolios"`
	Positions    []models.PortfolioPosition `json:"positions"`
	Activity     []AdminActivityItem        `json:"activity"`
}

func (s *AdminService) UserDetail(ctx context.Context, userID uuid.UUID) (AdminUserDetail, error) {
	var out AdminUserDetail
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ? AND role = ?", userID, "user").Take(&user).Error; err != nil {
		return out, err
	}
	page, err := s.Users(ctx, AdminUserQuery{UserID: &userID, Page: 1, PageSize: 1})
	if err != nil || len(page.Items) == 0 {
		return out, firstError(err, gorm.ErrRecordNotFound)
	}
	out.User = page.Items[0]
	if err := s.db.WithContext(ctx).Model(&models.DeviceToken{}).
		Select("id", "platform", "locale", "app_version", "device_model", "last_seen_at", "created_at", "updated_at").
		Where("user_id = ?", userID).Order("last_seen_at DESC").Scan(&out.Devices).Error; err != nil {
		return out, err
	}
	queries := []struct {
		dest  any
		order string
		limit int
	}{
		{&out.Favorites, "created_at DESC", 500},
		{&out.Alerts, "created_at DESC", 500},
		{&out.Badges, "earned_at DESC", 100},
		{&out.DailyPlans, "day_key DESC", 180},
		{&out.Learning, "updated_at DESC", 100},
		{&out.Reviews, "created_at DESC", 180},
		{&out.FeatureUsage, "created_at DESC", 250},
		{&out.Portfolios, "updated_at DESC", 100},
	}
	for _, query := range queries {
		if err := s.db.WithContext(ctx).Where("user_id = ?", userID).
			Order(query.order).Limit(query.limit).Find(query.dest).Error; err != nil {
			return out, err
		}
	}
	var gamification models.UserGamification
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Take(&gamification).Error; err == nil {
		out.Gamification = &gamification
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return out, err
	}
	if len(out.Portfolios) > 0 {
		ids := make([]uuid.UUID, 0, len(out.Portfolios))
		for _, portfolio := range out.Portfolios {
			ids = append(ids, portfolio.ID)
		}
		if err := s.db.WithContext(ctx).Where("portfolio_id IN ?", ids).
			Order("updated_at DESC").Limit(1000).Find(&out.Positions).Error; err != nil {
			return out, err
		}
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT type, label, reference, created_at FROM (
			SELECT 'favorite' AS type, 'Favoriye ekledi' AS label, code AS reference, created_at FROM watchlist_items WHERE user_id = ?
			UNION ALL
			SELECT 'alert' AS type, 'Fiyat alarmı oluşturdu' AS label, code AS reference, created_at FROM price_alerts WHERE user_id = ?
			UNION ALL
			SELECT 'badge' AS type, 'Rozet kazandı' AS label, code AS reference, earned_at AS created_at FROM user_badges WHERE user_id = ?
			UNION ALL
			SELECT 'gamification' AS type, type AS label, source_id AS reference, created_at FROM gamification_events WHERE user_id = ?
			UNION ALL
			SELECT 'feature' AS type, feature AS label, scope AS reference, created_at FROM feature_usages WHERE user_id = ?
			UNION ALL
			SELECT 'daily_plan' AS type, 'Günlük planını kaydetti' AS label, day_key AS reference, created_at FROM daily_plans WHERE user_id = ?
			UNION ALL
			SELECT 'learning' AS type, 'Öğrenme sorusunu yanıtladı' AS label, lesson_code AS reference, created_at FROM learning_attempts WHERE user_id = ?
			UNION ALL
			SELECT 'review' AS type, 'Karar günlüğünü güncelledi' AS label, code AS reference, created_at FROM discipline_reviews WHERE user_id = ?
			UNION ALL
			SELECT 'portfolio' AS type, 'Portföy işlemi kaydetti' AS label, code AS reference, created_at FROM portfolio_transactions WHERE user_id = ?
		) events ORDER BY created_at DESC LIMIT 100
	`, userID, userID, userID, userID, userID, userID, userID, userID, userID).Scan(&out.Activity).Error; err != nil {
		return out, err
	}
	return out, nil
}

type AdminOnlineUser struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Platform    string    `json:"platform"`
	Connections int       `json:"connections"`
	ConnectedAt time.Time `json:"connected_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

func (s *AdminService) OnlineUsers(ctx context.Context) ([]AdminOnlineUser, error) {
	if s.presence == nil {
		return []AdminOnlineUser{}, nil
	}
	entries, err := s.presence.Online(ctx)
	if err != nil || len(entries) == 0 {
		return []AdminOnlineUser{}, err
	}
	type grouped struct {
		connections int
		platforms   map[string]bool
		connected   time.Time
		lastSeen    time.Time
	}
	groups := make(map[string]*grouped)
	ids := make([]uuid.UUID, 0)
	for _, entry := range entries {
		id, parseErr := uuid.Parse(entry.UserID)
		if parseErr != nil {
			continue
		}
		group := groups[entry.UserID]
		if group == nil {
			group = &grouped{platforms: map[string]bool{}, connected: entry.ConnectedAt}
			groups[entry.UserID] = group
			ids = append(ids, id)
		}
		group.connections++
		group.platforms[entry.Platform] = true
		if group.connected.IsZero() || entry.ConnectedAt.Before(group.connected) {
			group.connected = entry.ConnectedAt
		}
		if entry.LastSeenAt.After(group.lastSeen) {
			group.lastSeen = entry.LastSeenAt
		}
	}
	var users []models.User
	if len(ids) > 0 {
		if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
			return nil, err
		}
	}
	out := make([]AdminOnlineUser, 0, len(users))
	for _, user := range users {
		group := groups[user.ID.String()]
		platforms := make([]string, 0, len(group.platforms))
		for platform := range group.platforms {
			platforms = append(platforms, platform)
		}
		sort.Strings(platforms)
		out = append(out, AdminOnlineUser{
			UserID: user.ID, Email: user.Email, DisplayName: user.DisplayName,
			Platform: strings.Join(platforms, "+"), Connections: group.connections,
			ConnectedAt: group.connected, LastSeenAt: group.lastSeen,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeenAt.After(out[j].LastSeenAt) })
	return out, nil
}

func (s *AdminService) onlineIDSet(ctx context.Context) map[string]bool {
	out := map[string]bool{}
	if s.presence == nil {
		return out
	}
	entries, err := s.presence.Online(ctx)
	if err != nil {
		return out
	}
	for _, entry := range entries {
		out[entry.UserID] = true
	}
	return out
}

func (s *AdminService) Audit(ctx context.Context, actorID *uuid.UUID, action, resourceType, resourceID, ip, userAgent string, metadata any) {
	if s == nil || s.db == nil {
		return
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		raw = []byte(`{"serialization_error":true}`)
	}
	entry := models.AdminAuditLog{
		ID: uuid.New(), ActorID: actorID, Action: action, ResourceType: resourceType,
		ResourceID: resourceID, Metadata: models.JSONDocument(raw),
		IPAddress: ip, UserAgent: truncate(userAgent, 320), CreatedAt: time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(&entry).Error; err != nil {
		s.log.Warn("admin audit persist failed", zap.Error(err), zap.String("action", action))
	}
}

type AdminAuditPage struct {
	Items    []models.AdminAuditLog `json:"items"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Total    int64                  `json:"total"`
}

func (s *AdminService) AuditLogs(ctx context.Context, page, pageSize int, action string) (AdminAuditPage, error) {
	var out AdminAuditPage
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 30
	}
	query := s.db.WithContext(ctx).Model(&models.AdminAuditLog{})
	if strings.TrimSpace(action) != "" {
		query = query.Where("action = ?", strings.TrimSpace(action))
	}
	if err := query.Count(&out.Total).Error; err != nil {
		return out, err
	}
	if err := query.Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&out.Items).Error; err != nil {
		return out, err
	}
	out.Page, out.PageSize = page, pageSize
	return out, nil
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func truncate(value string, maxLen int) string {
	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	return string(runes[:maxLen])
}
