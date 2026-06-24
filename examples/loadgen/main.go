package main

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type config struct {
	TargetURL string        `env:"TARGET_URL" env-default:"http://localhost:8080/api/hello"`
	RPS       int           `env:"RPS" env-default:"10"`
	Workers   int           `env:"WORKERS" env-default:"1"`
	Duration  time.Duration `env:"DURATION" env-default:"30s"`
}

type stats struct {
	total      atomic.Uint64
	success    atomic.Uint64
	fail       atomic.Uint64
	latencySum atomic.Uint64
	latencyMin atomic.Uint64
	latencyMax atomic.Uint64
}

func main() {
	var cfg config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("loadgen: target=%s rps=%d workers=%d duration=%s\n",
		cfg.TargetURL, cfg.RPS, cfg.Workers, cfg.Duration)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var s stats
	s.latencyMin.Store(math.MaxUint64)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	rate := time.Duration(float64(time.Second) / float64(cfg.RPS/cfg.Workers))

	for range cfg.Workers {
		go worker(ctx, client, cfg.TargetURL, rate, &s)
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			printStats(&s, true)
			return
		case <-ticker.C:
			printStats(&s, false)
		}
	}
}

func worker(ctx context.Context, client *http.Client, url string, rate time.Duration, s *stats) {
	ticker := time.NewTicker(rate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			doRequest(client, url, s)
		}
	}
}

func doRequest(client *http.Client, url string, s *stats) {
	start := time.Now()
	resp, err := client.Get(url)
	elapsed := time.Since(start)

	s.total.Add(1)
	latency := uint64(elapsed.Microseconds())
	s.latencySum.Add(latency)

	for {
		old := s.latencyMin.Load()
		if latency >= old || s.latencyMin.CompareAndSwap(old, latency) {
			break
		}
	}
	for {
		old := s.latencyMax.Load()
		if latency <= old || s.latencyMax.CompareAndSwap(old, latency) {
			break
		}
	}

	if err != nil {
		s.fail.Add(1)
		return
	}
	_ = resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.success.Add(1)
	} else {
		s.fail.Add(1)
	}
}

func printStats(s *stats, final bool) {
	total := s.total.Load()
	success := s.success.Load()
	fail := s.fail.Load()
	lMin := s.latencyMin.Load()
	lMax := s.latencyMax.Load()
	lSum := s.latencySum.Load()

	if total == 0 {
		if final {
			fmt.Println("no requests sent")
		}
		return
	}

	prefix := ""
	if final {
		prefix = "=== FINAL "
	}

	var latAvg uint64
	if total > 0 {
		latAvg = lSum / total
	}

	fmt.Printf("%sSTATS: total=%d success=%d fail=%d success_rate=%.1f%% lat_min=%dms lat_max=%dms lat_avg=%dms\n",
		prefix,
		total, success, fail,
		float64(success)/float64(total)*100,
		lMin/1000, lMax/1000, latAvg/1000,
	)
}
