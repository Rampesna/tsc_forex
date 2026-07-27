package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/models"
)

var (
	ErrInvalidTargetingPlan = errors.New("invalid targeting plan")
	ErrCampaignNotFound     = errors.New("campaign not found")
	ErrCampaignState        = errors.New("campaign is not in an approvable state")
	ErrAudienceChanged      = errors.New("campaign audience changed after preview")
)

type AdminPerformanceFilter struct {
	Direction       string  `json:"direction"`
	LookbackDays    int     `json:"lookback_days"`
	TopN            int     `json:"top_n"`
	MinAbsChangePct float64 `json:"min_abs_change_pct"`
}

// AdminTargetingPlan is the complete allow-listed targeting language. There is
// intentionally no raw SQL, column name, join or arbitrary predicate field.
type AdminTargetingPlan struct {
	Audience             string                  `json:"audience"`
	Platforms            []string                `json:"platforms"`
	PlanCodes            []string                `json:"plan_codes"`
	AssetCodes           []string                `json:"asset_codes"`
	AssetSources         []string                `json:"asset_sources"`
	Performance          *AdminPerformanceFilter `json:"performance"`
	RegisteredWithinDays int                     `json:"registered_within_days"`
	InactiveForDays      int                     `json:"inactive_for_days"`
	MaxRecipients        int                     `json:"max_recipients"`
}

type adminCampaignInterpretation struct {
	CampaignName string             `json:"campaign_name"`
	Title        string             `json:"title"`
	Body         string             `json:"body"`
	DeepLink     string             `json:"deep_link"`
	Summary      string             `json:"summary"`
	Plan         AdminTargetingPlan `json:"plan"`
}

type AdminCampaignAsset struct {
	Code      string  `json:"code"`
	ChangePct float64 `json:"change_pct"`
}

type AdminCampaignSampleUser struct {
	ID             uuid.UUID `json:"id"`
	Email          string    `json:"email"`
	DisplayName    string    `json:"display_name"`
	DevicePlatform string    `json:"device_platform"`
}

type AdminCampaignPreview struct {
	CampaignID          uuid.UUID                 `json:"campaign_id"`
	Name                string                    `json:"name"`
	Summary             string                    `json:"summary"`
	Title               string                    `json:"title"`
	Body                string                    `json:"body"`
	DeepLink            string                    `json:"deep_link"`
	Plan                AdminTargetingPlan        `json:"plan"`
	MatchedAssets       []AdminCampaignAsset      `json:"matched_assets"`
	RecipientCount      int                       `json:"recipient_count"`
	TotalMatched        int64                     `json:"total_matched"`
	SampleUsers         []AdminCampaignSampleUser `json:"sample_users"`
	ApprovalFingerprint string                    `json:"approval_fingerprint"`
}

type adminResolvedAudience struct {
	Assets       []AdminCampaignAsset
	Users        []AdminCampaignSampleUser
	TotalMatched int64
}

type storedAdminCampaignPreview struct {
	Summary       string                    `json:"summary"`
	MatchedAssets []AdminCampaignAsset      `json:"matched_assets"`
	SampleUsers   []AdminCampaignSampleUser `json:"sample_users"`
	TotalMatched  int64                     `json:"total_matched"`
}

