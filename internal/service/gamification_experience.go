package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

type DailyPlanInput struct {
	RiskAppetite string   `json:"risk_appetite"`
	WatchCodes   []string `json:"watch_codes"`
	LearningGoal string   `json:"learning_goal"`
	Reflection   string   `json:"reflection"`
}

type DailyPlanView struct {
	ID           uuid.UUID `json:"id"`
	DayKey       string    `json:"day_key"`
	RiskAppetite string    `json:"risk_appetite"`
	WatchCodes   []string  `json:"watch_codes"`
	LearningGoal string    `json:"learning_goal"`
	Reflection   string    `json:"reflection"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type QuizQuestion struct {
	Code    string   `json:"code"`
	Text    string   `json:"text"`
	Choices []string `json:"choices"`
	answer  int
}

type LessonStatus struct {
	Code         string       `json:"code"`
	Title        string       `json:"title"`
	Summary      string       `json:"summary"`
	Cards        []string     `json:"cards"`
	Question     QuizQuestion `json:"question"`
	Mastery      int          `json:"mastery"`
	Attempts     int          `json:"attempts"`
	Completed    bool         `json:"completed"`
	DueForReview bool         `json:"due_for_review"`
	NextReviewAt *time.Time   `json:"next_review_at,omitempty"`
}

type DisciplineQuest struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	XP          int    `json:"xp"`
	Completed   bool   `json:"completed"`
}

type SeasonView struct {
	Code      string        `json:"code"`
	Name      string        `json:"name"`
	League    string        `json:"league"`
	Points    int64         `json:"points"`
	Rank      int           `json:"rank"`
	EndsAt    time.Time     `json:"ends_at"`
	Standings []LeaderEntry `json:"standings"`
	Scoring   string        `json:"scoring"`
}

type WeeklyLearningSummary struct {
	MasteredLessons int   `json:"mastered_lessons"`
	TotalLessons    int   `json:"total_lessons"`
	SeasonPoints    int64 `json:"season_points"`
	DueReviews      int   `json:"due_reviews"`
}

type GamificationExperience struct {
	RuleVersion string                          `json:"rule_version"`
	DailyPlan   *DailyPlanView                  `json:"daily_plan,omitempty"`
	Lessons     []LessonStatus                  `json:"lessons"`
	Discipline  []DisciplineQuest               `json:"discipline_quests"`
	Season      SeasonView                      `json:"season"`
	PaperTrades []models.PaperTrade             `json:"paper_trades"`
	Privacy     models.GamificationPrivacy      `json:"privacy"`
	StudyGroups []repository.StudyGroupOverview `json:"study_groups"`
	Weekly      WeeklyLearningSummary           `json:"weekly_summary"`
}

type QuizAttemptInput struct {
	LessonCode     string `json:"lesson_code"`
	QuestionCode   string `json:"question_code"`
	Answer         int    `json:"answer"`
	IdempotencyKey string `json:"idempotency_key"`
}

type QuizAttemptResult struct {
	Correct  bool         `json:"correct"`
	XPGained int          `json:"xp_gained"`
	Created  bool         `json:"created"`
	Lesson   LessonStatus `json:"lesson"`
}

type PaperTradeInput struct {
	ClientID      string    `json:"client_id"`
	Code          string    `json:"code"`
	Side          string    `json:"side"`
	Quantity      float64   `json:"quantity"`
	EntryPrice    float64   `json:"entry_price"`
	Thesis        string    `json:"thesis"`
	RiskBudgetPct float64   `json:"risk_budget_pct"`
	Invalidation  string    `json:"invalidation"`
	OccurredAt    time.Time `json:"occurred_at"`
}

var lessonCatalog = []LessonStatus{
	{
		Code: "risk_budget", Title: "İşlemden önce risk bütçesi",
		Summary: "Pozisyon büyüklüğünü tahmine değil, kabul edilebilir kayba göre kur.",
		Cards: []string{
			"Risk bütçesi, tek bir karar yanlış çıktığında portföyün ne kadarını kaybetmeyi kabul ettiğini söyler.",
			"Önce geçersizleşme seviyesini, sonra bu seviyeye göre pozisyon miktarını belirle.",
			"Kârlı sonuç doğru süreç anlamına gelmez; plana uyum ayrı değerlendirilir.",
		},
		Question: QuizQuestion{
			Code: "risk_budget_q1", Text: "Pozisyon büyüklüğü en sağlıklı hangi sırayla belirlenir?",
			Choices: []string{
				"Önce hedef kâr, sonra kaldıraç",
				"Önce geçersizleşme seviyesi ve kabul edilebilir kayıp",
				"Son fiyat hareketine göre maksimum miktar",
			},
			answer: 1,
		},
	},
	{
		Code: "diversification", Title: "Yoğunlaşmayı doğru okumak",
		Summary: "Varlık sayısı tek başına çeşitlendirme değildir; ortak risk kaynaklarını izle.",
		Cards: []string{
			"Aynı makro etkene duyarlı farklı semboller, gerçekte tek bir yoğun risk oluşturabilir.",
			"Portföy yüzdelerini ve para birimi/varlık sınıfı dağılımını birlikte değerlendir.",
			"Yoğunlaşma uyarısı otomatik satış emri değil, kararını yeniden inceleme davetidir.",
		},
		Question: QuizQuestion{
			Code: "diversification_q1", Text: "Gerçek çeşitlendirmeyi en iyi hangisi açıklar?",
			Choices: []string{
				"Sembol sayısını artırmak",
				"Her pozisyona eşit para koymak",
				"Ortak risk kaynaklarını ve ağırlıkları birlikte sınırlamak",
			},
			answer: 2,
		},
	},
	{
		Code: "fees_behavior", Title: "Masraf ve davranış etkisi",
		Summary: "Sık işlem, küçük görünen ücretleri ve karar hatalarını bileşik hale getirir.",
		Cards: []string{
			"Komisyon, spread ve kur dönüşümü toplam maliyetin ayrı parçalarıdır.",
			"İşlem günlüğü, sonuçtan bağımsız olarak tekrarlanan davranış hatalarını görünür kılar.",
			"Bir işlemi sırf önceki kaybı geri almak için açmak plan değil, tepki davranışıdır.",
		},
		Question: QuizQuestion{
			Code: "fees_behavior_q1", Text: "Süreç kalitesini ölçmek için en yararlı kayıt hangisidir?",
			Choices: []string{
				"Yalnız gerçekleşen getiri",
				"Tez, risk sınırı, masraflar ve sonradan değerlendirme",
				"Sadece işlem sayısı",
			},
			answer: 1,
		},
	},
	{
		Code: "macro_calendar", Title: "Veri takvimi ve olay riski",
		Summary: "Faiz, enflasyon ve istihdam verileri öncesinde belirsizliği planına dahil et.",
		Cards: []string{
			"Önemli veri açıklamaları fiyatı iki yönde de hızla hareket ettirebilir; tahmin kadar zamanlama da risktir.",
			"Olay öncesinde pozisyon büyüklüğü, geçersizleşme seviyesi ve işlem yapmama seçeneği birlikte düşünülmelidir.",
			"İlk fiyat tepkisi kalıcı yön anlamına gelmez; veri, beklenti ve önceki değer birlikte okunur.",
		},
		Question: QuizQuestion{
			Code: "macro_calendar_q1", Text: "Yüksek etkili veri öncesinde en disiplinli yaklaşım hangisidir?",
			Choices: []string{
				"İlk hareketi kaçırmamak için riski artırmak",
				"Olay zamanını, risk sınırını ve işlem yapmama seçeneğini önceden belirlemek",
				"Yalnızca sosyal medya tahminini izlemek",
			},
			answer: 1,
		},
	},
	{
		Code: "cross_market", Title: "Piyasalar arası ilişkiyi okumak",
		Summary: "Bir varlığı yorumlarken kur, faiz, dolar ve risk iştahını aynı çerçevede değerlendir.",
		Cards: []string{
			"Gram altın, ons altın ile USD/TRY hareketinin birleşiminden etkilenir; tek grafiğe bakmak resmi eksik bırakır.",
			"Korelasyonlar sabit değildir; farklı dönemlerde ilişkinin yönü ve gücü değişebilir.",
			"Birden çok gösterge aynı tezi desteklese bile risk sınırı gereksiz hale gelmez.",
		},
		Question: QuizQuestion{
			Code: "cross_market_q1", Text: "Gram altındaki hareketi değerlendirirken hangi ikili temel başlangıçtır?",
			Choices: []string{"Yalnız BIST ve Bitcoin", "Ons altın ve USD/TRY", "Sadece günlük işlem hacmi"},
			answer:  1,
		},
	},
	{
		Code: "source_quality", Title: "Kaynak ve iddia kalitesi",
		Summary: "Bir yorumu karar girdisine çevirmeden önce tarihini, kaynağını ve karşı görüşü kontrol et.",
		Cards: []string{
			"Güncel olmayan doğru bir bilgi, bugünkü karar için yanıltıcı olabilir; yayın ve olay tarihini ayrı kontrol et.",
			"Tek bir yorum yerine birincil veri ile bağımsız ikinci bir kaynağı karşılaştır.",
			"Yapay zekâ çıktıları da hata içerebilir; kaynaklar kararın yerine değil, incelemenin başlangıcına konur.",
		},
		Question: QuizQuestion{
			Code: "source_quality_q1", Text: "Bir piyasa iddiasını kullanmadan önce en güçlü kontrol hangisidir?",
			Choices: []string{
				"En çok paylaşılan yorumu doğru kabul etmek",
				"Yayın/olay tarihini ve birincil veriyi bağımsız kaynakla karşılaştırmak",
				"Yalnız başlığa bakmak",
			},
			answer: 1,
		},
	},
}

var disciplineCatalog = []DisciplineQuest{
	{Code: "allocation_review", Title: "Yoğunlaşmayı gözden geçir", Description: "En büyük portföy ağırlığının oluşturduğu riski not et.", XP: 25},
	{Code: "thesis_review", Title: "Yatırım tezini sınırla", Description: "Bir pozisyonun hangi koşulda geçersiz olacağını yaz.", XP: 25},
	{Code: "monthly_review", Title: "Süreç günlüğü", Description: "Son kararında plana uyduğun ve sapma yaşadığın noktayı değerlendir.", XP: 25},
	{Code: "evidence_review", Title: "Veri ile varsayımı ayır", Description: "Kararındaki doğrulanmış veriyi ve yalnızca varsayım olan kısmı ayrı ayrı yaz.", XP: 25},
	{Code: "scenario_review", Title: "Alternatif senaryo kur", Description: "Ana beklentinin tersine gelişebilecek bir senaryoyu ve vereceğin tepkiyi not et.", XP: 25},
	{Code: "emotion_review", Title: "Duygu kontrolü yap", Description: "Kararında acele, kayıp telafisi veya fırsatı kaçırma baskısı olup olmadığını değerlendir.", XP: 25},
}

func (g *GamificationService) Experience(ctx context.Context, uid uuid.UUID) (GamificationExperience, error) {
	progress, err := g.repo.LearningProgress(ctx, uid)
	if err != nil {
		return GamificationExperience{}, err
	}
	lessons := buildLessonStatuses(progress)
	privacy, err := g.repo.Privacy(ctx, uid)
	if err != nil {
		return GamificationExperience{}, err
	}
	trades, err := g.repo.ListPaperTrades(ctx, uid, 40)
	if err != nil {
		return GamificationExperience{}, err
	}
	groups, err := g.repo.StudyGroups(ctx, uid)
	if err != nil {
		return GamificationExperience{}, err
	}
	season, err := g.Season(ctx, uid, 20)
	if err != nil {
		return GamificationExperience{}, err
	}

	var planView *DailyPlanView
	plan, planErr := g.repo.DailyPlan(ctx, uid, g.today())
	if planErr == nil {
		v := dailyPlanView(plan)
		planView = &v
	} else if !errors.Is(planErr, gorm.ErrRecordNotFound) {
		return GamificationExperience{}, planErr
	}
	completedReviews, err := g.repo.DisciplineReviewCodes(ctx, uid, g.today())
	if err != nil {
		return GamificationExperience{}, err
	}
	discipline := make([]DisciplineQuest, len(disciplineCatalog))
	copy(discipline, disciplineCatalog)
	for i := range discipline {
		discipline[i].Completed = completedReviews[discipline[i].Code]
	}
	mastered, due := 0, 0
	for _, lesson := range lessons {
		if lesson.Completed {
			mastered++
		}
		if lesson.DueForReview {
			due++
		}
	}
	return GamificationExperience{
		RuleVersion: gamificationRuleVersion, DailyPlan: planView, Lessons: lessons,
		Discipline: discipline, Season: season, PaperTrades: trades, Privacy: privacy,
		StudyGroups: groups, Weekly: WeeklyLearningSummary{
			MasteredLessons: mastered, TotalLessons: len(lessons),
			SeasonPoints: season.Points, DueReviews: due,
		},
	}, nil
}

func (g *GamificationService) SaveDailyPlan(
	ctx context.Context,
	uid uuid.UUID,
	input DailyPlanInput,
) (CheckInResult, error) {
	input.RiskAppetite = strings.ToLower(strings.TrimSpace(input.RiskAppetite))
	if input.RiskAppetite != "low" && input.RiskAppetite != "moderate" && input.RiskAppetite != "high" {
		return CheckInResult{}, fmt.Errorf("risk_appetite must be low, moderate or high")
	}
	if len(input.WatchCodes) > 3 {
		return CheckInResult{}, fmt.Errorf("watch_codes supports at most three instruments")
	}
	seen := map[string]bool{}
	codes := make([]string, 0, len(input.WatchCodes))
	for _, raw := range input.WatchCodes {
		code := strings.ToUpper(strings.TrimSpace(raw))
		if code == "" || seen[code] {
			continue
		}
		if _, ok := domain.CatalogByCode[code]; !ok {
			return CheckInResult{}, fmt.Errorf("unknown instrument code %s", code)
		}
		seen[code] = true
		codes = append(codes, code)
	}
	if !validLessonCode(input.LearningGoal) {
		return CheckInResult{}, fmt.Errorf("unknown learning goal")
	}
	if len([]rune(input.Reflection)) > 500 {
		return CheckInResult{}, fmt.Errorf("reflection is too long")
	}
	now := time.Now().UTC()
	plan := models.DailyPlan{
		UserID: uid, DayKey: g.today(), RiskAppetite: input.RiskAppetite,
		WatchCodes: strings.Join(codes, ","), LearningGoal: input.LearningGoal,
		Reflection: strings.TrimSpace(input.Reflection), CreatedAt: now, UpdatedAt: now,
	}
	if err := g.repo.SaveDailyPlan(ctx, &plan); err != nil {
		return CheckInResult{}, err
	}
	checkIn, err := g.CheckIn(ctx, uid)
	if err != nil {
		return CheckInResult{}, err
	}
	planAward, err := g.awardXP(ctx, uid, "daily-plan:"+g.today(), EventDailyPlan,
		"daily_plan", g.today(), "Günlük risk ve öğrenme planı", 5, 25)
	if err != nil {
		return CheckInResult{}, err
	}
	g.evaluate(ctx, uid, planAward.State, g.today())
	st, err := g.repo.State(ctx, uid)
	if err != nil {
		return CheckInResult{}, err
	}
	checkIn.Profile = g.buildProfile(ctx, uid, st)
	if planAward.Created {
		checkIn.XPGained += planAward.Ledger.Delta
	}
	return checkIn, nil
}

func (g *GamificationService) SubmitQuiz(
	ctx context.Context,
	uid uuid.UUID,
	input QuizAttemptInput,
) (QuizAttemptResult, error) {
	if len(input.IdempotencyKey) < 8 || len(input.IdempotencyKey) > 120 {
		return QuizAttemptResult{}, fmt.Errorf("invalid idempotency_key")
	}
	lesson, question, ok := findQuestion(input.LessonCode, input.QuestionCode)
	if !ok || input.Answer < 0 || input.Answer >= len(question.Choices) {
		return QuizAttemptResult{}, fmt.Errorf("invalid lesson question or answer")
	}
	rows, err := g.repo.LearningProgress(ctx, uid)
	if err != nil {
		return QuizAttemptResult{}, err
	}
	old := models.LearningProgress{UserID: uid, LessonCode: lesson.Code}
	for _, row := range rows {
		if row.LessonCode == lesson.Code {
			old = row
			break
		}
	}
	now := time.Now().UTC()
	correct := input.Answer == question.answer
	progress := old
	progress.UserID = uid
	progress.LessonCode = lesson.Code
	progress.Attempts++
	if correct {
		progress.Correct++
		progress.Mastery = min(100, progress.Mastery+35)
	} else {
		progress.Mastery = max(0, progress.Mastery-10)
	}
	if progress.Mastery >= 70 && progress.CompletedAt == nil {
		completed := now
		progress.CompletedAt = &completed
	}
	reviewAfter := 24 * time.Hour
	if progress.Mastery >= 90 {
		reviewAfter = 14 * 24 * time.Hour
	} else if progress.Mastery >= 70 {
		reviewAfter = 7 * 24 * time.Hour
	}
	next := now.Add(reviewAfter)
	progress.NextReviewAt = &next
	progress.UpdatedAt = now

	xp := 0
	awardKey := ""
	if correct && old.Correct == 0 {
		xp = 20
		awardKey = "quiz:first:" + lesson.Code + ":" + question.Code
	} else if correct && old.NextReviewAt != nil && !now.Before(*old.NextReviewAt) {
		year, week := now.ISOWeek()
		xp = 8
		awardKey = fmt.Sprintf("quiz:review:%s:%d-%02d", lesson.Code, year, week)
	}
	attempt := models.LearningAttempt{
		UserID: uid, IdempotencyKey: input.IdempotencyKey, LessonCode: lesson.Code,
		QuestionCode: question.Code, Answer: input.Answer, Correct: correct,
		XPAwarded: xp, CreatedAt: now,
	}
	created, err := g.repo.SaveLearningAttempt(ctx, &attempt, &progress)
	if err != nil {
		return QuizAttemptResult{}, err
	}
	if !created {
		progressRows, _ := g.repo.LearningProgress(ctx, uid)
		statuses := buildLessonStatuses(progressRows)
		return QuizAttemptResult{
			Correct: attempt.Correct, XPGained: attempt.XPAwarded,
			Created: false, Lesson: lessonByCode(statuses, lesson.Code),
		}, nil
	}
	if xp > 0 {
		out, awardErr := g.awardXP(ctx, uid, awardKey, EventLesson, "lesson",
			lesson.Code, "Doğrulanmış bilgi kontrolü", xp, min(xp, 20))
		if awardErr != nil {
			return QuizAttemptResult{}, awardErr
		}
		if !out.Created {
			xp = 0
		}
		g.evaluate(ctx, uid, out.State, g.today())
	}
	return QuizAttemptResult{
		Correct: correct, XPGained: xp, Created: true,
		Lesson: lessonByCode(buildLessonStatuses([]models.LearningProgress{progress}), lesson.Code),
	}, nil
}

func (g *GamificationService) CompleteDiscipline(
	ctx context.Context,
	uid uuid.UUID,
	code, note string,
) (repository.AwardOutcome, error) {
	valid := false
	for _, quest := range disciplineCatalog {
		if quest.Code == code {
			valid = true
			break
		}
	}
	note = strings.TrimSpace(note)
	if !valid || len([]rune(note)) < 20 || len([]rune(note)) > 1000 {
		return repository.AwardOutcome{}, fmt.Errorf("valid code and a 20-1000 character review are required")
	}
	review := models.DisciplineReview{
		UserID: uid, DayKey: g.today(), Code: code, Note: note, CreatedAt: time.Now().UTC(),
	}
	_, err := g.repo.SaveDisciplineReview(ctx, &review)
	if err != nil {
		return repository.AwardOutcome{}, err
	}
	out, err := g.awardXP(ctx, uid, "discipline:"+code+":"+g.today(), EventDiscipline,
		"discipline_review", code, "Portföy karar süreci değerlendirmesi", 25, 25)
	if err != nil {
		return repository.AwardOutcome{}, err
	}
	g.evaluate(ctx, uid, out.State, g.today())
	return out, nil
}

func (g *GamificationService) Season(
	ctx context.Context,
	uid uuid.UUID,
	limit int,
) (SeasonView, error) {
	code := g.currentSeasonCode()
	score, err := g.repo.SeasonScore(ctx, uid, code)
	if err != nil {
		return SeasonView{}, err
	}
	rows, err := g.repo.SeasonLeaderboard(ctx, code, limit)
	if err != nil {
		return SeasonView{}, err
	}
	entries := make([]LeaderEntry, 0, len(rows))
	myRank := 1
	for i, row := range rows {
		entry := LeaderEntry{
			Rank: i + 1, UserID: row.UserID.String(), DisplayName: row.DisplayName,
			AvatarURL: "", XP: row.XP, Level: levelForXP(row.XP),
		}
		if row.UserID == uid {
			myRank = i + 1
		}
		entries = append(entries, entry)
	}
	now := time.Now().In(g.loc)
	quarterEndMonth := time.Month(((int(now.Month())-1)/3+1)*3 + 1)
	year := now.Year()
	if quarterEndMonth == 13 {
		quarterEndMonth = 1
		year++
	}
	ends := time.Date(year, quarterEndMonth, 1, 0, 0, 0, 0, g.loc).UTC()
	return SeasonView{
		Code: code, Name: "Finansal Disiplin Sezonu", League: score.League,
		Points: score.Points, Rank: myRank, EndsAt: ends, Standings: entries,
		Scoring: "Puanlar getiriye göre değil; planlama, öğrenme, tekrar ve karar sürecine göre hesaplanır.",
	}, nil
}

func (g *GamificationService) CreatePaperTrade(
	ctx context.Context,
	uid uuid.UUID,
	input PaperTradeInput,
) (models.PaperTrade, bool, error) {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Side = strings.ToLower(strings.TrimSpace(input.Side))
	if _, ok := domain.CatalogByCode[input.Code]; !ok {
		return models.PaperTrade{}, false, fmt.Errorf("unknown instrument")
	}
	if input.Side != "buy" && input.Side != "sell" {
		return models.PaperTrade{}, false, fmt.Errorf("side must be buy or sell")
	}
	if len(input.ClientID) < 8 || input.Quantity <= 0 || input.EntryPrice <= 0 {
		return models.PaperTrade{}, false, fmt.Errorf("client_id, quantity and entry_price are required")
	}
	if input.RiskBudgetPct <= 0 || input.RiskBudgetPct > 5 {
		return models.PaperTrade{}, false, fmt.Errorf("risk_budget_pct must be between 0 and 5")
	}
	if len([]rune(strings.TrimSpace(input.Thesis))) < 20 ||
		len([]rune(strings.TrimSpace(input.Invalidation))) < 10 {
		return models.PaperTrade{}, false, fmt.Errorf("thesis and invalidation rule are too short")
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}
	if input.OccurredAt.After(time.Now().Add(time.Minute)) {
		return models.PaperTrade{}, false, fmt.Errorf("future paper trades are not allowed")
	}
	trade := models.PaperTrade{
		UserID: uid, ClientID: input.ClientID, Code: input.Code, Side: input.Side,
		Quantity: input.Quantity, EntryPrice: input.EntryPrice,
		Thesis: strings.TrimSpace(input.Thesis), RiskBudgetPct: input.RiskBudgetPct,
		Invalidation: strings.TrimSpace(input.Invalidation), OccurredAt: input.OccurredAt.UTC(),
	}
	created, err := g.repo.CreatePaperTrade(ctx, &trade)
	return trade, created, err
}

func (g *GamificationService) ReviewPaperTrade(
	ctx context.Context,
	uid, tradeID uuid.UUID,
	followedPlan bool,
	retrospective string,
) (models.PaperTrade, int, error) {
	retrospective = strings.TrimSpace(retrospective)
	if len([]rune(retrospective)) < 20 || len([]rune(retrospective)) > 1000 {
		return models.PaperTrade{}, 0, fmt.Errorf("retrospective must contain 20-1000 characters")
	}
	trade, err := g.repo.PaperTrade(ctx, uid, tradeID)
	if err != nil {
		return models.PaperTrade{}, 0, err
	}
	if trade.ReviewedAt != nil {
		return trade, 0, nil
	}
	now := time.Now().UTC()
	trade.FollowedPlan = &followedPlan
	trade.Retrospective = retrospective
	trade.ProcessScore = 100
	if !followedPlan {
		trade.ProcessScore = 70
	}
	trade.ReviewedAt = &now
	trade.UpdatedAt = now
	if err := g.repo.UpdatePaperTrade(ctx, &trade); err != nil {
		return models.PaperTrade{}, 0, err
	}
	out, err := g.awardXP(ctx, uid, "paper-review:"+trade.ID.String(), EventPaperReview,
		"paper_trade", trade.ID.String(), "Sonuçtan bağımsız karar retrospektifi", 30, 30)
	if err != nil {
		return models.PaperTrade{}, 0, err
	}
	return trade, out.Ledger.Delta, nil
}

func (g *GamificationService) SavePrivacy(
	ctx context.Context,
	uid uuid.UUID,
	privacy models.GamificationPrivacy,
) (models.GamificationPrivacy, error) {
	privacy.UserID = uid
	privacy.Alias = strings.TrimSpace(privacy.Alias)
	if len([]rune(privacy.Alias)) > 40 {
		return models.GamificationPrivacy{}, fmt.Errorf("alias is too long")
	}
	if err := g.repo.SavePrivacy(ctx, &privacy); err != nil {
		return models.GamificationPrivacy{}, err
	}
	return privacy, nil
}

func (g *GamificationService) CreateStudyGroup(
	ctx context.Context,
	uid uuid.UUID,
	name string,
) (repository.StudyGroupOverview, error) {
	privacy, err := g.repo.Privacy(ctx, uid)
	if err != nil {
		return repository.StudyGroupOverview{}, err
	}
	if !privacy.SocialOptIn {
		return repository.StudyGroupOverview{}, fmt.Errorf("social_opt_in is required")
	}
	name = strings.TrimSpace(name)
	if len([]rune(name)) < 3 || len([]rune(name)) > 60 {
		return repository.StudyGroupOverview{}, fmt.Errorf("group name must contain 3-60 characters")
	}
	group := models.StudyGroup{
		OwnerID: uid, Name: name, InviteCode: randomInviteCode(), CreatedAt: time.Now().UTC(),
	}
	member := models.StudyGroupMember{UserID: uid, Alias: safeAlias(privacy, uid)}
	if err := g.repo.CreateStudyGroup(ctx, &group, &member); err != nil {
		return repository.StudyGroupOverview{}, err
	}
	return repository.StudyGroupOverview{StudyGroup: group, MemberCount: 1}, nil
}

func (g *GamificationService) JoinStudyGroup(
	ctx context.Context,
	uid uuid.UUID,
	inviteCode string,
) error {
	privacy, err := g.repo.Privacy(ctx, uid)
	if err != nil {
		return err
	}
	if !privacy.SocialOptIn {
		return fmt.Errorf("social_opt_in is required")
	}
	member := models.StudyGroupMember{UserID: uid, Alias: safeAlias(privacy, uid)}
	return g.repo.JoinStudyGroup(ctx, strings.ToUpper(strings.TrimSpace(inviteCode)), &member)
}

func buildLessonStatuses(progress []models.LearningProgress) []LessonStatus {
	byCode := make(map[string]models.LearningProgress, len(progress))
	for _, row := range progress {
		byCode[row.LessonCode] = row
	}
	now := time.Now().UTC()
	out := make([]LessonStatus, len(lessonCatalog))
	for i, def := range lessonCatalog {
		def.Question.answer = 0
		row := byCode[def.Code]
		def.Mastery = row.Mastery
		def.Attempts = row.Attempts
		def.Completed = row.Mastery >= 70
		def.NextReviewAt = row.NextReviewAt
		def.DueForReview = row.NextReviewAt == nil || !now.Before(*row.NextReviewAt)
		out[i] = def
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DueForReview != out[j].DueForReview {
			return out[i].DueForReview
		}
		return out[i].Mastery < out[j].Mastery
	})
	return out
}

func findQuestion(lessonCode, questionCode string) (LessonStatus, QuizQuestion, bool) {
	for _, lesson := range lessonCatalog {
		if lesson.Code == lessonCode && lesson.Question.Code == questionCode {
			return lesson, lesson.Question, true
		}
	}
	return LessonStatus{}, QuizQuestion{}, false
}

func lessonByCode(lessons []LessonStatus, code string) LessonStatus {
	for _, lesson := range lessons {
		if lesson.Code == code {
			return lesson
		}
	}
	return LessonStatus{}
}

func validLessonCode(code string) bool {
	for _, lesson := range lessonCatalog {
		if lesson.Code == code {
			return true
		}
	}
	return false
}

func dailyPlanView(plan models.DailyPlan) DailyPlanView {
	codes := []string{}
	if plan.WatchCodes != "" {
		codes = strings.Split(plan.WatchCodes, ",")
	}
	return DailyPlanView{
		ID: plan.ID, DayKey: plan.DayKey, RiskAppetite: plan.RiskAppetite,
		WatchCodes: codes, LearningGoal: plan.LearningGoal, Reflection: plan.Reflection,
		CreatedAt: plan.CreatedAt, UpdatedAt: plan.UpdatedAt,
	}
}

func randomInviteCode() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return strings.ToUpper(uuid.NewString()[:8])
	}
	return strings.ToUpper(hex.EncodeToString(buf))
}

func safeAlias(privacy models.GamificationPrivacy, uid uuid.UUID) string {
	if privacy.Alias != "" {
		return privacy.Alias
	}
	return "Yatırımcı-" + strings.ToUpper(strings.ReplaceAll(uid.String()[:4], "-", ""))
}
