package service

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// ErrUnsupportedInterval is returned for a chart interval the store can't derive.
var ErrUnsupportedInterval = errors.New("unsupported interval")

// baseResolutions are the OHLC resolutions the store maintains natively. Every
// requested chart interval is derived from one of these (e.g. 30m from 15m, 1w
// and 1mo from 1d), so we persist a small, fixed set.
var baseResolutions = []struct {
	Name string
	Secs int
}{
	{"1m", 60},
	{"5m", 300},
	{"15m", 900},
	{"1h", 3600},
	{"4h", 14400},
	{"1d", 86400},
}

// intervalToBase maps a client-requested interval to (baseResolution, factor).
// factor>1 means re-bucket `factor` base candles into one requested candle.
var intervalToBase = map[string]struct {
	Base     string
	Secs     int
	Factor   int
	Calendar string
}{
	"1m":  {Base: "1m", Secs: 60, Factor: 1},
	"5m":  {Base: "5m", Secs: 300, Factor: 1},
	"15m": {Base: "15m", Secs: 900, Factor: 1},
	"30m": {Base: "15m", Secs: 900, Factor: 2},
	"1h":  {Base: "1h", Secs: 3600, Factor: 1},
	"4h":  {Base: "4h", Secs: 14400, Factor: 1},
	"1d":  {Base: "1d", Secs: 86400, Factor: 1},
	"1w":  {Base: "1d", Secs: 86400, Factor: 7, Calendar: "week"},
	"1mo": {Base: "1d", Secs: 86400, Factor: 31, Calendar: "month"},
}

const sparkCap = 40

type candleAgg struct {
	bucket time.Time
	open   float64
	high   float64
	low    float64
	close  float64
	count  int64
}

func (a *candleAgg) toModel(code, res string) models.Candle {
	return models.Candle{
		Code: code, Resolution: res, Bucket: a.bucket,
		Open: a.open, High: a.high, Low: a.low, Close: a.close, Count: a.count,
	}
}

func (a *candleAgg) toOHLC() models.OHLC {
	return models.OHLC{Bucket: a.bucket, Open: a.open, High: a.high, Low: a.low, Close: a.close, Count: a.count}
}

// CandleStore is the hot path for chart, sparkline and 24h-stat reads. It builds
// OHLC candles in memory from the engine's materialized instrument updates (NOT
// from throttled raw ticks), persists closed candles to a compact `candles`
// table for durable history, and answers reads from memory / an indexed range
// scan — never by aggregating millions of raw ticks at request time.
//
// It degrades gracefully: with no DB it still serves live in-memory candles,
// sparklines and stats (just without deep history).
type CandleStore struct {
	repo repository.CandleRepository // may be nil (minimal mode)
	log  *zap.Logger
	loc  *time.Location

	startOnce     sync.Once
	bootstrapOnce sync.Once
	workers       sync.WaitGroup

	mu      sync.RWMutex
	current map[string]*candleAgg // key: code|res → forming candle
	spark   map[string][]float64  // code → recent closes for mini charts
	pending []models.Candle       // closed candles awaiting DB flush
}

// NewCandleStore wires the store to the bus. repo may be nil.
func NewCandleStore(bus *eventbus.Bus, repo repository.CandleRepository, log *zap.Logger) *CandleStore {
	if log == nil {
		log = zap.NewNop()
	}
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		loc = time.UTC
	}
	s := &CandleStore{
		repo:    repo,
		log:     log,
		loc:     loc,
		current: make(map[string]*candleAgg, 1024),
		spark:   make(map[string][]float64, 128),
	}
	if bus != nil {
		bus.Subscribe(eventbus.TopicInstrumentUpdated, s.onUpdate)
	}
	return s
}

func bucketStart(ts time.Time, secs int) time.Time {
	e := ts.Unix()
	return time.Unix(e-(e%int64(secs)), 0).UTC()
}

func key(code, res string) string { return code + "|" + res }

