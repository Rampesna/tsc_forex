// wsload is a WebSocket load generator that simulates thousands of concurrent
// mobile clients connecting to the TSC Forex live price stream. It ramps up N
// connections at a controlled rate, then holds them open for a duration while
// measuring:
//
//   - connection success / failure / reconnect counts
//   - time-to-first-message per connection
//   - end-to-end delivery latency (now - envelope.timestamp), bucketed into a
//     histogram for p50/p90/p99 (requires client & server clocks to be NTP
//     synced; run on/near the server for the most accurate numbers)
//   - message + byte throughput
//
// Example (10k clients, ramped at 500/s, held for 60s):
//
//	go run ./tools/wsload -url wss://tsc-api.talhacan.com/ws -conns 10000 -rate 500 -duration 60s
//
// Run it ON the server against the NodePort to isolate backend capacity from
// client-side network limits:
//
//	go run ./tools/wsload -url ws://127.0.0.1:4700/ws -conns 10000 -rate 1000 -duration 120s
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var latencyBucketsMs = []float64{1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000}

type histogram struct {
	counts []int64 // one more than buckets (last = overflow)
	sum    int64   // total ms
	n      int64
}

func newHistogram() *histogram {
	return &histogram{counts: make([]int64, len(latencyBucketsMs)+1)}
}

func (h *histogram) observe(ms float64) {
	atomic.AddInt64(&h.n, 1)
	atomic.AddInt64(&h.sum, int64(ms))
	idx := sort.SearchFloat64s(latencyBucketsMs, ms)
	atomic.AddInt64(&h.counts[idx], 1)
}

func (h *histogram) percentile(p float64) string {
	total := atomic.LoadInt64(&h.n)
	if total == 0 {
		return "n/a"
	}
	target := int64(float64(total) * p)
	var cum int64
	for i := range h.counts {
		cum += atomic.LoadInt64(&h.counts[i])
		if cum >= target {
			if i < len(latencyBucketsMs) {
				return fmt.Sprintf("<%gms", latencyBucketsMs[i])
			}
			return fmt.Sprintf(">%gms", latencyBucketsMs[len(latencyBucketsMs)-1])
		}
	}
	return "n/a"
}

type stats struct {
	dialed      int64
	connected   int64
	dialErrors  int64
	readErrors  int64
	reconnects  int64
	messages    int64
	bytes       int64
	peak        int64
	firstMsgHst *histogram
	latHst      *histogram
}

type envelope struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	url := flag.String("url", "ws://127.0.0.1:8080/ws", "WebSocket URL")
	conns := flag.Int("conns", 10000, "number of concurrent connections")
	rate := flag.Int("rate", 500, "connections opened per second (ramp)")
	duration := flag.Duration("duration", 60*time.Second, "hold duration after ramp")
	token := flag.String("token", "", "optional JWT appended as ?token=")
	flag.Parse()

	target := *url
	if *token != "" {
		target += "?token=" + *token
	}

	st := &stats{firstMsgHst: newHistogram(), latHst: newHistogram()}

	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() { <-sig; fmt.Println("\ninterrupt — shutting down…"); cancel() }()

	// Periodic live reporter.
	go reporter(ctx, st, *conns)

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		// Small buffers keep per-connection memory low at 10k scale.
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	var wg sync.WaitGroup
	ticker := time.NewTicker(time.Second / time.Duration(max(1, *rate)))
	defer ticker.Stop()

	start := time.Now()
	for i := 0; i < *conns; i++ {
		select {
		case <-ctx.Done():
			goto hold
		case <-ticker.C:
		}
		wg.Add(1)
		go runClient(ctx, &wg, dialer, target, st)
	}
	fmt.Printf("ramp complete: %d clients dialed in %s\n", *conns, time.Since(start).Round(time.Millisecond))

hold:
	// Hold connections open for the configured duration, then stop.
	select {
	case <-time.After(*duration):
	case <-ctx.Done():
	}
	cancel()
	wg.Wait()
	printSummary(st, time.Since(start))
}

func runClient(ctx context.Context, wg *sync.WaitGroup, d websocket.Dialer, url string, st *stats) {
	defer wg.Done()
	atomic.AddInt64(&st.dialed, 1)
	conn, _, err := d.DialContext(ctx, url, nil)
	if err != nil {
		atomic.AddInt64(&st.dialErrors, 1)
		return
	}
	atomic.AddInt64(&st.connected, 1)
	defer func() {
		_ = conn.Close()
		atomic.AddInt64(&st.connected, -1)
	}()

	dialedAt := time.Now()
	first := true
	conn.SetReadLimit(1 << 20)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			atomic.AddInt64(&st.readErrors, 1)
			return
		}
		atomic.AddInt64(&st.messages, 1)
		atomic.AddInt64(&st.bytes, int64(len(data)))
		if first {
			st.firstMsgHst.observe(float64(time.Since(dialedAt).Milliseconds()))
			first = false
		}
		var e envelope
		if json.Unmarshal(data, &e) == nil && !e.Timestamp.IsZero() {
			lat := time.Since(e.Timestamp).Milliseconds()
			if lat >= 0 && lat < 60000 { // ignore absurd skew
				st.latHst.observe(float64(lat))
			}
		}
	}
}

func reporter(ctx context.Context, st *stats, target int) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	var lastMsgs int64
	last := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			now := time.Now()
			if live := atomic.LoadInt64(&st.connected); live > atomic.LoadInt64(&st.peak) {
				atomic.StoreInt64(&st.peak, live)
			}
			msgs := atomic.LoadInt64(&st.messages)
			rate := float64(msgs-lastMsgs) / now.Sub(last).Seconds()
			lastMsgs, last = msgs, now
			fmt.Printf("[%s] live=%d/%d dialErr=%d readErr=%d msgs=%d (%.0f/s) p50=%s p99=%s\n",
				time.Now().Format("15:04:05"),
				atomic.LoadInt64(&st.connected), target,
				atomic.LoadInt64(&st.dialErrors), atomic.LoadInt64(&st.readErrors),
				msgs, rate,
				st.latHst.percentile(0.50), st.latHst.percentile(0.99),
			)
		}
	}
}

func printSummary(st *stats, elapsed time.Duration) {
	fmt.Println("\n──────── STRESS TEST SUMMARY ────────")
	fmt.Printf("elapsed:        %s\n", elapsed.Round(time.Second))
	fmt.Printf("dialed:         %d\n", atomic.LoadInt64(&st.dialed))
	fmt.Printf("peak connected: %d\n", atomic.LoadInt64(&st.peak))
	fmt.Printf("dial errors:    %d\n", atomic.LoadInt64(&st.dialErrors))
	fmt.Printf("read errors:    %d\n", atomic.LoadInt64(&st.readErrors))
	fmt.Printf("messages:       %d\n", atomic.LoadInt64(&st.messages))
	fmt.Printf("bytes:          %.1f MB\n", float64(atomic.LoadInt64(&st.bytes))/1e6)
	fmt.Printf("first-msg p50:  %s  p99: %s\n", st.firstMsgHst.percentile(0.50), st.firstMsgHst.percentile(0.99))
	if n := atomic.LoadInt64(&st.latHst.n); n > 0 {
		fmt.Printf("delivery lat:   avg=%.0fms p50=%s p90=%s p99=%s (n=%d)\n",
			float64(atomic.LoadInt64(&st.latHst.sum))/float64(n),
			st.latHst.percentile(0.50), st.latHst.percentile(0.90), st.latHst.percentile(0.99), n)
	}
	fmt.Println("─────────────────────────────────────")
}
