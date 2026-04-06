# Proc-Sentry

**Proc-Sentry** is a specialized, lightweight Prometheus exporter designed to monitor the **Top N** resource-consuming processes on Linux systems. It is built for performance, efficiency, and container awareness.

## Purpose

Standard exporters like `node_exporter` provide great system-level metrics but lack granular process details. Conversely, exporters that track _every_ process often suffer from "cardinality explosions" in Prometheus, consuming massive amounts of storage and memory.

**Proc-Sentry bridges this gap.** It intelligently scans all processes but only exports metrics for the top consumers of CPU, Memory, and Disk I/O. This gives you deep visibility into what's actually slowing down your node, without the overhead of monitoring thousands of idle processes.

## Benefits

- **Ultra Lightweight**: Written in Go, deployed as a static binary in a scratch-based Docker image (~12MB).
- **Low Overhead**: Uses ~8MB RAM and negligible CPU. Zero Disk I/O impact.
- **Container Aware**: Automatically detects `container_id` for processes running in Docker, Containerd, or Kubernetes (Cgroups v1 & v2).
- **User Context**: Resolves numeric UIDs to human-readable usernames (e.g., `root`, `nginx`, `postgres`).
- **Secure**: Capabilities-aware (`SYS_PTRACE`), read-only root filesystem, and AppArmor compatible.
- **Cardinality Controlled**: Uses a "Sort-Then-Select" union strategy across 4 dimensions (CPU, Memory, Disk Read, Disk Write), capping at `4 * TOP_N` unique time series.

## Metrics Exported

| Metric | Description | Labels |
|:-------|:------------|:-------|
| `proc_process_top_cpu_percent` | CPU usage percentage (per-core scale, 100% = 1 core) | pid, user, command, runtime, rank, hostname, container_id, ports, full_path |
| `proc_process_top_memory_bytes` | RSS memory in bytes | _(same)_ |
| `proc_process_top_disk_read_bytes` | Cumulative disk read bytes | _(same)_ |
| `proc_process_top_disk_write_bytes` | Cumulative disk write bytes | _(same)_ |
| `proc_scrape_duration_seconds` | Histogram of collection duration | |
| `proc_scrape_errors_total` | Counter of collection errors | |
| `proc_scrape_panics_total` | Counter of recovered panics | |
| `proc_processes_scraped_total` | Total processes seen per scrape | runtime |

## Architecture

```
                    +--------------+
                    |   /metrics   |  <-- Prometheus scrapes here
                    |   (HTTP)     |
                    +------+-------+
                           |
               prometheus.Collector interface
               (builds metrics on-demand from snapshot)
                           |
                    +------+-------+
                    |  Collector   |  <-- Background goroutine (every SCRAPE_INTERVAL)
                    |   Loop       |
                    +------+-------+
                           |
          +----------------+----------------+
          |                |                |
   +-----------+    +-----------+    +-----------+
   | ReadProc  |    | Top-N     |    | Port      |
   | /proc/pid |    | Selection |    | Resolve   |
   | stat,io,  |    | Union of  |    | (winners  |
   | cgroup    |    | 4 dims    |    |  only)    |
   +-----------+    +-----------+    +-----------+
```

1. **Background Collection**: A goroutine scans `/proc` at a configurable interval (default: 5s).
2. **Process Reading**: Reads `stat` (CPU ticks), `status` (UID/memory), `io` (disk), `cgroup` (container), and `exe` (full path) for all processes.
3. **CPU Delta Accounting**: Computes per-process CPU% using tick deltas against system-wide CPU time, scaled by core count to match `top` output.
4. **Top-N Selection**: Sorts 4 independent lists (CPU, Memory, DiskRead, DiskWrite) and takes the union of top N from each. At most `4 * TOP_N` unique processes.
5. **Port Resolution**: Scans `/proc/[pid]/net/tcp[6]` and matches socket inodes to file descriptors, **only for winner processes** to avoid expensive full-system scans.
6. **Metric Emission**: Implements the `prometheus.Collector` interface. Metrics are built on-demand during Prometheus scrapes from a read-locked snapshot, avoiding race conditions.

## Project Structure

