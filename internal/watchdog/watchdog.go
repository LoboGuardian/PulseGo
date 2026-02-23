package watchdog

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/LoboGuardian/pulsego/internal/metrics"
)

type Config struct {
	URL              string
	Interval         time.Duration
	JitterSamples    int
	JitterInterval   time.Duration
	JitterThreshold  time.Duration
	LatencyThreshold time.Duration
	LossThreshold    float64
	GamingMode       bool
}

type Stats struct {
	mu            sync.RWMutex
	Samples       int
	LatencyMin    time.Duration
	LatencyMax    time.Duration
	LatencySum    time.Duration
	JitterMin     time.Duration
	JitterMax     time.Duration
	JitterSum     time.Duration
	LossSum       float64
	LatencyAlerts int
	JitterAlerts  int
	LossAlerts    int
	GradeCounts   map[string]int
}

type Alert struct {
	Type      string
	Value     interface{}
	Threshold interface{}
	Timestamp time.Time
}

type Watcher struct {
	Config    Config
	Stats     *Stats
	Alerts    []Alert
	alertsMu  sync.Mutex
	running   bool
	runningMu sync.Mutex
	stopChan  chan struct{}
}

func NewWatcher(cfg Config) *Watcher {
	return &Watcher{
		Config: cfg,
		Stats: &Stats{
			GradeCounts: make(map[string]int),
		},
		Alerts:   make([]Alert, 0),
		stopChan: make(chan struct{}),
	}
}

func (w *Watcher) Start(ctx context.Context) error {
	w.runningMu.Lock()
	w.running = true
	w.runningMu.Unlock()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(w.Config.Interval)
	defer ticker.Stop()

	fmt.Printf("\033[2J\033[H")
	fmt.Println("PulseGo Watchdog - Network Monitoring")
	fmt.Println("=====================================")
	fmt.Printf("Interval: %v | Target: %s\n", w.Config.Interval, w.Config.URL)
	if w.Config.GamingMode {
		fmt.Println("Mode: Gaming (latency-focused, no bandwidth saturation)")
	}
	fmt.Println("Press Ctrl+C to stop and see summary")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sigChan:
			return nil
		case <-w.stopChan:
			return nil
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Watcher) Stop() {
	w.runningMu.Lock()
	defer w.runningMu.Unlock()
	if w.running {
		w.running = false
		close(w.stopChan)
	}
}

func (w *Watcher) tick(ctx context.Context) {
	timestamp := time.Now()
	latencyResult, err := metrics.MeasureLatency(ctx, w.Config.URL)
	if err != nil {
		fmt.Printf("\r\033[K[%s] Error: %v\n", timestamp.Format("15:04:05"), err)
		return
	}

	var jitterResult *metrics.JitterResult
	if w.Config.JitterSamples > 0 {
		jitterResult, _ = metrics.MeasureJitter(ctx, w.Config.URL, w.Config.JitterSamples, w.Config.JitterInterval)
	}

	var jitter time.Duration
	var loss float64
	if jitterResult != nil {
		jitter = jitterResult.Jitter
		loss = jitterResult.PacketLoss
	}

	health := metrics.CalculateHealthScore(0, jitter, latencyResult.Latency, "Unknown")

	w.updateStats(latencyResult.Latency, jitter, loss, health.Grade)

	alerts := w.checkAlerts(latencyResult.Latency, jitter, loss)
	for _, alert := range alerts {
		w.addAlert(alert)
	}

	w.printLine(timestamp, latencyResult.Latency, jitter, loss, health.Grade, len(alerts) > 0)
}

func (w *Watcher) updateStats(latency, jitter time.Duration, loss float64, grade string) {
	w.Stats.mu.Lock()
	defer w.Stats.mu.Unlock()

	w.Stats.Samples++

	if w.Stats.Samples == 1 || latency < w.Stats.LatencyMin {
		w.Stats.LatencyMin = latency
	}
	if latency > w.Stats.LatencyMax {
		w.Stats.LatencyMax = latency
	}
	w.Stats.LatencySum += latency

	if jitter > 0 {
		if w.Stats.Samples == 1 || jitter < w.Stats.JitterMin {
			w.Stats.JitterMin = jitter
		}
		if jitter > w.Stats.JitterMax {
			w.Stats.JitterMax = jitter
		}
		w.Stats.JitterSum += jitter
	}

	w.Stats.LossSum += loss
	w.Stats.GradeCounts[grade]++
}

