package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

// GoldRepository persists and queries calculated gram-gold prices.
type GoldRepository interface {
	Save(ctx context.Context, g domain.GramGold) error
	History(ctx context.Context, symbol string, from, to time.Time, limit int) ([]models.GoldPrice, error)
	Aggregate(ctx context.Context, symbol, interval string, from, to time.Time) ([]models.OHLC, error)
	Latest(ctx context.Context, symbol string) (models.GoldPrice, error)
	// PruneOlderThan bounds the legacy raw gold stream. Long-range charts use
	// the generalized candle table; this table only needs a recent safety window.
	PruneOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type goldRepository struct {
	db      *gorm.DB
	metrics *metrics.Metrics
}

// NewGoldRepository builds a GORM-backed repository.
func NewGoldRepository(db *gorm.DB, m *metrics.Metrics) GoldRepository {
	return &goldRepository{db: db, metrics: m}
}

func (r *goldRepository) observe(start time.Time, err error) {
	if r.metrics != nil {
		r.metrics.DBLatency.Observe(time.Since(start).Seconds())
		if err != nil {
			r.metrics.DBErrors.Inc()
		}
	}
}

func (r *goldRepository) Save(ctx context.Context, g domain.GramGold) error {
	start := time.Now()
	rec := models.GoldPrice{
		ID:        uuid.New(),
		Symbol:    g.Symbol,
		Price:     g.Price,
		MidPrice:  g.MidPrice,
		Spread:    g.Spread,
		XAUUSD:    g.XAUUSD,
		USDTRY:    g.USDTRY,
		Provider:  g.Provider,
		CreatedAt: g.Timestamp,
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}
	err := r.db.WithContext(ctx).Create(&rec).Error
	r.observe(start, err)
	return err
}

func (r *goldRepository) History(ctx context.Context, symbol string, from, to time.Time, limit int) ([]models.GoldPrice, error) {
	start := time.Now()
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}
	q := r.db.WithContext(ctx).Model(&models.GoldPrice{})
	if symbol != "" {
		q = q.Where("symbol = ?", symbol)
	}
	if !from.IsZero() {
		q = q.Where("created_at >= ?", from)
	}
	if !to.IsZero() {
		q = q.Where("created_at <= ?", to)
	}
	var rows []models.GoldPrice
	err := q.Order("created_at DESC").Limit(limit).Find(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *goldRepository) Latest(ctx context.Context, symbol string) (models.GoldPrice, error) {
	start := time.Now()
	var rec models.GoldPrice
	q := r.db.WithContext(ctx).Model(&models.GoldPrice{})
	if symbol != "" {
		q = q.Where("symbol = ?", symbol)
	}
	err := q.Order("created_at DESC").First(&rec).Error
	r.observe(start, err)
	return rec, err
}

func (r *goldRepository) PruneOlderThan(ctx context.Context, before time.Time) (int64, error) {
	start := time.Now()
	var total int64
	for {
		res := r.db.WithContext(ctx).Exec(`
WITH doomed AS (
	SELECT ctid
	FROM gold_prices
	WHERE created_at < ?
	LIMIT ?
)
DELETE FROM gold_prices AS p
USING doomed
WHERE p.ctid = doomed.ctid;`, before, timeSeriesPruneBatchSize)
		if res.Error != nil {
			r.observe(start, res.Error)
			return total, res.Error
		}
		total += res.RowsAffected
		if res.RowsAffected < timeSeriesPruneBatchSize {
			r.observe(start, nil)
			return total, nil
		}
		if err := ctx.Err(); err != nil {
			r.observe(start, err)
			return total, err
		}
	}
}

// intervalToSeconds maps supported aggregation intervals to seconds. The weekly
// and monthly buckets keep long-range charts (5Y/MAX) at a sane candle count.
var intervalToSeconds = map[string]int{
	"1m":  60,
	"5m":  300,
	"15m": 900,
	"30m": 1800,
	"1h":  3600,
	"4h":  14400,
	"1d":  86400,
	"1w":  604800,
	"1mo": 2592000, // 30d epoch bucket (approximate, chart-friendly)
}

// SupportedInterval reports whether interval is valid.
func SupportedInterval(interval string) bool {
	_, ok := intervalToSeconds[interval]
	return ok
}

// Aggregate builds OHLC candles using Postgres time bucketing (to_timestamp/floor).
func (r *goldRepository) Aggregate(ctx context.Context, symbol, interval string, from, to time.Time) ([]models.OHLC, error) {
	start := time.Now()
	secs, ok := intervalToSeconds[interval]
	if !ok {
		return nil, fmt.Errorf("unsupported interval %q", interval)
	}

	// Bucket by flooring epoch seconds to the interval width; first/last via window.
	query := `
WITH bucketed AS (
  SELECT
    to_timestamp(floor(extract(epoch from created_at) / ?) * ?) AS bucket,
    price,
    created_at,
    first_value(price) OVER (PARTITION BY floor(extract(epoch from created_at) / ?) ORDER BY created_at ASC) AS open_price,
    first_value(price) OVER (PARTITION BY floor(extract(epoch from created_at) / ?) ORDER BY created_at DESC) AS close_price
  FROM gold_prices
  WHERE symbol = ? AND created_at >= ? AND created_at <= ?
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

	if from.IsZero() {
		from = time.Now().Add(-24 * time.Hour)
	}
	if to.IsZero() {
		to = time.Now()
	}

	var rows []models.OHLC
	err := r.db.WithContext(ctx).Raw(query,
		secs, secs, secs, secs, symbol, from, to,
	).Scan(&rows).Error
	r.observe(start, err)
	return rows, err
}
