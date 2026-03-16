# System Stats

A fast, parallel system statistics collector written in Go.

## Features

- **⚡ High Performance** - Optimized with caching, merged queries, and parallel execution
- **Parallel collection** - All stats are collected concurrently using goroutines
- **Multiple output formats** - JSON or human-readable list format
- **Multi-mode support** - Combine multiple stat categories in a single run
- **Benchmark mode** - Measure collection time for each stat category
- **Configurable timeouts** - Control external command execution timeouts
- **Cross-platform** - Supports Linux and Windows
- **Docker support** - Enhanced Docker detection with automatic path discovery

## Performance Optimizations

The collector uses several optimization techniques:

| Optimization | Description | Impact |
|--------------|-------------|--------|
| **Query Merging** | Combines multiple PowerShell queries into single calls | ~50% faster for HostInfo |
| **Intelligent Caching** | Caches static data (disk info, memory, Docker status) with TTL | Up to 99% faster for repeated calls |
| **Parallel Execution** | All stat categories run concurrently | Near-linear speedup on multi-core systems |
| **Batch Docker Operations** | Single `docker inspect` for all containers | ~70x faster for Docker stats |
| **Smart Retry Logic** | Automatic re-check of Docker daemon availability | Resilient to Docker restarts |

### Benchmark Results (AMD Ryzen 5 3600)

| Category | Time |
|----------|------|
| CPU Info | ~1.4s |
| Host Info | ~0.4s |
| Memory | ~105µs (cached) |
| Disk Info | ~1.5s (cached) |
| Docker Stats | ~44ns (cached) |
| Process Info | ~1.2s |
| GPU Info | ~0.3s |

> **Note:** Actual performance varies depending on hardware and system configuration. Linux implementations typically show faster performance due to direct `/proc` and `/sys` filesystem access.

## Installation

```bash
go build -o system-stats ./cmd/system-stats
```

## Usage

### Basic Usage

```bash
# All stats in JSON format (default)
./system-stats

# All stats in list format
./system-stats -mode all -format list

# Specific category
./system-stats -mode cpu
./system-stats -mode mem
./system-stats -mode disk
./system-stats -mode net
./system-stats -mode gpu
```

### Multi-Mode

Combine multiple categories:

```bash
./system-stats -mode cpu,mem,disk
./system-stats -mode host,cpu,mem,net
```

### Output Formats

```bash
# JSON output (default)
./system-stats -format json

# Human-readable list format
./system-stats -format list

# Save to file
./system-stats -out stats.json
```

### Benchmark Mode

Measure collection time for each category:

```bash
./system-stats -mode all -benchmark -format json
```

Example output:
```json
{
  "benchmark": {
    "cpu": "101ms",
    "disk": "16.9ms",
    "mem": "105µs",
    "net": "21.3ms",
    "process": "88.5ms",
    "total": "107.6ms"
  }
}
```

## Available Modes

| Mode | Description |
|------|-------------|
| `all` | All categories |
| `host` | Hostname, uptime, OS info, kernel version, virtualization |
| `cpu` | CPU info, times, and usage percentage |
| `mem` | Memory usage (total, available, used, free) |
| `swap` | Swap devices and usage |
| `disk` | Disk usage and I/O counters |
| `diskinfo` | Disk device information (serial, model, type) |
| `net` | Network I/O and interfaces |
| `netproto` | Network protocol counters (TCP, UDP, IP) |
| `gpu` | GPU info (name, vendor, memory, temperature) |
| `sensor` | Hardware sensors (temperatures) |
| `battery` | Battery status and capacity |
| `proc` | Top processes by CPU usage |
| `docker` | Docker container stats |
| `virt` | Virtualization detection |
| `loadmisc` | Load average and process stats |

## Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-mode` | `all` | Stats mode (comma-separated for multiple) |
| `-format` | `json` | Output format: `json` or `list` |
| `-out` | - | Output file path |
| `-timeout` | `5s` | Timeout for external commands |
| `-cpu-sampling` | `100ms` | Sampling interval for CPU percent |
| `-top-procs` | `10` | Number of top processes to show |
| `-benchmark` | `false` | Show timing metrics |

## Examples

### Get CPU and Memory stats
```bash
./system-stats -mode cpu,mem -format list
```

### Save disk info to file
```bash
./system-stats -mode disk,diskinfo -out disk-stats.json
```

### Run with custom timeout
```bash
./system-stats -timeout 10s -mode docker
```

### Benchmark specific categories
```bash
./system-stats -mode cpu,mem,disk -benchmark
```

## JSON Output Structure

```json
{
  "Mode": "all",
  "HostInfo": { ... },
  "CPUInfo": [ ... ],
  "CPUTimes": [ ... ],
  "CPUPercent": [ ... ],
  "Memory": { ... },
  "DiskUsage": { ... },
  "NetIO": [ ... ],
  "GPUs": [ ... ],
  "Processes": [ ... ],
  "BenchmarkInfo": { ... },
  "CollectedStats": ["all"]
}
```

## Architecture

The collector uses a fully parallel architecture with intelligent caching:

```
collectAll()
├── Host, LoadMisc, Virt (parallel)
├── CPU (parallel internally, merged queries)
├── Memory, Swap (parallel)
├── Disk, DiskInfo (parallel, cached)
├── Net, NetProto (parallel)
├── Sensors, Battery, Process, GPU, Docker (all parallel)
└── Results aggregation with benchmark timing
```

### Caching Strategy

| Data Type | Cache TTL | Reason |
|-----------|-----------|--------|
| Docker path & availability | 5s | Daemon may restart |
| Disk device info | 30s | Hardware doesn't change often |
| Total memory | 60s | System memory is static |
| CPU info | Per-call | Clock speed may change |

### Platform Implementations

| Feature | Linux | Windows |
|---------|-------|---------|
| Host Info | `/proc`, `/sys`, `uname` | WMI, PowerShell |
| CPU Info | `/proc/cpuinfo`, `/sys` | WMI, PowerShell |
| Memory | `/proc/meminfo` | WMI, PowerShell |
| Disk Info | `/proc/diskstats`, `/sys/block` | WMI, PowerShell |
| Network | `/proc/net`, `netstat` | WMI, PowerShell |
| GPU | `lspci`, DRM sysfs, `nvidia-smi` | WMI, PowerShell |
| Docker | Docker CLI | Docker CLI |
| Sensors | `/sys/class/hwmon`, thermal zones | WMI, PowerShell |
| Battery | `/sys/class/power_supply` | WMI, PowerShell |
| Virtualization | DMI, `/sys/class/dmi` | WMI, PowerShell |

### Docker Detection

The collector automatically finds Docker in:
- System PATH
- Standard installation directories

On Linux, Docker is detected via standard PATH lookup. The collector checks daemon availability with caching to avoid repeated connection attempts.

## System Requirements

- **Linux**: Full support via `/proc` and `/sys` filesystems
  - Requires standard Linux utilities: `uname`, `lspci` (for GPU), `lsblk` (for disk info)
  - Docker support requires Docker CLI in PATH
- **Windows**: Support via WMI and PowerShell commands
  - Requires PowerShell 5.0+
  - Docker support requires Docker CLI in PATH

Some features require elevated privileges:
- Docker stats (requires Docker daemon access)
- Some sensor readings (may require root on Linux)
- Disk serial numbers (may require admin on Windows)
- GPU temperature monitoring (may require root/admin)

## License

MIT
