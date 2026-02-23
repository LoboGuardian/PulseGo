package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Config struct {
	URL        string
	Downloads  int
	Timeout    time.Duration
	ChunkSize  int
	StressMode bool
}

type Result struct {
	DownloadSpeed float64
	BytesReceived int64
	Duration      time.Duration
	Connections   int
	AvgSpeed      float64
	PeakSpeed     float64
	Errors        int
}

type streamResult struct {
	bytes   int64
	speeds  []float64
	success bool
}

func Run(ctx context.Context, cfg Config) (*Result, error) {
	if cfg.StressMode {
		return runStress(ctx, cfg)
	}
	return runStandard(ctx, cfg)
}

func runStandard(ctx context.Context, cfg Config) (*Result, error) {
	client := &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.Downloads,
			MaxIdleConnsPerHost: cfg.Downloads,
		},
	}

	start := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalBytes int64
	var errors int

	download := func() {
		defer wg.Done()
		req, err := http.NewRequestWithContext(ctx, "GET", cfg.URL, nil)
		if err != nil {
			mu.Lock()
			errors++
			mu.Unlock()
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			mu.Lock()
			errors++
			mu.Unlock()
			return
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			mu.Lock()
			errors++
			mu.Unlock()
			return
		}

		mu.Lock()
		totalBytes += int64(len(data))
		mu.Unlock()
	}

	wg.Add(cfg.Downloads)
	for i := 0; i < cfg.Downloads; i++ {
		go download()
	}

	wg.Wait()
	duration := time.Since(start)

	if totalBytes == 0 {
		return nil, fmt.Errorf("no data received")
	}

	bits := float64(totalBytes * 8)
	mbps := (bits / 1_000_000) / duration.Seconds()

	return &Result{
		DownloadSpeed: mbps,
		BytesReceived: totalBytes,
		Duration:      duration,
		Connections:   cfg.Downloads,
		PeakSpeed:     mbps,
		Errors:        errors,
	}, nil
}

func runStress(ctx context.Context, cfg Config) (*Result, error) {
	connections := cfg.Downloads
	if connections < 10 {
		connections = 10
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        connections,
			MaxIdleConnsPerHost: connections,
		},
	}

	stressCtx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	start := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalBytes int64
	var errors int

	download := func() {
		defer wg.Done()
		for {
			select {
			case <-stressCtx.Done():
				return
			default:
			}

			req, err := http.NewRequestWithContext(stressCtx, "GET", cfg.URL, nil)
			if err != nil {
				mu.Lock()
				errors++
				mu.Unlock()
				return
			}

			resp, err := client.Do(req)
			if err != nil {
				mu.Lock()
				errors++
				mu.Unlock()
				continue
			}

			data, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				mu.Lock()
				errors++
				mu.Unlock()
				continue
			}

			mu.Lock()
			totalBytes += int64(len(data))
			mu.Unlock()
		}
	}

	for i := 0; i < connections; i++ {
		wg.Add(1)
		go download()
	}

	wg.Wait()
	duration := time.Since(start)

	mu.Lock()
	bits := float64(totalBytes * 8)
	avgMbps := (bits / 1_000_000) / duration.Seconds()
	bytes := totalBytes
	mu.Unlock()

	if bytes == 0 {
		return nil, fmt.Errorf("no data received")
	}

	return &Result{
		DownloadSpeed: avgMbps,
		BytesReceived: bytes,
		Duration:      duration,
		Connections:   connections,
		PeakSpeed:     avgMbps,
		Errors:        errors,
	}, nil
}

func RunP2P(ctx context.Context, targets []string, duration time.Duration) (*Result, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets specified")
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalBytes int64
	var errors int

	client := &http.Client{
		Timeout: duration,
	}

	worker := func(target string) {
		defer wg.Done()
		req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
		if err != nil {
			mu.Lock()
			errors++
			mu.Unlock()
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			mu.Lock()
			errors++
			mu.Unlock()
			return
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			mu.Lock()
			errors++
			mu.Unlock()
			return
		}

		mu.Lock()
		totalBytes += int64(len(data))
		mu.Unlock()
	}

	start := time.Now()
	wg.Add(len(targets))
	for _, target := range targets {
		go worker(target)
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}

	elapsed := time.Since(start)
	if elapsed == 0 {
		elapsed = time.Nanosecond
	}

	bits := float64(totalBytes * 8)
	mbps := (bits / 1_000_000) / elapsed.Seconds()

	return &Result{
		DownloadSpeed: mbps,
		BytesReceived: totalBytes,
		Duration:      elapsed,
		Connections:   len(targets),
		Errors:        errors,
	}, nil
}
