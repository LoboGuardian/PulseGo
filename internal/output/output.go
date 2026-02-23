package output

import (
	"encoding/json"
	"fmt"
	"time"
)

type JSONOutput struct {
	Timestamp   time.Time  `json:"timestamp"`
	Download    Download   `json:"download"`
	Latency     Latency    `json:"latency"`
	Jitter      Jitter     `json:"jitter,omitempty"`
	Bufferbloat Bufferbloat `json:"bufferbloat,omitempty"`
	Health      Health     `json:"health"`
}

type Download struct {
	SpeedMbps   float64 `json:"speed_mbps"`
	BytesTotal  int64   `json:"bytes_total"`
	Duration    string  `json:"duration"`
	Connections int     `json:"connections"`
}

type Latency struct {
	TTFB    string `json:"ttfb"`
	Total   string `json:"total"`
}

type Jitter struct {
	Value       string  `json:"value"`
	Min         string  `json:"min"`
	Max         string  `json:"max"`
	PacketLoss float64 `json:"packet_loss_percent"`
}

type Bufferbloat struct {
	Severity string `json:"severity"`
	Delta    string `json:"delta"`
}

type Health struct {
	Grade  string `json:"grade"`
	Score  int    `json:"score"`
	Level  string `json:"level"`
}

func FormatJSON(downloadSpeed float64, bytes int64, duration time.Duration, connections int, latency, jitter, bbloat time.Duration, jitterLoss float64, bloatSeverity string, grade string, score int) string {
	out := JSONOutput{
		Timestamp: time.Now(),
		Download: Download{
			SpeedMbps:   downloadSpeed,
			BytesTotal:  bytes,
			Duration:    duration.Round(time.Millisecond).String(),
			Connections: connections,
		},
		Latency: Latency{
			TTFB:  latency.Round(time.Millisecond).String(),
			Total: latency.Round(time.Millisecond).String(),
		},
		Jitter: Jitter{
			Value:       jitter.Round(time.Millisecond).String(),
			PacketLoss: jitterLoss,
		},
		Bufferbloat: Bufferbloat{
			Severity: bloatSeverity,
			Delta:    bbloat.Round(time.Millisecond).String(),
		},
		Health: Health{
			Grade: grade,
			Score: score,
			Level: getLevel(grade),
		},
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data)
}

func FormatJSONSimple(mbps float64) string {
	out := map[string]float64{"download_mbps": mbps}
	data, _ := json.Marshal(out)
	return string(data)
}

func FormatPrometheus(downloadSpeed float64, latency, jitter time.Duration, score int, grade string) string {
	return fmt.Sprintf(`# HELP pulsego_download_speed Download speed in Mbps
# TYPE pulsego_download_speed gauge
pulsego_download_speed %.2f

# HELP pulsego_latency Latency in milliseconds
# TYPE pulsego_latency gauge
pulsego_latency %.2f

# HELP pulsego_jitter Jitter in milliseconds
# TYPE pulsego_jitter gauge
pulsego_jitter %.2f

# HELP pulsego_health_score Health score (0-100)
# TYPE pulsego_health_score gauge
pulsego_health_score %d

# HELP pulsego_health_grade Health grade (A=5, B=4, C=3, D=2, F=1)
# TYPE pulsego_health_grade gauge
pulsego_health_grade %d
`, downloadSpeed, float64(latency.Milliseconds()), float64(jitter.Milliseconds()), score, gradeValue(grade))
}

func getLevel(grade string) string {
	switch grade {
	case "A":
		return "Gold"
	case "B":
		return "Silver"
	case "C":
		return "Bronze"
	default:
		return "Basic"
	}
}

func gradeValue(grade string) int {
	switch grade {
	case "A":
		return 5
	case "B":
		return 4
	case "C":
		return 3
	case "D":
		return 2
	default:
		return 1
	}
}