func (s *AdminService) PreviewCampaign(ctx context.Context, managerID uuid.UUID, prompt string) (AdminCampaignPreview, error) {
	var out AdminCampaignPreview
	prompt = strings.TrimSpace(prompt)
	if len([]rune(prompt)) < 8 || len([]rune(prompt)) > 2000 {
		return out, fmt.Errorf("%w: prompt length", ErrInvalidTargetingPlan)
	}
	interpretation, err := s.interpretCampaignPrompt(ctx, prompt)
	if err != nil {
		return out, err
	}
	if err := s.validateTargetingPlan(&interpretation.Plan); err != nil {
		return out, err
	}
	if err := validateCampaignContent(interpretation.Title, interpretation.Body, interpretation.DeepLink); err != nil {
		return out, err
	}
	audience, err := s.resolveAudience(ctx, interpretation.Plan)
	if err != nil {
		return out, err
	}
	campaignID := uuid.New()
	fingerprint := campaignFingerprint(campaignID, interpretation.Plan, interpretation.Title, interpretation.Body, len(audience.Users))
	out = AdminCampaignPreview{
		CampaignID: campaignID, Name: interpretation.CampaignName,
		Summary: interpretation.Summary, Title: interpretation.Title,
		Body: interpretation.Body, DeepLink: interpretation.DeepLink,
		Plan: interpretation.Plan, MatchedAssets: audience.Assets,
		RecipientCount: len(audience.Users), TotalMatched: audience.TotalMatched,
		ApprovalFingerprint: fingerprint,
	}
	if len(audience.Users) > 10 {
		out.SampleUsers = audience.Users[:10]
	} else {
		out.SampleUsers = audience.Users
	}
	normalizeAdminCampaignPreview(&out)
	planJSON, _ := json.Marshal(interpretation.Plan)
	previewJSON, _ := json.Marshal(map[string]any{
		"summary": interpretation.Summary, "matched_assets": audience.Assets,
		"sample_users": out.SampleUsers, "total_matched": audience.TotalMatched,
	})
	campaign := models.NotificationCampaign{
		ID: campaignID, ManagerID: managerID, Name: truncate(interpretation.CampaignName, 120),
		Prompt: prompt, Status: "draft", TargetingPlan: models.JSONDocument(planJSON),
		Preview: models.JSONDocument(previewJSON), Title: interpretation.Title,
		Body: interpretation.Body, DeepLink: interpretation.DeepLink,
		EstimatedRecipients: len(audience.Users), ApprovalFingerprint: fingerprint,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(&campaign).Error; err != nil {
		return AdminCampaignPreview{}, err
	}
	return out, nil
}

// RefreshCampaignPreview reopens an existing draft without creating a second
// campaign. The stored allow-listed plan is resolved again so the manager sees
// the current audience and receives a fresh approval fingerprint.
func (s *AdminService) RefreshCampaignPreview(ctx context.Context, managerID, campaignID uuid.UUID) (AdminCampaignPreview, error) {
	var out AdminCampaignPreview
	var campaign models.NotificationCampaign
	if err := s.db.WithContext(ctx).
		Where("id = ? AND manager_id = ?", campaignID, managerID).
		Take(&campaign).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return out, ErrCampaignNotFound
		}
		return out, err
	}
	if campaign.Status != "draft" {
		return out, ErrCampaignState
	}

	var plan AdminTargetingPlan
	if err := json.Unmarshal(campaign.TargetingPlan, &plan); err != nil {
		return out, fmt.Errorf("%w: stored plan", ErrInvalidTargetingPlan)
	}
	if err := s.validateTargetingPlan(&plan); err != nil {
		return out, err
	}
	if err := validateCampaignContent(campaign.Title, campaign.Body, campaign.DeepLink); err != nil {
		return out, err
	}

	audience, err := s.resolveAudience(ctx, plan)
	if err != nil {
		return out, err
	}
	storedPreview := storedAdminCampaignPreview{}
	if len(campaign.Preview) > 0 {
		_ = json.Unmarshal(campaign.Preview, &storedPreview)
	}

	fingerprint := campaignFingerprint(campaign.ID, plan, campaign.Title, campaign.Body, len(audience.Users))
	out = AdminCampaignPreview{
		CampaignID:          campaign.ID,
		Name:                campaign.Name,
		Summary:             storedPreview.Summary,
		Title:               campaign.Title,
		Body:                campaign.Body,
		DeepLink:            campaign.DeepLink,
		Plan:                plan,
		MatchedAssets:       audience.Assets,
		RecipientCount:      len(audience.Users),
		TotalMatched:        audience.TotalMatched,
		ApprovalFingerprint: fingerprint,
	}
	if len(audience.Users) > 10 {
		out.SampleUsers = audience.Users[:10]
	} else {
		out.SampleUsers = audience.Users
	}
	normalizeAdminCampaignPreview(&out)

	planJSON, _ := json.Marshal(plan)
	previewJSON, _ := json.Marshal(storedAdminCampaignPreview{
		Summary:       storedPreview.Summary,
		MatchedAssets: audience.Assets,
		SampleUsers:   out.SampleUsers,
		TotalMatched:  audience.TotalMatched,
	})
	result := s.db.WithContext(ctx).Model(&models.NotificationCampaign{}).
		Where("id = ? AND manager_id = ? AND status = ?", campaign.ID, managerID, "draft").
		Updates(map[string]any{
			"targeting_plan":       models.JSONDocument(planJSON),
			"preview":              models.JSONDocument(previewJSON),
			"estimated_recipients": len(audience.Users),
			"approval_fingerprint": fingerprint,
			"updated_at":           time.Now().UTC(),
		})
	if result.Error != nil {
		return AdminCampaignPreview{}, result.Error
	}
	if result.RowsAffected != 1 {
		return AdminCampaignPreview{}, ErrCampaignState
	}
	return out, nil
}

func (s *AdminService) interpretCampaignPrompt(ctx context.Context, prompt string) (adminCampaignInterpretation, error) {
	if s.aiCfg.Enabled && strings.TrimSpace(s.aiCfg.APIKey) != "" {
		if interpreted, err := s.interpretCampaignWithAI(ctx, prompt); err == nil {
			return interpreted, nil
		} else {
			s.log.Warn("admin AI targeting interpretation failed; using deterministic fallback")
		}
	}
	return deterministicCampaignInterpretation(prompt), nil
}

