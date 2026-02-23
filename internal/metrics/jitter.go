package metrics

import (
	"context"
	"math"
	"net/http"
	"sort"
	"time"
)

type JitterResult struct {
	Jitter        time.Duration
	MinLatency   time.Duration
	MaxLatency   time.Duration
	AvgLatency   time.Duration
	PacketLoss   float64
	Samples      int
}

func MeasureJitter(ctx context.Context, url string, samples int, interval time.Duration) (*JitterResult, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	latencies := make([]time.Duration, 0, samples)

	for i := 0; i < samples; i++ {
		start := time.Now()
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		_, err = client.Do(req)
		if err != nil {
			continue
		}

		latency := time.Since(start)
		latencies = append(latencies, latency)

		if i < samples-1 {
			select {
			case <-ctx.Done():
				break
			case <-time.After(interval):
			}
		}
	}

	if len(latencies) < 2 {
		return &JitterResult{
			Jitter:      0,
			Samples:     len(latencies),
			PacketLoss:  float64(samples-len(latencies)) / float64(samples) * 100,
		}, nil
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}
	avgLatency := sum / time.Duration(len(latencies))

	var varianceSum float64
	for i := 1; i < len(latencies); i++ {
		diff := float64(latencies[i] - latencies[i-1])
		varianceSum += diff * diff
	}
	jitter := time.Duration(math.Sqrt(varianceSum / float64(len(latencies)-1)))

	return &JitterResult{
		Jitter:      jitter,
		MinLatency:  latencies[0],
		MaxLatency:  latencies[len(latencies)-1],
		AvgLatency:  avgLatency,
		Samples:     len(latencies),
		PacketLoss:  float64(samples-len(latencies)) / float64(samples) * 100,
	}, nil
}
