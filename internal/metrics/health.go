package metrics

import (
	"fmt"
	"time"
)

type HealthScore struct {
	Grade        string
	Score        int
	DownloadMbps float64
	Jitter       time.Duration
	Latency      time.Duration
	Bufferbloat  string
	Details      []string
}

func CalculateHealthScore(downloadMbps float64, jitter, latency time.Duration, bufferbloat string) *HealthScore {
	score := 0
	details := []string{}

	if downloadMbps >= 100 {
		score += 30
	} else if downloadMbps >= 50 {
		score += 20
	} else if downloadMbps >= 25 {
		score += 10
	}

	if latency < 50*time.Millisecond {
		score += 25
		details = append(details, "Excellent latency")
	} else if latency < 100*time.Millisecond {
		score += 15
		details = append(details, "Good latency")
	} else if latency < 200*time.Millisecond {
		score += 5
		details = append(details, "Moderate latency")
	} else {
		details = append(details, "High latency")
	}

	if jitter < 5*time.Millisecond {
		score += 25
		details = append(details, "Excellent jitter")
	} else if jitter < 15*time.Millisecond {
		score += 15
		details = append(details, "Acceptable jitter")
	} else if jitter < 30*time.Millisecond {
		score += 5
		details = append(details, "High jitter")
	} else {
		details = append(details, "Very high jitter")
	}

	switch bufferbloat {
	case "Low":
		score += 20
		details = append(details, "Low bufferbloat")
	case "Medium":
		score += 10
		details = append(details, "Moderate bufferbloat")
	case "High":
		score += 0
		details = append(details, "High bufferbloat")
	}

	grade := "F"
	if score >= 90 {
		grade = "A"
	} else if score >= 75 {
		grade = "B"
	} else if score >= 60 {
		grade = "C"
	} else if score >= 40 {
		grade = "D"
	}

	return &HealthScore{
		Grade:        grade,
		Score:        score,
		DownloadMbps: downloadMbps,
		Jitter:       jitter,
		Latency:      latency,
		Bufferbloat:  bufferbloat,
		Details:      details,
	}
}

func (h *HealthScore) String() string {
	return fmt.Sprintf("Grade: %s (%d/100) | Download: %.2f Mbps | Latency: %v | Jitter: %v | Bufferbloat: %s",
		h.Grade, h.Score, h.DownloadMbps, h.Latency, h.Jitter, h.Bufferbloat)
}
