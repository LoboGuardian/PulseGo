package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/LoboGuardian/pulsego/internal/engine"
	"github.com/LoboGuardian/pulsego/internal/metrics"
	"github.com/LoboGuardian/pulsego/internal/output"
)

var (
	simple    = flag.Bool("simple", false, "Simple output for humans")
	format    = flag.String("format", "text", "Output format: text, json, prometheus")
	url       = flag.String("url", "http://speedtest.tele2.net/10MB.zip", "URL for speed test")
	downloads = flag.Int("downloads", 4, "Number of simultaneous connections")
	timeout   = flag.Duration("timeout", 120*time.Second, "Timeout per download")
	jitter    = flag.Bool("jitter", true, "Measure jitter")
	bbloat    = flag.Bool("bufferbloat", true, "Measure bufferbloat")
	stress    = flag.Bool("stress", false, "Stress mode (high concurrency)")
	p2p       = flag.String("p2p", "", "P2P mode: comma-separated list of URLs")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout*3)
	defer cancel()

	if *format == "text" {
		fmt.Println("PulseGo - Network Health Monitor")
		fmt.Println("==================================")
	}

	if *p2p != "" {
		runP2P(ctx)
		return
	}

	latencyResult, _ := metrics.MeasureLatency(ctx, *url)
	if *format == "text" && latencyResult != nil {
		fmt.Printf("Latency: %v (TTFB: %v)\n", latencyResult.Latency, latencyResult.TTFB)
	}

	engineCfg := engine.Config{
		URL:        *url,
		Downloads:  *downloads,
		Timeout:    *timeout,
		StressMode: *stress,
	}

	if *format == "text" {
		if *stress {
			fmt.Printf("Stress test (%d connections)...\n", *downloads)
		} else {
			fmt.Printf("Downloading (%d connections)...\n", *downloads)
		}
	}
	result, err := engine.Run(ctx, engineCfg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if *simple {
		fmt.Printf("%.2f Mbps\n", result.DownloadSpeed)
		os.Exit(0)
	}

	var jitterResult *metrics.JitterResult
	var bbResult *metrics.BufferbloatResult

	if *jitter && !*stress {
		if *format == "text" {
			fmt.Println("\nMeasuring Jitter...")
		}
		jitterResult, _ = metrics.MeasureJitter(ctx, *url, 10, 200*time.Millisecond)
	}

	if *bbloat && !*stress {
		if *format == "text" {
			fmt.Println("\nMeasuring Bufferbloat...")
		}
		bbResult, _ = metrics.MeasureBufferbloat(ctx, *url)
	}

	var bloatStr string
	if bbResult != nil {
		bloatStr = bbResult.Severity
	} else {
		bloatStr = "Unknown"
	}

	var jitterDur time.Duration
	var jitterLoss float64
	if jitterResult != nil {
		jitterDur = jitterResult.Jitter
		jitterLoss = jitterResult.PacketLoss
	}

	health := metrics.CalculateHealthScore(result.DownloadSpeed, jitterDur, latencyResult.Latency, bloatStr)

	switch *format {
	case "json":
		fmt.Println(output.FormatJSON(
			result.DownloadSpeed,
			result.BytesReceived,
			result.Duration,
			result.Connections,
			latencyResult.Latency,
			jitterDur,
			bbResult.BloatDelta,
			jitterLoss,
			bloatStr,
			health.Grade,
			health.Score,
		))
	case "prometheus":
		fmt.Print(output.FormatPrometheus(
			result.DownloadSpeed,
			latencyResult.Latency,
			jitterDur,
			health.Score,
			health.Grade,
		))
	default:
		fmt.Printf("Download: %.2f Mbps | %.2f MB in %v\n",
			result.DownloadSpeed,
			float64(result.BytesReceived)/1_000_000,
			result.Duration,
		)
		if *stress {
			fmt.Printf("Connections: %d | Peak: %.2f Mbps | Errors: %d\n",
				result.Connections, result.PeakSpeed, result.Errors)
		}
		if jitterResult != nil {
			fmt.Printf("Jitter: %v | Min: %v | Max: %v | Loss: %.1f%%\n",
				jitterResult.Jitter, jitterResult.MinLatency, jitterResult.MaxLatency, jitterResult.PacketLoss)
		}
		if bbResult != nil {
			fmt.Printf("Bufferbloat: %s (Delta %v)\n", bbResult.Severity, bbResult.BloatDelta)
		}
		fmt.Println("\n" + health.String())
	}
}

func runP2P(ctx context.Context) {
	targets := strings.Split(*p2p, ",")
	for i := range targets {
		targets[i] = strings.TrimSpace(targets[i])
	}

	fmt.Printf("P2P test with %d nodes...\n", len(targets))

	result, err := engine.RunP2P(ctx, targets, *timeout)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if *simple {
		fmt.Printf("%.2f Mbps\n", result.DownloadSpeed)
		return
	}

	fmt.Printf("P2P Download: %.2f Mbps | %.2f MB in %v\n",
		result.DownloadSpeed,
		float64(result.BytesReceived)/1_000_000,
		result.Duration,
	)
	fmt.Printf("Nodes: %d | Errors: %d\n", result.Connections, result.Errors)
}