func (s *CandleStore) onUpdate(ev eventbus.Event) {
	ip, ok := ev.Payload.(domain.InstrumentPrice)
	if !ok || ip.Price <= 0 {
		return
	}
	ts := ip.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	s.mu.Lock()
	for _, br := range baseResolutions {
		k := key(ip.Code, br.Name)
		b := bucketStart(ts, br.Secs)
		a := s.current[k]
		if a == nil || !a.bucket.Equal(b) {
			if a != nil && !a.bucket.After(b) {
				// Bucket rolled over → the previous candle is final; queue it.
				s.pending = append(s.pending, a.toModel(ip.Code, br.Name))
			}
			a = &candleAgg{bucket: b, open: ip.Price, high: ip.Price, low: ip.Price}
			s.current[k] = a
		}
		a.close = ip.Price
		if ip.Price > a.high {
			a.high = ip.Price
		}
		if ip.Price < a.low {
			a.low = ip.Price
		}
		a.count++
	}
	// Sparkline ring (mini charts): one sample per material update, capped.
	buf := s.spark[ip.Code]
	buf = append(buf, ip.Price)
	if len(buf) > sparkCap {
		buf = buf[len(buf)-sparkCap:]
	}
	s.spark[ip.Code] = buf
	s.mu.Unlock()
}

// Start launches the periodic DB flush + retention loops. No-op when repo is nil.
func (s *CandleStore) Start(ctx context.Context) {
	if s.repo == nil {
		return
	}
	s.startOnce.Do(func() {
		s.workers.Add(2)
		go func() {
			defer s.workers.Done()
			s.flushLoop(ctx)
		}()
		go func() {
			defer s.workers.Done()
			s.retentionLoop(ctx)
		}()
	})
}

// flushLoop persists closed candles (immediately queued on rollover) and the
// in-progress candles (so a restart keeps the forming bar) every few seconds.
func (s *CandleStore) flushLoop(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			s.flush(context.Background())
			return
		case <-t.C:
			s.flush(ctx)
		}
	}
}

func (s *CandleStore) flush(ctx context.Context) {
	s.mu.Lock()
	closed := s.pending
	s.pending = nil
	batch := append([]models.Candle(nil), closed...)
	// Also persist current (forming) candles so history survives a restart.
	for k, a := range s.current {
		code, res := splitKey(k)
		batch = append(batch, a.toModel(code, res))
	}
	s.mu.Unlock()
	if len(batch) == 0 {
		return
	}
	// Every replica observes largely the same market stream and flushes the
	// same rows. Go map iteration is random, so without a canonical order each
	// transaction acquires PostgreSQL conflict-row locks differently and can
	// deadlock another replica. A total key order makes lock acquisition
	// deterministic cluster-wide.
	sort.Slice(batch, func(i, j int) bool {
		if batch[i].Code != batch[j].Code {
			return batch[i].Code < batch[j].Code
		}
		if batch[i].Resolution != batch[j].Resolution {
			return batch[i].Resolution < batch[j].Resolution
		}
		return batch[i].Bucket.Before(batch[j].Bucket)
	})
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := s.repo.UpsertBatch(cctx, batch); err != nil {
		// Forming candles are included again automatically on the next flush.
		// Closed candles are not, so put only those back at the front of the
		// pending queue to avoid a history gap after transient DB contention.
		if len(closed) > 0 {
			s.mu.Lock()
			s.pending = append(closed, s.pending...)
			s.mu.Unlock()
		}
		s.log.Warn("candle flush failed", zap.Error(err), zap.Int("candles", len(batch)))
	}
}

func splitKey(k string) (code, res string) {
	for i := len(k) - 1; i >= 0; i-- {
		if k[i] == '|' {
			return k[:i], k[i+1:]
		}
	}
	return k, ""
}

// retentionLoop trims raw 1m candles (kept 14 days); higher resolutions are tiny
// and kept indefinitely for long-range charts.
func (s *CandleStore) retentionLoop(ctx context.Context) {
	t := time.NewTicker(6 * time.Hour)
	defer t.Stop()
	run := func() {
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		if n, err := s.repo.Prune(cctx, "1m", time.Now().Add(-14*24*time.Hour)); err == nil && n > 0 {
			s.log.Info("pruned 1m candles", zap.Int64("rows", n))
		}
		if n, err := s.repo.Prune(cctx, "5m", time.Now().Add(-60*24*time.Hour)); err == nil && n > 0 {
			s.log.Info("pruned 5m candles", zap.Int64("rows", n))
		}
	}
	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}

// SetRepo attaches the DB repository. Call once at startup before Start().
func (s *CandleStore) SetRepo(r repository.CandleRepository) { s.repo = r }

