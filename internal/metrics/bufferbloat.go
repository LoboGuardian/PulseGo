package metrics

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type BufferbloatResult struct {
	LatencyUnderLoad   time.Duration
	LatencyIdle        time.Duration
	BloatDelta         time.Duration
	Severity           string
}

func MeasureBufferbloat(ctx context.Context, url string) (*BufferbloatResult, error) {
	idleLatency, err := measureSingleLatency(ctx, url)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	client := &http.Client{Timeout: 5 * time.Second}
	
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
			client.Do(req)
		}()
	}
	wg.Wait()

	underLoadLatency, err := measureSingleLatency(ctx, url)
	if err != nil {
		return nil, err
	}

	delta := underLoadLatency - idleLatency
	severity := "Low"
	if delta > 100*time.Millisecond {
		severity = "Medium"
	}
	if delta > 300*time.Millisecond {
		severity = "High"
	}

	return &BufferbloatResult{
		LatencyUnderLoad: underLoadLatency,
		LatencyIdle:      idleLatency,
		BloatDelta:       delta,
		Severity:         severity,
	}, nil
}

func measureSingleLatency(ctx context.Context, url string) (time.Duration, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	_, err = client.Do(req)
	if err != nil {
		return 0, err
	}
	return time.Since(start), nil
}
