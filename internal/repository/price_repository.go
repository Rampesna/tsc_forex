package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

// PriceRepository persists and queries the generalized per-instrument history.
type PriceRepository interface {
	Save(ctx context.Context, p domain.InstrumentPrice) error
	History(ctx context.Context, code string, from, to time.Time, limit int) ([]models.PriceTick, error)
	Latest(ctx context.Context, code string) (models.PriceTick, error)
	Aggregate(ctx context.Context, code, interval string, from, to time.Time) ([]models.OHLC, error)
	Stats(ctx context.Context, code string, since time.Time) (models.PriceStats, error)
	Sparklines(ctx context.Context, points int) (map[string][]float64, error)
}

type priceRepository struct {
	db      *gorm.DB
	metrics *metrics.Metrics
}

// NewPriceRepository builds a GORM-backed instrument price repository.
func NewPriceRepository(db *gorm.DB, m *metrics.Metrics) PriceRepository {
	return &priceRepository{db: db, metrics: m}
}

func (r *priceRepository) observe(start time.Time, err error) {
	if r.metrics != nil {
		r.metrics.DBLatency.Observe(time.Since(start).Seconds())
		if err != nil && err != gorm.ErrRecordNotFound {
			r.metrics.DBErrors.Inc()
		}
	}
}

func (r *priceRepository) Save(ctx context.Context, p domain.InstrumentPrice) error {
	start := time.Now()
	rec := models.PriceTick{
		Code:      p.Code,
		Kind:      p.Kind,
		Price:     p.Price,
		Bid:       p.Bid,
		Ask:       p.Ask,
		Provider:  p.Provider,
		CreatedAt: p.Timestamp,
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}
	err := r.db.WithContext(ctx).Create(&rec).Error
	r.observe(start, err)
	return err
}

func (r *priceRepository) History(ctx context.Context, code string, from, to time.Time, limit int) ([]models.PriceTick, error) {
	start := time.Now()
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}
	q := r.db.WithContext(ctx).Model(&models.PriceTick{})
	if code != "" {
		q = q.Where("code = ?", code)
	}
	if !from.IsZero() {
		q = q.Where("created_at >= ?", from)
	}
	if !to.IsZero() {
		q = q.Where("created_at <= ?", to)
	}
	var rows []models.PriceTick
	err := q.Order("created_at DESC").Limit(limit).Find(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *priceRepository) Latest(ctx context.Context, code string) (models.PriceTick, error) {
	start := time.Now()
	var rec models.PriceTick
	q := r.db.WithContext(ctx).Model(&models.PriceTick{})
	if code != "" {
		q = q.Where("code = ?", code)
	}
	err := q.Order("created_at DESC").First(&rec).Error
	r.observe(start, err)
	return rec, err
}

// Stats returns a rolling-window summary (open/high/low/last/count) for an
// instrument since the given time. Used for the 24h header on the detail screen.
func (r *priceRepository) Stats(ctx context.Context, code string, since time.Time) (models.PriceStats, error) {
	start := time.Now()
	if since.IsZero() {
		since = time.Now().Add(-24 * time.Hour)
	}
	query := `
SELECT
  COALESCE((SELECT price FROM price_ticks WHERE code = ? AND created_at >= ? ORDER BY created_at ASC LIMIT 1), 0) AS open,
  COALESCE(MAX(price), 0) AS high,
  COALESCE(MIN(price), 0) AS low,
  COALESCE((SELECT price FROM price_ticks WHERE code = ? AND created_at >= ? ORDER BY created_at DESC LIMIT 1), 0) AS last,
  COUNT(*) AS count
FROM price_ticks
WHERE code = ? AND created_at >= ?;`
	var s models.PriceStats
	err := r.db.WithContext(ctx).Raw(query, code, since, code, since, code, since).Scan(&s).Error
	r.observe(start, err)
	return s, err
}

// Sparklines returns the last N prices per instrument (ascending by time) in a
// single query, used to seed the market-list mini charts with real DB history.
func (r *priceRepository) Sparklines(ctx context.Context, points int) (map[string][]float64, error) {
	start := time.Now()
	if points <= 0 || points > 200 {
		points = 10
	}
	query := `
SELECT code, price FROM (
  SELECT code, price, created_at,
         row_number() OVER (PARTITION BY code ORDER BY created_at DESC) AS rn
  FROM price_ticks
  WHERE created_at > now() - interval '6 hours'
) t
WHERE rn <= ?
ORDER BY code ASC, created_at ASC;`
	type row struct {
		Code  string
		Price float64
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(query, points).Scan(&rows).Error
	r.observe(start, err)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]float64, 32)
	for _, rr := range rows {
		out[rr.Code] = append(out[rr.Code], rr.Price)
	}
	return out, nil
}

// Aggregate builds OHLC candles for an instrument using epoch-second bucketing.
func (r *priceRepository) Aggregate(ctx context.Context, code, interval string, from, to time.Time) ([]models.OHLC, error) {
	start := time.Now()
	secs, ok := intervalToSeconds[interval]
	if !ok {
		return nil, fmt.Errorf("unsupported interval %q", interval)
	}
	if from.IsZero() {
		from = time.Now().Add(-24 * time.Hour)
	}
	if to.IsZero() {
		to = time.Now()
	}
	query := `
WITH bucketed AS (
  SELECT
    to_timestamp(floor(extract(epoch from created_at) / ?) * ?) AS bucket,
    price,
    created_at,
    first_value(price) OVER (PARTITION BY floor(extract(epoch from created_at) / ?) ORDER BY created_at ASC) AS open_price,
    first_value(price) OVER (PARTITION BY floor(extract(epoch from created_at) / ?) ORDER BY created_at DESC) AS close_price
  FROM price_ticks
  WHERE code = ? AND created_at >= ? AND created_at <= ?
)
SELECT
  bucket,
  max(open_price) AS open,
  max(price) AS high,
  min(price) AS low,
  max(close_price) AS close,
  count(*) AS count
FROM bucketed
GROUP BY bucket
ORDER BY bucket ASC;`
	var rows []models.OHLC
	err := r.db.WithContext(ctx).Raw(query, secs, secs, secs, secs, code, from, to).Scan(&rows).Error
	r.observe(start, err)
	return rows, err
}
