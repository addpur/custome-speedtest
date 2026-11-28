# Custome Speedtest (custome-speedtest) ⚡

> **High-Performance In-Memory Network Speed Test Engine**  
> GitHub Repository: [github.com/addpur/custome-speedtest](https://github.com/addpur/custome-speedtest)  
> Built with Go, Fiber v2, and Uber Fx. Fully compatible with OpenSpeedTest standards for both **Browser (Web GUI)** and **Headless CLI (VM-to-VM testing)**.

---

## 🌟 Highlights

- **Pure In-Memory Buffer (Zero Disk I/O):**
  - Download and upload buffers are stored 100% in RAM (32 MB default).
  - Multiple concurrent users read from the exact same in-memory slice simultaneously without disk bottlenecks or SSD wear.
  - Upload data is read directly into memory and discarded immediately (`io.Discard`).
- **Dual Testing Modes:**
  - **Browser (Web GUI):** Modern, responsive HTML5 dashboard with dynamic gauges and real-time bandwidth graph.
  - **Headless CLI:** Test network throughput directly between servers/VMs without GUI or web browser.
- **Light & Dark Theme:**
  - One-click toggle between sleek modern dark glassmorphism and crisp light theme with preference persistence.
- **Accurate Telemetry:**
  - **Latency (Ping)** with high-resolution timers (`performance.now()`).
  - **Jitter** calculation following RFC 1889 standard.
  - **Parallel Multi-Stream Download & Upload** (1 to 16 concurrent streams).
- **Clean Architecture with Uber Fx:**
  - Decoupled modules (`speedtest`, `rest`, `fe`, `entity`) orchestrated seamlessly with unified lifecycle management in `bundlefx/bundle.go`.
- **Docker Ready:**
  - Multi-stage minimal Alpine container with health check.

---

## 🚀 Quick Start

### 1. Run Server Locally (Go)

```bash
# Clone and enter directory
cd speedtest

# Run the server
go run main.go
```
The server will start on `http://localhost:3000`.

---

### 2. Run with Docker & Docker Compose

#### Using Docker Compose (Recommended)

```bash
# Build and start container in background
docker compose up -d

# View container logs
docker compose logs -f

# Stop container
docker compose down
```

#### Using Standalone Docker

```bash
# Build Docker image
docker build -t custome-speedtest .

# Run container on port 3000
docker run -d --name custome-speedtest -p 3000:3000 custome-speedtest
```

---

## 🌐 Web Browser Usage

1. Open your browser and navigate to:
   ```
   http://<SERVER-IP>:3000
   ```
2. **Features available in Web UI:**
   - **Start Test:** Click **"START TEST"** to initiate consecutive Ping, Jitter, Download, and Upload tests.
   - **Speedometer Gauge:** Dynamic 60fps gauge that auto-scales (20, 50, 100, 500, 1000, 2500 Mbps).
   - **Real-time Chart:** Live Bezier curve plotting instantaneous transfer throughput.
   - **Theme Switcher:** Click the 🌙 / ☀️ icon in the top header to toggle between Dark Mode and Light Mode.
   - **Settings (⚙️):** Adjust test duration (5–30s), parallel streams (1–16), TCP overhead factor, or auto-run on load.
   - **History (🕒):** Review historical speed tests stored in local storage and synced to the server.
   - **Stress Mode (⚡):** Run an extended test with higher parallel streams to stress-test network links.

---

## 💻 CLI Mode: Headless Testing Between VMs

Custome Speedtest supports 100% headless terminal testing between virtual machines, containers, or remote servers without needing a browser, GUI, or even having Go installed on the client!

### Method 1: Zero-Install 1-Line cURL (Recommended for Any VM)

On any client VM (Linux/macOS) with only standard `curl` installed—**no Go, no git clone, and no project files required**:

```bash
# Auto-detects server and runs test with 4 streams, 10 seconds:
curl -sL http://192.168.1.10:3000/cli | bash

# With custom parameters (e.g. 16 parallel streams, 5 seconds):
curl -sL http://192.168.1.10:3000/cli | bash -s -- -p 16 -d 5
```

---

### Method 2: Single Standalone Binary (Zero-Dependency)

If you compile the binary once on your build machine or server:
```bash
# On server: compile single standalone static binary
go build -o speedtest main.go
```
You can copy or curl this **single file** (`speedtest`) to any client VM. The client VM **does NOT need Go or source files**:
```bash
./custome-speedtest cli -s http://192.168.1.10:3000 -p 16 -d 5
```

### CLI Command Options

```
Usage of speedtest-cli:
  -s, --server string
        Target server URL (default "http://localhost:3000")
  -d, --duration int
        Test duration per phase in seconds (default 10)
  -p, --parallel int
        Number of parallel streams (1-16) (default 4)
  --overhead float
        Overhead compensation factor (e.g. 1.05) (default 1.0)
  --json
        Output results in JSON format (ideal for scripts and CI/CD)
```

### Example CLI Output (Interactive)

```text
======================================================
  ⚡ Custome Speedtest CLI (custome-speedtest)
======================================================
  Target Server : http://192.168.1.10:3000
  Streams       : 4 parallel connections
  Duration      : 10s per phase
------------------------------------------------------
[*] Testing Latency & Jitter (10 samples)... Done.
    Latency: 1.15 ms | Jitter: 0.28 ms
[*] Testing Download Speed (4 streams, 10s)...
    Download: 948.35 Mbps (1185.44 MB transferred)
[*] Testing Upload Speed (4 streams, 10s)...
    Upload  : 932.10 Mbps (1165.12 MB transferred)
------------------------------------------------------
  🏆 TEST SUMMARY:
    Latency  : 1.15 ms
    Jitter   : 0.28 ms
    Download : 948.35 Mbps
    Upload   : 932.10 Mbps
======================================================
```

### Example CLI Output (JSON Mode)

```bash
./custome-speedtest cli -s http://192.168.1.10:3000 -d 5 -p 4 --json
```

Output:
```json
{
  "server": "http://192.168.1.10:3000",
  "ping_ms": 1.15,
  "jitter_ms": 0.28,
  "download_mbps": 948.35,
  "download_bytes": 1243000000,
  "upload_mbps": 932.10,
  "upload_bytes": 1221700000,
  "timestamp": "2026-11-25T14:55:00Z"
}
```

---

## 📡 Testing with Raw cURL (Zero Tooling Required)

If your remote VM only has `curl` installed, you can test directly against Custome Speedtest's REST endpoints:

### 1. Test Latency (Ping)
```bash
curl -w "Time: %{time_total}s | Status: %{http_code}\n" -s http://<SERVER-IP>:3000/api/v1/ping
```

### 2. Test Download Speed (e.g. 100 MB chunk)
```bash
curl -o /dev/null -w "Speed: %{speed_download} bytes/sec (%{size_download} bytes in %{time_total}s)\n" \
  "http://<SERVER-IP>:3000/api/v1/download?size=104857600"
```

### 3. Test Upload Speed (e.g. 50 MB payload)
```bash
dd if=/dev/zero bs=1M count=50 2>/dev/null | curl -X POST -H "Content-Type: application/octet-stream" \
  --data-binary @- -o /dev/null \
  -w "Speed: %{speed_upload} bytes/sec (%{size_upload} bytes in %{time_total}s)\n" \
  "http://<SERVER-IP>:3000/api/v1/upload"
```

---

## ⚙️ Configuration Reference

Configuration options can be supplied via environment variables or `.env` file:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `LISTEN_ADDR` | `:3000` | Address and port to bind HTTP server |
| `BUFFER_SIZE` | `33554432` (32MB) | Size of in-memory incompressible buffer in RAM |
| `DEFAULT_DURATION` | `10` | Default test duration in seconds |
| `DEFAULT_PARALLEL` | `4` | Default number of parallel connection streams |
| `DEFAULT_OVERHEAD` | `1.0` | Default TCP/IP overhead multiplier |
| `AUTO_RUN` | `false` | Enable automatic background tests on startup |

---

## 🏗️ Architecture & Clean Code Design

```
speedtest/
├── bundlefx/              # Uber Fx composition root & lifecycle orchestrator
│   ├── bundle.go          # Unified server lifecycle (Fiber, Scheduler, Shutdown)
│   └── entity/            # Environment and config provider
├── speedtest/             # Speedtest core engine domain
│   ├── domain/            # Interfaces, Models, Configurations
│   └── service/           # Buffer generator, Engine, Store, Registry, Scheduler
├── rest/                  # HTTP Transport Layer (Fiber v2)
│   ├── delivery/          # Route handlers (Download, Upload, Ping, Results, Stress)
│   ├── domain/            # REST DTOs and API requests/responses
│   └── service/           # API orchestration service
├── cli/                   # Headless VM-to-VM speedtest runner
│   ├── runner.go          # Multi-threaded HTTP test client & JSON exporter
│   └── runner_test.go     # In-memory unit tests
├── fe/                    # Frontend presentation layer
│   ├── modul.go           # Embedded filesystem handler (embed.FS)
│   └── static/            # Static assets
│       ├── index.html     # Semantic HTML5 layout
│       ├── style.css      # Dark & Light responsive styling
│       └── app.js         # Canvas gauge, real-time graph, multi-thread test engine
├── main.go                # Unified entrypoint (Server & CLI sub-command)
├── Dockerfile             # Multi-stage minimal Alpine production build
├── docker-compose.yml     # Container service declaration
└── go.mod                 # Dependencies
```

### Why In-Memory Buffers?
Typical speed test applications write temporary files to disk, creating bottlenecks when network links exceed disk write speeds (e.g. 1Gbps, 10Gbps). Custome Speedtest allocates a single `[]byte` buffer of 4 MB in RAM. All incoming download requests stream slices of this read-only buffer concurrently. Upload payloads are read in 32KB network buffers and drained straight to `/dev/null` (`io.Discard`). This guarantees maximum line-rate throughput without wearing out storage media.

---

## 🧪 Testing

Run comprehensive automated tests across all packages:

```bash
go test -v ./...
```
All unit tests run in-memory without requiring external network connectivity.