func (s *AdminService) interpretCampaignWithAI(ctx context.Context, prompt string) (adminCampaignInterpretation, error) {
	model := strings.TrimSpace(s.aiCfg.Model)
	if model == "" {
		model = "gpt-5.6-sol"
	}
	maxRecipients := s.adminCfg.MaxRecipients
	if maxRecipients <= 0 {
		maxRecipients = 100000
	}
	schema := map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"campaign_name": map[string]any{"type": "string"},
			"title":         map[string]any{"type": "string"},
			"body":          map[string]any{"type": "string"},
			"deep_link":     map[string]any{"type": "string"},
			"summary":       map[string]any{"type": "string"},
			"plan": map[string]any{
				"type": "object", "additionalProperties": false,
				"properties": map[string]any{
					"audience":      map[string]any{"type": "string", "enum": []string{"all_users", "asset_followers", "subscribers", "inactive_users"}},
					"platforms":     map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []string{"ios", "android"}}, "maxItems": 2},
					"plan_codes":    map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []string{PlanFree, PlanPlus, PlanPro, PlanExpert, PlanElite}}, "maxItems": 5},
					"asset_codes":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "maxItems": 50},
					"asset_sources": map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []string{"watchlist", "alerts", "portfolio"}}, "maxItems": 3},
					"performance": map[string]any{
						"anyOf": []any{
							map[string]any{"type": "null"},
							map[string]any{
								"type": "object", "additionalProperties": false,
								"properties": map[string]any{
									"direction":          map[string]any{"type": "string", "enum": []string{"decliners", "gainers"}},
									"lookback_days":      map[string]any{"type": "integer", "minimum": 1, "maximum": 365},
									"top_n":              map[string]any{"type": "integer", "minimum": 1, "maximum": 50},
									"min_abs_change_pct": map[string]any{"type": "number", "minimum": 0, "maximum": 100},
								},
								"required": []string{"direction", "lookback_days", "top_n", "min_abs_change_pct"},
							},
						},
					},
					"registered_within_days": map[string]any{"type": "integer", "minimum": 0, "maximum": 3650},
					"inactive_for_days":      map[string]any{"type": "integer", "minimum": 0, "maximum": 3650},
					"max_recipients":         map[string]any{"type": "integer", "minimum": 1, "maximum": maxRecipients},
				},
				"required": []string{"audience", "platforms", "plan_codes", "asset_codes", "asset_sources", "performance", "registered_within_days", "inactive_for_days", "max_recipients"},
			},
		},
		"required": []string{"campaign_name", "title", "body", "deep_link", "summary", "plan"},
	}
	payload := map[string]any{
		"model":        model,
		"instructions": "TSC Forex admin bildirim hedefleme planlayıcısısın. Kullanıcının Türkçe veya başka dildeki talebini yalnızca verilen JSON şemasına çevir. Asla SQL, tablo/kolon adı, kod veya serbest sorgu üretme. Belirsiz durumda dar ve güvenli hedef seç; planı genişletme. Bildirim kısa, sakin, yatırım tavsiyesi veya al/sat emri içermeyen Türkçe metin olsun. title en fazla 60, body en fazla 220 karakter olsun. deep_link yalnızca boş veya tscforex:// ile başlayan uygulama bağlantısı olsun.",
		"input":        prompt, "max_output_tokens": 1500, "store": false,
		"reasoning": map[string]any{"effort": "medium"},
		"text": map[string]any{"format": map[string]any{
			"type": "json_schema", "name": "admin_notification_targeting",
			"strict": true, "schema": schema,
		}},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return adminCampaignInterpretation{}, err
	}
	baseURL := strings.TrimRight(strings.TrimSpace(s.aiCfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	timeout := s.aiCfg.Timeout
	if timeout <= 0 || timeout > 60*time.Second {
		timeout = 45 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, baseURL+"/responses", bytes.NewReader(raw))
	if err != nil {
		return adminCampaignInterpretation{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.aiCfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return adminCampaignInterpretation{}, err
	}
	defer res.Body.Close()
	body := io.LimitReader(res.Body, 1<<20)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, body)
		return adminCampaignInterpretation{}, fmt.Errorf("openai status %d", res.StatusCode)
	}
	var envelope struct {
		Status string `json:"status"`
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.NewDecoder(body).Decode(&envelope); err != nil {
		return adminCampaignInterpretation{}, err
	}
	if envelope.Status != "" && envelope.Status != "completed" {
		return adminCampaignInterpretation{}, fmt.Errorf("openai status %s", envelope.Status)
	}
	for _, item := range envelope.Output {
		for _, content := range item.Content {
			if content.Type == "output_text" && content.Text != "" {
				var interpreted adminCampaignInterpretation
				if err := json.Unmarshal([]byte(content.Text), &interpreted); err != nil {
					return adminCampaignInterpretation{}, err
				}
				return interpreted, nil
			}
		}
	}
	return adminCampaignInterpretation{}, errors.New("missing AI targeting output")
}

func deterministicCampaignInterpretation(prompt string) adminCampaignInterpretation {
	lower := strings.ToLower(prompt)
	plan := AdminTargetingPlan{
		Audience: "all_users", AssetSources: []string{"watchlist"},
		MaxRecipients: 100000,
	}
	if strings.Contains(lower, "favori") || strings.Contains(lower, "varlık") || strings.Contains(lower, "sahip") {
		plan.Audience = "asset_followers"
	}
	if strings.Contains(lower, "alarm") {
		plan.AssetSources = append(plan.AssetSources, "alerts")
	}
	if strings.Contains(lower, "portföy") {
		plan.AssetSources = append(plan.AssetSources, "portfolio")
	}
	if strings.Contains(lower, "ios") || strings.Contains(lower, "apple") {
		plan.Platforms = append(plan.Platforms, "ios")
	}
	if strings.Contains(lower, "android") {
		plan.Platforms = append(plan.Platforms, "android")
	}
	for _, item := range []struct{ needle, code string }{
		{"ücretsiz", PlanFree}, {"başlangıç", PlanPlus}, {"plus", PlanPlus},
		{"pro", PlanPro}, {"uzman", PlanExpert}, {"expert", PlanExpert}, {"elite", PlanElite},
	} {
		if strings.Contains(lower, item.needle) {
			plan.PlanCodes = append(plan.PlanCodes, item.code)
		}
	}
	if strings.Contains(lower, "düş") || strings.Contains(lower, "gerile") || strings.Contains(lower, "yüks") || strings.Contains(lower, "art") {
		direction := "decliners"
		if strings.Contains(lower, "yüks") || strings.Contains(lower, "art") {
			direction = "gainers"
		}
		days := 1
		switch {
		case strings.Contains(lower, "hafta"):
			days = 7
		case strings.Contains(lower, "ay"):
			days = 30
		}
		plan.Performance = &AdminPerformanceFilter{Direction: direction, LookbackDays: days, TopN: 5}
		plan.Audience = "asset_followers"
	}
	title := "Piyasa hareketi uyarısı"
	body := "Takip ettiğiniz varlıklarda dikkat çeken bir hareket var. Güncel verileri ve risklerinizi gözden geçirmenizi öneririz."
	return adminCampaignInterpretation{
		CampaignName: "Akıllı hedefleme", Title: title, Body: body,
		DeepLink: "tscforex://markets", Summary: "Talep, güvenli hedefleme kurallarıyla yorumlandı.",
		Plan: plan,
	}
}

func (s *AdminService) validateTargetingPlan(plan *AdminTargetingPlan) error {
	if plan == nil {
		return fmt.Errorf("%w: missing plan", ErrInvalidTargetingPlan)
	}
	validAudience := map[string]bool{"all_users": true, "asset_followers": true, "subscribers": true, "inactive_users": true}
	if !validAudience[plan.Audience] {
		return fmt.Errorf("%w: audience", ErrInvalidTargetingPlan)
	}
	plan.Platforms = uniqueAllowed(plan.Platforms, map[string]bool{"ios": true, "android": true})
	plan.PlanCodes = uniqueAllowed(plan.PlanCodes, map[string]bool{PlanFree: true, PlanPlus: true, PlanPro: true, PlanExpert: true, PlanElite: true})
	plan.AssetSources = uniqueAllowed(plan.AssetSources, map[string]bool{"watchlist": true, "alerts": true, "portfolio": true})
	if len(plan.AssetSources) == 0 && plan.Audience == "asset_followers" {
		plan.AssetSources = []string{"watchlist"}
	}
	codePattern := regexp.MustCompile(`^[A-Z0-9_./-]{1,32}$`)
	codes := make([]string, 0, len(plan.AssetCodes))
	seen := map[string]bool{}
	for _, code := range plan.AssetCodes {
		code = strings.ToUpper(strings.TrimSpace(code))
		if codePattern.MatchString(code) && !seen[code] {
			seen[code] = true
			codes = append(codes, code)
		}
	}
	plan.AssetCodes = codes
	if plan.Performance != nil {
		if plan.Performance.Direction != "decliners" && plan.Performance.Direction != "gainers" {
			return fmt.Errorf("%w: performance direction", ErrInvalidTargetingPlan)
		}
		if plan.Performance.LookbackDays < 1 || plan.Performance.LookbackDays > 365 ||
			plan.Performance.TopN < 1 || plan.Performance.TopN > 50 ||
			plan.Performance.MinAbsChangePct < 0 || plan.Performance.MinAbsChangePct > 100 {
			return fmt.Errorf("%w: performance bounds", ErrInvalidTargetingPlan)
		}
	}
	if plan.Audience == "asset_followers" && len(plan.AssetCodes) == 0 && plan.Performance == nil {
		return fmt.Errorf("%w: asset audience requires codes or performance", ErrInvalidTargetingPlan)
	}
	if plan.RegisteredWithinDays < 0 || plan.RegisteredWithinDays > 3650 ||
		plan.InactiveForDays < 0 || plan.InactiveForDays > 3650 {
		return fmt.Errorf("%w: day filter", ErrInvalidTargetingPlan)
	}
	maxRecipients := s.adminCfg.MaxRecipients
	if maxRecipients <= 0 {
		maxRecipients = 100000
	}
	if plan.MaxRecipients <= 0 {
		plan.MaxRecipients = min(1000, maxRecipients)
	}
	if plan.MaxRecipients > maxRecipients {
		plan.MaxRecipients = maxRecipients
	}
	return nil
}

func uniqueAllowed(values []string, allowed map[string]bool) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if allowed[value] && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func normalizeAdminCampaignPreview(preview *AdminCampaignPreview) {
	if preview == nil {
		return
	}
	if preview.Plan.Platforms == nil {
		preview.Plan.Platforms = []string{}
	}
	if preview.Plan.PlanCodes == nil {
		preview.Plan.PlanCodes = []string{}
	}
	if preview.Plan.AssetCodes == nil {
		preview.Plan.AssetCodes = []string{}
	}
	if preview.Plan.AssetSources == nil {
		preview.Plan.AssetSources = []string{}
	}
	if preview.MatchedAssets == nil {
		preview.MatchedAssets = []AdminCampaignAsset{}
	}
	if preview.SampleUsers == nil {
		preview.SampleUsers = []AdminCampaignSampleUser{}
	}
}

func validateCampaignContent(title, body, deepLink string) error {
	if len([]rune(strings.TrimSpace(title))) < 2 || len([]rune(title)) > 128 ||
		len([]rune(strings.TrimSpace(body))) < 4 || len([]rune(body)) > 512 {
		return fmt.Errorf("%w: notification content", ErrInvalidTargetingPlan)
	}
	if deepLink != "" && !strings.HasPrefix(deepLink, "tscforex://") {
		return fmt.Errorf("%w: deep link", ErrInvalidTargetingPlan)
	}
	return nil
}

func (s *AdminService) resolveAudience(ctx context.Context, plan AdminTargetingPlan) (adminResolvedAudience, error) {
	out := adminResolvedAudience{
		Assets: []AdminCampaignAsset{},
		Users:  []AdminCampaignSampleUser{},
	}
	assets := append([]string{}, plan.AssetCodes...)
	if plan.Performance != nil {
		performanceAssets, err := s.resolvePerformanceAssets(ctx, *plan.Performance)
		if err != nil {
			return out, err
		}
		out.Assets = performanceAssets
		assets = assets[:0]
		for _, asset := range performanceAssets {
			assets = append(assets, asset.Code)
		}
	} else {
		for _, code := range assets {
			out.Assets = append(out.Assets, AdminCampaignAsset{Code: code})
		}
	}

	query := s.db.WithContext(ctx).Table("users AS u").Where("u.role = ?", "user")
	if len(plan.Platforms) > 0 {
		query = query.Where(`
			EXISTS (SELECT 1 FROM device_tokens d WHERE d.user_id = u.id AND d.platform IN ?)
			OR EXISTS (SELECT 1 FROM refresh_tokens r WHERE r.user_id = u.id AND r.platform IN ?)
		`, plan.Platforms, plan.Platforms)
	}
	if len(plan.PlanCodes) > 0 {
		hasFree := false
		paid := make([]string, 0)
		for _, code := range plan.PlanCodes {
			if code == PlanFree {
				hasFree = true
			} else {
				paid = append(paid, code)
			}
		}
		now := time.Now().UTC()
		switch {
		case hasFree && len(paid) > 0:
			query = query.Where(`(
				NOT EXISTS (SELECT 1 FROM subscription_entitlements e WHERE e.user_id = u.id AND e.status = 'active' AND e.expires_at > ? AND e.revoked_at IS NULL)
				OR EXISTS (SELECT 1 FROM subscription_entitlements e WHERE e.user_id = u.id AND e.plan_code IN ? AND e.status = 'active' AND e.expires_at > ? AND e.revoked_at IS NULL)
			)`, now, paid, now)
		case hasFree:
			query = query.Where("NOT EXISTS (SELECT 1 FROM subscription_entitlements e WHERE e.user_id = u.id AND e.status = 'active' AND e.expires_at > ? AND e.revoked_at IS NULL)", now)
		case len(paid) > 0:
			query = query.Where("EXISTS (SELECT 1 FROM subscription_entitlements e WHERE e.user_id = u.id AND e.plan_code IN ? AND e.status = 'active' AND e.expires_at > ? AND e.revoked_at IS NULL)", paid, now)
		}
	} else if plan.Audience == "subscribers" {
		query = query.Where("EXISTS (SELECT 1 FROM subscription_entitlements e WHERE e.user_id = u.id AND e.status = 'active' AND e.expires_at > ? AND e.revoked_at IS NULL)", time.Now().UTC())
	}
	if len(assets) > 0 {
		clauses := make([]string, 0, len(plan.AssetSources))
		args := make([]any, 0, len(plan.AssetSources))
		for _, source := range plan.AssetSources {
			switch source {
			case "watchlist":
				clauses = append(clauses, "EXISTS (SELECT 1 FROM watchlist_items w WHERE w.user_id = u.id AND w.code IN ?)")
				args = append(args, assets)
			case "alerts":
				clauses = append(clauses, "EXISTS (SELECT 1 FROM price_alerts a WHERE a.user_id = u.id AND a.code IN ?)")
				args = append(args, assets)
			case "portfolio":
				clauses = append(clauses, "EXISTS (SELECT 1 FROM portfolios p JOIN portfolio_positions pp ON pp.portfolio_id = p.id WHERE p.user_id = u.id AND pp.code IN ?)")
				args = append(args, assets)
			}
		}
		if len(clauses) > 0 {
			query = query.Where("("+strings.Join(clauses, " OR ")+")", args...)
		}
	}
	if plan.RegisteredWithinDays > 0 {
		query = query.Where("u.created_at >= ?", time.Now().UTC().AddDate(0, 0, -plan.RegisteredWithinDays))
	}
	if plan.InactiveForDays > 0 || plan.Audience == "inactive_users" {
		days := plan.InactiveForDays
		if days <= 0 {
			days = 30
		}
		cutoff := time.Now().UTC().AddDate(0, 0, -days)
		query = query.Where(`COALESCE(
			(SELECT MAX(d.last_seen_at) FROM device_tokens d WHERE d.user_id = u.id),
			(SELECT MAX(r.created_at) FROM refresh_tokens r WHERE r.user_id = u.id),
			u.last_login_at, u.updated_at, u.created_at
		) < ?`, cutoff)
	}
	if err := query.Count(&out.TotalMatched).Error; err != nil {
		return out, err
	}
	type row struct {
		ID          uuid.UUID
		Email       string
		DisplayName string
		HasIOS      int
		HasAndroid  int
	}
	var rows []row
	if err := query.Select(`
		u.id, u.email, u.display_name,
		CASE WHEN
			EXISTS (SELECT 1 FROM device_tokens d WHERE d.user_id = u.id AND d.platform = 'ios')
			OR EXISTS (SELECT 1 FROM refresh_tokens r WHERE r.user_id = u.id AND r.platform = 'ios')
		THEN 1 ELSE 0 END AS has_ios,
		CASE WHEN
			EXISTS (SELECT 1 FROM device_tokens d WHERE d.user_id = u.id AND d.platform = 'android')
			OR EXISTS (SELECT 1 FROM refresh_tokens r WHERE r.user_id = u.id AND r.platform = 'android')
		THEN 1 ELSE 0 END AS has_android
	`).Order("u.id ASC").Limit(plan.MaxRecipients).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		platform := "unknown"
		switch {
		case row.HasIOS == 1 && row.HasAndroid == 1:
			platform = "both"
		case row.HasIOS == 1:
			platform = "ios"
		case row.HasAndroid == 1:
			platform = "android"
		}
		out.Users = append(out.Users, AdminCampaignSampleUser{
			ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, DevicePlatform: platform,
		})
	}
	return out, nil
}