```
proc-sentry/
├── main.go                  # Entry point, HTTP server, graceful shutdown
├── config/
│   └── config.go            # Environment variable parsing and validation
├── collector/
│   ├── collector.go          # Background loop, prometheus.Collector interface
│   ├── process.go            # Process struct, /proc reading
│   ├── topn.go               # Top-N union selection algorithm
│   └── ports.go              # TCP port resolution via inode matching
├── container/
│   └── detect.go             # Cgroup parsing, runtime detection (docker/k8s/containerd)
├── internal/
│   └── usermap.go            # /etc/passwd cache and UID resolution
├── cmd/
│   └── test-accuracy/
│       └── main.go           # CPU accuracy benchmark test
├── Dockerfile                # Multi-stage build (scratch runtime, ~12MB)
├── docker-compose.yml        # Docker Compose configuration
├── k8s-manifest.yaml         # Kubernetes DaemonSet + Service
└── grafana_dashboard.json    # Grafana dashboard
```

## Configuration

The exporter is configured entirely via environment variables. Invalid values cause a startup failure with a clear error message.

| Variable | Default | Description |
|:---------|:--------|:------------|
| `TOP_N` | `40` | Top processes per metric dimension. Valid range: 1-500. |
| `SCRAPE_INTERVAL` | `5s` | Collection interval (Go duration format). Valid range: 1s-5m. |
| `ENABLE_DISK_IO` | `true` | Enable/disable disk read/write metrics. |
| `ENABLE_PORTS` | `true` | Enable/disable listening port resolution. |
| `PROCFS_PATH` | `/proc` | Path to `/proc` filesystem. Use `/host/proc` in containers. |
| `PROC_HOSTNAME` | `os.Hostname()` | Override the `hostname` label value. |
| `METRICS_PORT` | `9105` | Port to serve metrics on. |

## Prometheus Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: "proc-sentry"
    scrape_interval: 15s
    static_configs:
      - targets: ["<YOUR_NODE_IP>:9105"]
```

For **Kubernetes**, use a `ServiceMonitor` if using the Prometheus Operator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: proc-sentry
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app: proc-sentry
  endpoints:
    - port: metrics
```

## Grafana Dashboard

A professional dashboard covering CPU, Memory, Disk I/O, and Container linkage is included.

1. Download `grafana_dashboard.json` from this repository.
2. Open Grafana > **Dashboards** > **New** > **Import**.
3. Upload the JSON file.
4. Select your Prometheus datasource.

## Development

### Build

```bash
go build -o proc-sentry .
```

### Run Tests

```bash
go test ./... -v
```

### Run Locally

```bash
sudo ./proc-sentry
# or with custom config:
TOP_N=20 SCRAPE_INTERVAL=10s ./proc-sentry
```

### CPU Accuracy Benchmark

```bash
go build -o proc-sentry . && sudo ./proc-sentry &
sleep 2
go run ./cmd/test-accuracy/
```

## Troubleshooting

### 1. Verify Metrics Output

```bash
curl -s localhost:9105/metrics | grep "proc_process_top_cpu_percent" | sort -nr -k 2 | head -n 5
```

### 2. Compare with `ps` (CPU)

```bash
ps -eo pid,user,comm,%cpu --sort=-%cpu | head -n 6
```

### 3. Compare with `ps` (Memory)

```bash
ps -eo pid,user,comm,rss,%mem --sort=-%mem | head -n 6
```

_(Note: `ps` reports RSS in KB, while proc-sentry reports bytes)_

### 4. Check Container Logs

```bash
docker logs proc-sentry
```

_Expected: `Starting proc-sentry on :9105 (TOP_N=40, ...)`_

### 5. AppArmor Ptrace Denied

If you see `apparmor="DENIED" operation="ptrace"` in `dmesg` or `journalctl`, add `security_opt: - apparmor=unconfined` to your `docker-compose.yml`, or `--security-opt apparmor=unconfined` to `docker run`.

### 6. Health Check

```bash
curl -s localhost:9105/health
# Expected: OK
```

## Deployment

See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed installation instructions:

- [Docker Run](DEPLOYMENT.md#2-quick-start-docker)
- [Docker Compose](DEPLOYMENT.md#3-docker-compose)
- [Kubernetes (DaemonSet)](DEPLOYMENT.md#4-kubernetes)

## License

MIT License