// bootstrapLookback bounds how far back each resolution is aggregated from ticks.
// Fine resolutions stay recent (huge tick counts); coarse ones go back far to
// capture the historical daily backfill for long-range charts.
var bootstrapLookback = map[string]time.Duration{
	"1m":  2 * 24 * time.Hour,
	"5m":  60 * 24 * time.Hour,
	"15m": 180 * 24 * time.Hour,
	"1h":  2 * 365 * 24 * time.Hour,
	"4h":  5 * 365 * 24 * time.Hour,
	"1d":  40 * 365 * 24 * time.Hour, // back to ~1999 backfill
}

// Bootstrap aggregates existing price_ticks into candles for any resolution that
// has none yet, so charts have history immediately (including backfilled rows).
// Runs in the background; safe to call once at startup.
func (s *CandleStore) Bootstrap(ctx context.Context) {
	if s.repo == nil {
		return
	}
	s.bootstrapOnce.Do(func() {
		s.workers.Add(1)
		go func() {
			defer s.workers.Done()
			// Single-flight across pods: only the lock holder runs the heavy scans.
			locked, release, err := s.repo.TryLockBootstrap(ctx)
			if err != nil {
				s.log.Warn("candle bootstrap lock failed", zap.Error(err))
				return
			}
			if !locked {
				s.log.Info("candle bootstrap skipped (another pod holds the lock)")
				return
			}
			defer release()

			now := time.Now().UTC()
			codes := make([]string, 0, len(domain.Catalog))
			for _, instrument := range domain.Catalog {
				codes = append(codes, instrument.Code)
			}
			for _, br := range baseResolutions {
				// Look for real history (older than 3 bar-widths ago) so the live
				// forming candle just flushed to the DB doesn't skip the bootstrap.
				cutoff := now.Add(-3 * time.Duration(br.Secs) * time.Second)
				missing, err := s.repo.MissingHistoryCodes(ctx, br.Name, cutoff, codes)
				if err != nil {
					s.log.Warn("candle bootstrap check failed", zap.String("res", br.Name), zap.Error(err))
					continue
				}
				if len(missing) == 0 {
					continue
				}
				from := now.Add(-bootstrapLookback[br.Name])
				cctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
				n, err := s.repo.SeedFromTicks(cctx, br.Name, br.Secs, from, now, missing)
				cancel()
				if err != nil {
					s.log.Warn("candle bootstrap failed",
						zap.String("res", br.Name),
						zap.Int("missing_codes", len(missing)),
						zap.Error(err))
					continue
				}
				s.log.Info("candle bootstrap done",
					zap.String("res", br.Name),
					zap.Int("missing_codes", len(missing)),
					zap.Int64("rows", n))
			}
		}()
	})
}

// Wait blocks until bootstrap, flush and retention workers have exited. Call it
// after canceling the context passed to Start/Bootstrap and before closing the
// database, so no shutdown flush races a closed connection pool.
func (s *CandleStore) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.workers.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// liveCandle returns a snapshot copy of the forming candle for (code,res).
func (s *CandleStore) liveCandle(code, res string) (models.OHLC, bool) {
	s.mu.RLock()
	a := s.current[key(code, res)]
	s.mu.RUnlock()
	if a == nil {
		return models.OHLC{}, false
	}
	return a.toOHLC(), true
}

