package metrics

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"time"
)

type LatencyResult struct {
	TTFB          time.Duration
	Latency       time.Duration
	Connected     time.Duration
	TLSHandshake  time.Duration
	Error         error
}

func MeasureLatency(ctx context.Context, url string) (*LatencyResult, error) {
	start := time.Now()
	var ttfb, connected, tlsHandshake time.Duration

	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			connected = time.Since(start)
		},
	TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			tlsHandshake = time.Since(start)
		},
		GotFirstResponseByte: func() {
			ttfb = time.Since(start)
		},
	}

	req, err := http.NewRequestWithContext(
		httptrace.WithClientTrace(ctx, trace),
		"GET", url, nil,
	)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &LatencyResult{
		TTFB:          ttfb,
		Latency:       time.Since(start),
		Connected:     connected,
		TLSHandshake:  tlsHandshake,
	}, nil
}

func FormatLatency(r *LatencyResult) string {
	if r.Error != nil {
		return fmt.Sprintf("Error: %v", r.Error)
	}
	return fmt.Sprintf("TTFB: %v | Latency: %v", r.TTFB, r.Latency)
}
