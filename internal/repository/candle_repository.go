package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

// CandleRepository persists and queries pre-aggregated OHLC candles. Charts read
// from here (a compact, indexed table) instead of aggregating raw price_ticks at
// request time — the difference between a multi-second scan of millions of rows
// and a millisecond indexed range read.
type CandleRepository interface {
	// UpsertBatch idempotently writes closed candles. Concurrent writers (multiple
	// pods observing the same upstream stream) converge: high=GREATEST, low=LEAST,
	// close=latest, count=GREATEST — so no double counting or lost extremes.
	UpsertBatch(ctx context.Context, candles []models.Candle) error
	// Range returns candles for one instrument+resolution in [from,to] ascending.
	Range(ctx context.Context, code, resolution string, from, to time.Time, limit int) ([]models.OHLC, error)
	// FirstOpenSince returns, per instrument, the open of the earliest candle at
	// `resolution` with bucket >= since. Seeds day-open baselines fast: a DISTINCT
	// ON over the tiny candles table instead of a day of raw ticks.
	FirstOpenSince(ctx context.Context, resolution string, since time.Time) (map[string]float64, error)
	// SeedFromTicks aggregates existing price_ticks into candles for a resolution
	// over [from,to]. Idempotent (does not overwrite live candles). Used once to
	// bootstrap chart history (including the historical backfill rows).
	SeedFromTicks(ctx context.Context, resolution string, secs int, from, to time.Time) (int64, error)
	// HasHistoryBefore reports whether any candle older than `before` exists for a
	// resolution. Used to decide whether to bootstrap history from ticks WITHOUT
	// being fooled by the just-flushed current (forming) candle.
	HasHistoryBefore(ctx context.Context, resolution string, before time.Time) (bool, error)
	// TryLockBootstrap acquires a cluster-wide advisory lock so only ONE pod runs
	// the heavy tick→candle bootstrap (5 pods hammering a weak DB at once starves
	// it). Returns locked=false if another pod holds it; call release when done.
	TryLockBootstrap(ctx context.Context) (locked bool, release func(), err error)
	// Prune deletes candles for a resolution older than `before` (retention).
	Prune(ctx context.Context, resolution string, before time.Time) (int64, error)
}

type candleRepository struct {
	db      *gorm.DB
	metrics *metrics.Metrics
}

// NewCandleRepository builds a GORM-backed candle repository.
func NewCandleRepository(db *gorm.DB, m *metrics.Metrics) CandleRepository {
	return &candleRepository{db: db, metrics: m}
}

func (r *candleRepository) observe(start time.Time, err error) {
	if r.metrics != nil {
		r.metrics.DBLatency.Observe(time.Since(start).Seconds())
		if err != nil && err != gorm.ErrRecordNotFound {
			r.metrics.DBErrors.Inc()
		}
	}
}

func (r *candleRepository) UpsertBatch(ctx context.Context, candles []models.Candle) error {
	if len(candles) == 0 {
		return nil
	}
	start := time.Now()
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "code"}, {Name: "resolution"}, {Name: "bucket"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"high":  gorm.Expr("GREATEST(candles.high, EXCLUDED.high)"),
			"low":   gorm.Expr("LEAST(candles.low, EXCLUDED.low)"),
			"close": gorm.Expr("EXCLUDED.close"),
			"count": gorm.Expr("GREATEST(candles.count, EXCLUDED.count)"),
		}),
	}).CreateInBatches(candles, 500).Error
	r.observe(start, err)
	return err
}

func (r *candleRepository) Range(ctx context.Context, code, resolution string, from, to time.Time, limit int) ([]models.OHLC, error) {
	start := time.Now()
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}
	q := r.db.WithContext(ctx).Model(&models.Candle{}).
		Select("bucket, open, high, low, close, count").
		Where("code = ? AND resolution = ?", code, resolution)
	if !from.IsZero() {
		q = q.Where("bucket >= ?", from)
	}
	if !to.IsZero() {
		q = q.Where("bucket <= ?", to)
	}
	// Newest first with LIMIT keeps the most recent window when the range is huge,
	// then we return ascending for charts.
	var rows []models.OHLC
	err := q.Order("bucket DESC").Limit(limit).Find(&rows).Error
	r.observe(start, err)
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, nil
}

func (r *candleRepository) FirstOpenSince(ctx context.Context, resolution string, since time.Time) (map[string]float64, error) {
	start := time.Now()
	type row struct {
		Code string
		Open float64
	}
	var rows []row
	query := `
SELECT DISTINCT ON (code) code, open
FROM candles
WHERE resolution = ? AND bucket >= ?
ORDER BY code, bucket ASC;`
	err := r.db.WithContext(ctx).Raw(query, resolution, since).Scan(&rows).Error
	r.observe(start, err)
	if err != nil {
		return nil, err
	}
	out := make(map[string]float64, len(rows))
	for _, rr := range rows {
		out[rr.Code] = rr.Open
	}
	return out, nil
}

func (r *candleRepository) SeedFromTicks(ctx context.Context, resolution string, secs int, from, to time.Time) (int64, error) {
	start := time.Now()
	// Bucket raw ticks and insert candles that don't already exist. first/last
	// via window functions give open/close; ON CONFLICT DO NOTHING protects any
	// live candles already written by the running aggregator.
	query := `
INSERT INTO candles (code, resolution, bucket, open, high, low, close, count)
SELECT code, ? AS resolution, bucket,
       max(open_price) AS open,
       max(price)      AS high,
       min(price)      AS low,
       max(close_price) AS close,
       count(*)        AS count
FROM (
  SELECT code,
         to_timestamp(floor(extract(epoch from created_at) / ?) * ?) AS bucket,
         price,
         first_value(price) OVER (PARTITION BY code, floor(extract(epoch from created_at) / ?) ORDER BY created_at ASC)  AS open_price,
         first_value(price) OVER (PARTITION BY code, floor(extract(epoch from created_at) / ?) ORDER BY created_at DESC) AS close_price
  FROM price_ticks
  WHERE created_at >= ? AND created_at <= ?
) b
GROUP BY code, bucket
ON CONFLICT (code, resolution, bucket) DO NOTHING;`
	res := r.db.WithContext(ctx).Exec(query, resolution, secs, secs, secs, secs, from, to)
	r.observe(start, res.Error)
	return res.RowsAffected, res.Error
}

func (r *candleRepository) HasHistoryBefore(ctx context.Context, resolution string, before time.Time) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.Candle{}).
		Where("resolution = ? AND bucket < ?", resolution, before).Limit(1).Count(&n).Error
	return n > 0, err
}

// bootstrapLockKey is an arbitrary, app-unique advisory-lock key.
const bootstrapLockKey int64 = 917326451

func (r *candleRepository) TryLockBootstrap(ctx context.Context) (bool, func(), error) {
	sqlDB, err := r.db.DB()
	if err != nil {
		return false, nil, err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return false, nil, err
	}
	var ok bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", bootstrapLockKey).Scan(&ok); err != nil {
		_ = conn.Close()
		return false, nil, err
	}
	if !ok {
		_ = conn.Close()
		return false, nil, nil
	}
	release := func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", bootstrapLockKey)
		_ = conn.Close()
	}
	return true, release, nil
}

func (r *candleRepository) Prune(ctx context.Context, resolution string, before time.Time) (int64, error) {
	start := time.Now()
	res := r.db.WithContext(ctx).
		Where("resolution = ? AND bucket < ?", resolution, before).
		Delete(&models.Candle{})
	r.observe(start, res.Error)
	return res.RowsAffected, res.Error
}