// Candles returns OHLC candles for a requested interval, merging durable history
// (indexed candle range read) with the live forming candle. Millisecond-class.
func (s *CandleStore) Candles(ctx context.Context, code, interval string, from, to time.Time, limit int) ([]models.OHLC, error) {
	m, ok := intervalToBase[interval]
	if !ok {
		return nil, ErrUnsupportedInterval
	}
	if limit <= 0 {
		limit = 1000
	}
	if limit > 2000 {
		limit = 2000
	}
	var base []models.OHLC
	if s.repo != nil {
		// Fetch enough base candles to cover the requested (coarser) window.
		fetch := limit * m.Factor
		if fetch <= 0 {
			fetch = 1000
		}
		rows, err := s.repo.Range(ctx, code, m.Base, from, to, fetch)
		if err != nil {
			return nil, err
		}
		base = rows
	}
	// Merge the live forming candle for the base resolution.
	if live, ok := s.liveCandle(code, m.Base); ok {
		if to.IsZero() || !live.Bucket.After(to) {
			if n := len(base); n > 0 && base[n-1].Bucket.Equal(live.Bucket) {
				base[n-1] = live
			} else if n == 0 || live.Bucket.After(base[n-1].Bucket) {
				base = append(base, live)
			}
		}
	}
	if m.Factor == 1 {
		return base, nil
	}
	var out []models.OHLC
	if m.Calendar != "" {
		out = rebucketCalendar(base, m.Calendar)
	} else {
		out = rebucket(base, m.Secs*m.Factor)
	}
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

// rebucket aggregates fine base candles into coarser ones of `secs` width.
func rebucket(in []models.OHLC, secs int) []models.OHLC {
	if len(in) == 0 {
		return in
	}
	out := make([]models.OHLC, 0, len(in))
	var cur *models.OHLC
	for _, c := range in {
		b := bucketStart(c.Bucket, secs)
		if cur == nil || !cur.Bucket.Equal(b) {
			if cur != nil {
				out = append(out, *cur)
			}
			nc := models.OHLC{Bucket: b, Open: c.Open, High: c.High, Low: c.Low, Close: c.Close, Count: c.Count}
			cur = &nc
			continue
		}
		if c.High > cur.High {
			cur.High = c.High
		}
		if c.Low < cur.Low {
			cur.Low = c.Low
		}
		cur.Close = c.Close
		cur.Count += c.Count
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out
}

// rebucketCalendar aligns weekly candles to Monday 00:00 UTC and monthly
// candles to the first calendar day. Fixed Unix-duration buckets made weeks
// start on Thursday and represented a "month" as arbitrary 30-day windows.
func rebucketCalendar(in []models.OHLC, period string) []models.OHLC {
	if len(in) == 0 {
		return in
	}
	bucket := func(ts time.Time) time.Time {
		t := ts.UTC()
		switch period {
		case "week":
			day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			daysSinceMonday := (int(day.Weekday()) + 6) % 7
			return day.AddDate(0, 0, -daysSinceMonday)
		case "month":
			return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		default:
			return t
		}
	}

	out := make([]models.OHLC, 0, len(in))
	var cur *models.OHLC
	for _, c := range in {
		b := bucket(c.Bucket)
		if cur == nil || !cur.Bucket.Equal(b) {
			if cur != nil {
				out = append(out, *cur)
			}
			next := models.OHLC{
				Bucket: b,
				Open:   c.Open,
				High:   c.High,
				Low:    c.Low,
				Close:  c.Close,
				Count:  c.Count,
			}
			cur = &next
			continue
		}
		if c.High > cur.High {
			cur.High = c.High
		}
		if c.Low < cur.Low {
			cur.Low = c.Low
		}
		cur.Close = c.Close
		cur.Count += c.Count
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out
}

// Sparkline returns the recent closes for one instrument (mini chart), from
// memory. Zero DB.
func (s *CandleStore) Sparkline(code string, points int) []float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	buf := s.spark[code]
	if points > 0 && len(buf) > points {
		buf = buf[len(buf)-points:]
	}
	out := make([]float64, len(buf))
	copy(out, buf)
	return out
}

// SparklinesAll returns recent closes for every instrument, from memory.
func (s *CandleStore) SparklinesAll(points int) map[string][]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string][]float64, len(s.spark))
	for code, buf := range s.spark {
		b := buf
		if points > 0 && len(b) > points {
			b = b[len(b)-points:]
		}
		cp := make([]float64, len(b))
		copy(cp, b)
		out[code] = cp
	}
	return out
}

// DayStat is the 24h summary served from the in-memory daily candle.
type DayStat struct {
	Open, High, Low, Close float64
	Count                  int64
	OK                     bool
}

// Stats returns today's open/high/low/close for one instrument from the forming
// 1d candle — instant, no DB.
func (s *CandleStore) Stats(code string) DayStat {
	live, ok := s.liveCandle(code, "1d")
	if !ok {
		return DayStat{}
	}
	return DayStat{Open: live.Open, High: live.High, Low: live.Low, Close: live.Close, Count: live.Count, OK: true}
}

// SeedDayOpens loads day-open baselines from the candles table (fast) for the
// Istanbul day starting at `since`. Returns nil on error / no DB.
func (s *CandleStore) SeedDayOpens(ctx context.Context, since time.Time) map[string]float64 {
	if s.repo == nil {
		return nil
	}
	opens, err := s.repo.FirstOpenSince(ctx, "5m", since)
	if err != nil {
		s.log.Warn("day-open seed from candles failed", zap.Error(err))
		return nil
	}
	return opens
}