func (s *AdminService) resolvePerformanceAssets(ctx context.Context, filter AdminPerformanceFilter) ([]AdminCampaignAsset, error) {
	type candleRow struct {
		Code   string
		Bucket time.Time
		Close  float64
	}
	var rows []candleRow
	from := time.Now().UTC().AddDate(0, 0, -filter.LookbackDays-2)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT code, bucket, close FROM candles
		WHERE resolution = '1d' AND bucket >= ? AND close > 0
		ORDER BY code ASC, bucket ASC
	`, from).Scan(&rows).Error; err != nil {
		return nil, err
	}
	type bounds struct{ first, last float64 }
	byCode := map[string]bounds{}
	for _, row := range rows {
		value := byCode[row.Code]
		if value.first == 0 {
			value.first = row.Close
		}
		value.last = row.Close
		byCode[row.Code] = value
	}
	out := make([]AdminCampaignAsset, 0, len(byCode))
	for code, value := range byCode {
		if value.first <= 0 || value.last <= 0 || value.first == value.last {
			continue
		}
		change := (value.last - value.first) / value.first * 100
		if filter.MinAbsChangePct > 0 && mathAbs(change) < filter.MinAbsChangePct {
			continue
		}
		if filter.Direction == "decliners" && change >= 0 {
			continue
		}
		if filter.Direction == "gainers" && change <= 0 {
			continue
		}
		out = append(out, AdminCampaignAsset{Code: code, ChangePct: mathRound(change, 4)})
	}
	sort.Slice(out, func(i, j int) bool {
		if filter.Direction == "decliners" {
			return out[i].ChangePct < out[j].ChangePct
		}
		return out[i].ChangePct > out[j].ChangePct
	})
	if len(out) > filter.TopN {
		out = out[:filter.TopN]
	}
	return out, nil
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func mathRound(v float64, precision int) float64 {
	pow := 1.0
	for i := 0; i < precision; i++ {
		pow *= 10
	}
	if v >= 0 {
		return float64(int64(v*pow+0.5)) / pow
	}
	return float64(int64(v*pow-0.5)) / pow
}

func campaignFingerprint(id uuid.UUID, plan AdminTargetingPlan, title, body string, count int) string {
	raw, _ := json.Marshal(plan)
	sum := sha256.Sum256([]byte(id.String() + "\x00" + string(raw) + "\x00" + title + "\x00" + body + fmt.Sprintf("\x00%d", count)))
	return hex.EncodeToString(sum[:])
}

type AdminCampaignApproval struct {
	ExpectedRecipients int    `json:"expected_recipients"`
	Fingerprint        string `json:"fingerprint"`
	Title              string `json:"title"`
	Body               string `json:"body"`
	DeepLink           string `json:"deep_link"`
}

func (s *AdminService) ApproveCampaign(ctx context.Context, managerID, campaignID uuid.UUID, approval AdminCampaignApproval) (models.NotificationCampaign, error) {
	var campaign models.NotificationCampaign
	if err := s.db.WithContext(ctx).Where("id = ? AND manager_id = ?", campaignID, managerID).Take(&campaign).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return campaign, ErrCampaignNotFound
		}
		return campaign, err
	}
	if campaign.Status != "draft" || approval.Fingerprint == "" || approval.Fingerprint != campaign.ApprovalFingerprint {
		return campaign, ErrCampaignState
	}
	if approval.Title == "" {
		approval.Title = campaign.Title
	}
	if approval.Body == "" {
		approval.Body = campaign.Body
	}
	if approval.DeepLink == "" {
		approval.DeepLink = campaign.DeepLink
	}
	if err := validateCampaignContent(approval.Title, approval.Body, approval.DeepLink); err != nil {
		return campaign, err
	}
	var plan AdminTargetingPlan
	if err := json.Unmarshal(campaign.TargetingPlan, &plan); err != nil {
		return campaign, err
	}
	if err := s.validateTargetingPlan(&plan); err != nil {
		return campaign, err
	}
	audience, err := s.resolveAudience(ctx, plan)
	if err != nil {
		return campaign, err
	}
	if approval.ExpectedRecipients != len(audience.Users) || campaign.EstimatedRecipients != len(audience.Users) {
		return campaign, ErrAudienceChanged
	}
	now := time.Now().UTC()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked models.NotificationCampaign
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", campaignID).Take(&locked).Error; err != nil {
			return err
		}
		if locked.Status != "draft" {
			return ErrCampaignState
		}
		recipients := make([]models.NotificationCampaignRecipient, 0, len(audience.Users))
		for _, user := range audience.Users {
			recipients = append(recipients, models.NotificationCampaignRecipient{
				ID: uuid.New(), CampaignID: campaignID, UserID: user.ID,
				PlatformSummary: user.DevicePlatform, Status: "pending",
				CreatedAt: now, UpdatedAt: now,
			})
		}
		if len(recipients) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(recipients, 500).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.NotificationCampaign{}).Where("id = ?", campaignID).Updates(map[string]any{
			"status": "queued", "title": approval.Title, "body": approval.Body,
			"deep_link": approval.DeepLink, "resolved_recipients": len(recipients),
			"approved_at": now, "updated_at": now,
		}).Error
	})
	if err != nil {
		return campaign, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", campaignID).Take(&campaign).Error; err != nil {
		return campaign, err
	}
	return campaign, nil
}

func (s *AdminService) Campaigns(ctx context.Context, page, pageSize int) ([]models.NotificationCampaign, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	query := s.db.WithContext(ctx).Model(&models.NotificationCampaign{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]models.NotificationCampaign, 0)
	err := query.Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error
	return rows, total, err
}

func (s *AdminService) CancelCampaign(ctx context.Context, managerID, campaignID uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var campaign models.NotificationCampaign
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND manager_id = ?", campaignID, managerID).Take(&campaign).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCampaignNotFound
			}
			return err
		}
		if campaign.Status != "draft" && campaign.Status != "queued" && campaign.Status != "running" {
			return ErrCampaignState
		}
		now := time.Now().UTC()
		if err := tx.Model(&models.NotificationCampaignRecipient{}).
			Where("campaign_id = ? AND status IN ?", campaignID, []string{"pending", "processing"}).
			Updates(map[string]any{"status": "cancelled", "processed_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&models.NotificationCampaign{}).Where("id = ?", campaignID).
			Updates(map[string]any{"status": "cancelled", "completed_at": now, "updated_at": now}).Error
	})
}

func (s *AdminService) StartCampaignWorker(ctx context.Context) {
	if s == nil || s.db == nil || !s.adminCfg.Enabled {
		return
	}
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				campaign, ok, err := s.claimCampaign(ctx)
				if err != nil {
					s.log.Warn("campaign claim failed")
					continue
				}
				if ok {
					s.processCampaign(ctx, campaign)
				}
			}
		}
	}()
}

func (s *AdminService) claimCampaign(ctx context.Context) (models.NotificationCampaign, bool, error) {
	var campaign models.NotificationCampaign
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		stale := now.Add(-2 * time.Minute)
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? OR (status = ? AND updated_at < ?)", "queued", "running", stale).
			Order("approved_at ASC").Take(&campaign).Error; err != nil {
			return err
		}
		updates := map[string]any{"status": "running", "updated_at": now}
		if campaign.StartedAt == nil {
			updates["started_at"] = now
			campaign.StartedAt = &now
		}
		campaign.Status = "running"
		campaign.UpdatedAt = now
		return tx.Model(&models.NotificationCampaign{}).Where("id = ?", campaign.ID).
			Updates(updates).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return campaign, false, nil
	}
	return campaign, err == nil, err
}

func (s *AdminService) processCampaign(ctx context.Context, campaign models.NotificationCampaign) {
	batchSize := s.adminCfg.CampaignBatchSize
	if batchSize <= 0 || batchSize > 500 {
		batchSize = 100
	}
	sendRate := s.adminCfg.SendPerSecond
	if sendRate <= 0 {
		sendRate = 20
	}
	interval := time.Second / time.Duration(sendRate)
	if interval < 10*time.Millisecond {
		interval = 10 * time.Millisecond
	}
	rate := time.NewTicker(interval)
	defer rate.Stop()
	var plan AdminTargetingPlan
	if err := json.Unmarshal(campaign.TargetingPlan, &plan); err != nil {
		s.failCampaign(ctx, campaign.ID, "invalid targeting plan")
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		var state string
		if err := s.db.WithContext(ctx).Model(&models.NotificationCampaign{}).
			Select("status").Where("id = ?", campaign.ID).Row().Scan(&state); err != nil || state != "running" {
			return
		}
		recipients, err := s.claimCampaignRecipients(ctx, campaign.ID, batchSize)
		if err != nil {
			s.log.Warn("campaign recipient fetch failed")
			return
		}
		if len(recipients) == 0 {
			var remaining int64
			if err := s.db.WithContext(ctx).Model(&models.NotificationCampaignRecipient{}).
				Where("campaign_id = ? AND status IN ?", campaign.ID, []string{"pending", "processing"}).
				Count(&remaining).Error; err != nil {
				return
			}
			if remaining == 0 {
				s.finalizeCampaign(ctx, campaign.ID)
				return
			}
			_ = s.db.WithContext(ctx).Model(&models.NotificationCampaign{}).Where("id = ?", campaign.ID).
				Update("updated_at", time.Now().UTC()).Error
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
				continue
			}
		}
		processedCount, failedCount := 0, 0
		for _, recipient := range recipients {
			select {
			case <-ctx.Done():
				return
			case <-rate.C:
			}
			var stillRunning bool
			if err := s.db.WithContext(ctx).Raw(
				"SELECT EXISTS(SELECT 1 FROM notification_campaigns WHERE id = ? AND status = 'running')",
				campaign.ID,
			).Scan(&stillRunning).Error; err != nil || !stillRunning {
				return
			}
			err := errors.New("notification service unavailable")
			if s.notifier != nil {
				err = s.notifier.NotifyCampaignWithResult(
					ctx, recipient.ID, recipient.UserID,
					campaign.Title, campaign.Body, campaign.DeepLink, plan.Platforms,
				)
			}
			now := time.Now().UTC()
			updates := map[string]any{"updated_at": now}
			if err != nil {
				if recipient.Attempts >= 3 {
					updates["status"] = "failed"
					updates["processed_at"] = now
				} else {
					updates["status"] = "pending"
				}
				updates["last_error"] = truncate(err.Error(), 512)
			} else {
				updates["status"] = "processed"
				updates["processed_at"] = now
				updates["last_error"] = ""
			}
			result := s.db.WithContext(ctx).Model(&models.NotificationCampaignRecipient{}).
				Where("id = ? AND status = ?", recipient.ID, "processing").Updates(updates)
			if result.Error == nil && result.RowsAffected == 1 {
				if updates["status"] == "processed" {
					processedCount++
				} else if updates["status"] == "failed" {
					failedCount++
				}
			}
		}
		campaignUpdates := map[string]any{"updated_at": time.Now().UTC()}
		if processedCount > 0 {
			campaignUpdates["processed_recipients"] = gorm.Expr("processed_recipients + ?", processedCount)
		}
		if failedCount > 0 {
			campaignUpdates["failed_recipients"] = gorm.Expr("failed_recipients + ?", failedCount)
		}
		_ = s.db.WithContext(ctx).Model(&models.NotificationCampaign{}).
			Where("id = ? AND status = ?", campaign.ID, "running").Updates(campaignUpdates).Error
	}
}

func (s *AdminService) claimCampaignRecipients(ctx context.Context, campaignID uuid.UUID, limit int) ([]models.NotificationCampaignRecipient, error) {
	var recipients []models.NotificationCampaignRecipient
	now := time.Now().UTC()
	retryBefore := now.Add(-10 * time.Second)
	staleProcessing := now.Add(-2 * time.Minute)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(`campaign_id = ? AND attempts < 3 AND (
				(status = 'pending' AND (attempts = 0 OR updated_at < ?))
				OR (status = 'processing' AND updated_at < ?)
			)`, campaignID, retryBefore, staleProcessing).
			Order("created_at ASC").Limit(limit).Find(&recipients).Error; err != nil {
			return err
		}
		if len(recipients) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, 0, len(recipients))
		for i := range recipients {
			ids = append(ids, recipients[i].ID)
			recipients[i].Attempts++
			recipients[i].Status = "processing"
			recipients[i].UpdatedAt = now
		}
		return tx.Model(&models.NotificationCampaignRecipient{}).Where("id IN ?", ids).
			Updates(map[string]any{
				"status": "processing", "attempts": gorm.Expr("attempts + 1"), "updated_at": now,
			}).Error
	})
	return recipients, err
}

func (s *AdminService) failCampaign(ctx context.Context, campaignID uuid.UUID, reason string) {
	now := time.Now().UTC()
	s.log.Warn("campaign failed", zap.String("campaign_id", campaignID.String()), zap.String("reason", truncate(reason, 200)))
	_ = s.db.WithContext(ctx).Model(&models.NotificationCampaign{}).
		Where("id = ? AND status = ?", campaignID, "running").
		Updates(map[string]any{"status": "failed", "completed_at": now, "updated_at": now}).Error
}

func (s *AdminService) finalizeCampaign(ctx context.Context, campaignID uuid.UUID) {
	var processed, failed int64
	_ = s.db.WithContext(ctx).Model(&models.NotificationCampaignRecipient{}).
		Where("campaign_id = ? AND status = ?", campaignID, "processed").Count(&processed).Error
	_ = s.db.WithContext(ctx).Model(&models.NotificationCampaignRecipient{}).
		Where("campaign_id = ? AND status = ?", campaignID, "failed").Count(&failed).Error
	status := "completed"
	if processed == 0 && failed > 0 {
		status = "failed"
	}
	now := time.Now().UTC()
	_ = s.db.WithContext(ctx).Model(&models.NotificationCampaign{}).Where("id = ? AND status = ?", campaignID, "running").
		Updates(map[string]any{
			"status": status, "processed_recipients": processed, "failed_recipients": failed,
			"completed_at": now, "updated_at": now,
		}).Error
}