func (w *Watcher) checkAlerts(latency, jitter time.Duration, loss float64) []Alert {
	alerts := []Alert{}
	now := time.Now()

	if w.Config.LatencyThreshold > 0 && latency > w.Config.LatencyThreshold {
		alerts = append(alerts, Alert{
			Type:      "latency",
			Value:     latency,
			Threshold: w.Config.LatencyThreshold,
			Timestamp: now,
		})
		w.Stats.mu.Lock()
		w.Stats.LatencyAlerts++
		w.Stats.mu.Unlock()
	}

	if w.Config.JitterThreshold > 0 && jitter > w.Config.JitterThreshold {
		alerts = append(alerts, Alert{
			Type:      "jitter",
			Value:     jitter,
			Threshold: w.Config.JitterThreshold,
			Timestamp: now,
		})
		w.Stats.mu.Lock()
		w.Stats.JitterAlerts++
		w.Stats.mu.Unlock()
	}

	if w.Config.LossThreshold > 0 && loss > w.Config.LossThreshold {
		alerts = append(alerts, Alert{
			Type:      "loss",
			Value:     loss,
			Threshold: w.Config.LossThreshold,
			Timestamp: now,
		})
		w.Stats.mu.Lock()
		w.Stats.LossAlerts++
		w.Stats.mu.Unlock()
	}

	return alerts
}

func (w *Watcher) addAlert(alert Alert) {
	w.alertsMu.Lock()
	defer w.alertsMu.Unlock()
	w.Alerts = append(w.Alerts, alert)

	maxAlerts := 100
	if len(w.Alerts) > maxAlerts {
		w.Alerts = w.Alerts[len(w.Alerts)-maxAlerts:]
	}
}

func (w *Watcher) printLine(ts time.Time, latency, jitter time.Duration, loss float64, grade string, hasAlert bool) {
	alertMarker := " "
	if hasAlert {
		alertMarker = "!"
	}

	jitterStr := "--"
	if jitter > 0 {
		jitterStr = fmt.Sprintf("%v", jitter.Round(time.Millisecond))
	}

	lossStr := "--"
	if loss >= 0 {
		lossStr = fmt.Sprintf("%.1f%%", loss)
	}

	gradeColor := gradeColor(grade)
	fmt.Printf("\r\033[K[%s] %s Lat: %-8v Jitter: %-8v Loss: %-6s %s%s\033[0m",
		ts.Format("15:04:05"),
		alertMarker,
		latency.Round(time.Millisecond),
		jitterStr,
		lossStr,
		gradeColor,
		grade,
	)
}

func gradeColor(grade string) string {
	switch grade {
	case "A":
		return "\033[32m"
	case "B":
		return "\033[36m"
	case "C":
		return "\033[33m"
	case "D", "F":
		return "\033[31m"
	default:
		return "\033[0m"
	}
}

func (w *Watcher) PrintSummary() {
	w.Stats.mu.RLock()
	defer w.Stats.mu.RUnlock()

	fmt.Println("\n\nSummary")
	fmt.Println("=======")
	fmt.Printf("Samples: %d | Duration: ~%v\n", w.Stats.Samples, time.Duration(w.Stats.Samples)*w.Config.Interval)

	if w.Stats.Samples > 0 {
		avgLatency := w.Stats.LatencySum / time.Duration(w.Stats.Samples)
		fmt.Printf("\nLatency:\n")
		fmt.Printf("  Min: %v | Max: %v | Avg: %v\n",
			w.Stats.LatencyMin.Round(time.Millisecond),
			w.Stats.LatencyMax.Round(time.Millisecond),
			avgLatency.Round(time.Millisecond))
	}

	if w.Stats.JitterSum > 0 {
		samplesWithJitter := w.Stats.Samples
		avgJitter := w.Stats.JitterSum / time.Duration(samplesWithJitter)
		fmt.Printf("\nJitter:\n")
		fmt.Printf("  Min: %v | Max: %v | Avg: %v\n",
			w.Stats.JitterMin.Round(time.Millisecond),
			w.Stats.JitterMax.Round(time.Millisecond),
			avgJitter.Round(time.Millisecond))
	}

	if w.Stats.Samples > 0 {
		avgLoss := w.Stats.LossSum / float64(w.Stats.Samples)
		fmt.Printf("\nPacket Loss:\n")
		fmt.Printf("  Avg: %.2f%%\n", avgLoss)
	}

	fmt.Printf("\nGrade Distribution:\n")
	for _, g := range []string{"A", "B", "C", "D", "F"} {
		count := w.Stats.GradeCounts[g]
		if count > 0 {
			bar := ""
			for i := 0; i < count && i < 20; i++ {
				bar += "="
			}
			fmt.Printf("  %s: %d %s\033[0m\n", gradeColor(g), count, bar)
		}
	}

	totalAlerts := w.Stats.LatencyAlerts + w.Stats.JitterAlerts + w.Stats.LossAlerts
	if totalAlerts > 0 {
		fmt.Printf("\nAlerts:\n")
		fmt.Printf("  Latency: %d | Jitter: %d | Loss: %d | Total: %d\n",
			w.Stats.LatencyAlerts, w.Stats.JitterAlerts, w.Stats.LossAlerts, totalAlerts)
	}
}
