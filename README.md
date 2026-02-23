# PulseGo: Network Health & Performance Monitoring in Go

PulseGo is a high-performance network diagnostic tool that measures the "pulse" of your connection. While others only show speed, PulseGo analyzes Jitter, Latency, and Packet Stability to grade your network for real-world tasks like Gaming, 4K Streaming, and VoIP.

## Why PulseGo?

Most speedtests fail to account for micro-latency. PulseGo uses the power of **Go Goroutines** to efficiently saturate bandwidth while measuring packet consistency.

### Key Features

- **QoE Analysis:** Automatic network classification (e.g., Gaming: Grade A, Streaming: Grade B)
- **Precision Metrics:** Jitter, Latency to First Byte (TTFB), and packet loss measurement
- **Cloud Native & DevOps Friendly:** Native JSON output for Prometheus, InfluxDB, or Grafana integration
- **Zero Dependencies:** Lightweight static binary; runs on any architecture (Linux, macOS, Windows, Raspberry Pi)
- **Multi-threaded Testing:** Optimized concurrency engine for high-speed fiber connections

## Installation

If you have Go installed:

```bash
go install github.com/LoboGuardian/pulsego@latest
```

Or download the binary from the Releases section.

## Usage

### Basic (Human-readable)

```bash
pulsego run --simple
```

> **Output:** `Download: 300 Mbps | Upload: 150 Mbps | Jitter: 2ms | Grade: Gaming Ready`

### Advanced (DevOps/JSON)

```bash
pulsego run --format=json > metrics.json
```

## Project Structure

The project follows a clean, modular structure:

- `/internal/engine`: Core engine managing concurrent requests
- `/internal/metrics`: Mathematical algorithms for network health calculation
- `/cmd/pulsego`: Command-line interface (CLI)

## Makefile Commands

- `make build` - Build the binary
- `make run` - Run the application
- `make test` - Run tests
- `make clean` - Clean build artifacts
- `make install` - Install binary
- `make fmt` - Format code
- `make vet` - Run code analysis

## Contributing

Contributions are welcome! If you have ideas to improve the saturation engine or want to add a new data exporter, open an Issue or send a Pull Request.

## License

MIT License - See LICENSE file for details.
