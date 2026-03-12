package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"system-stats/internal/config"
	"system-stats/internal/services"
)

var modes = map[string]services.Mode{
	"all":      services.ModeAll,
	"cpu":      services.ModeCPU,
	"mem":      services.ModeMem,
	"disk":     services.ModeDisk,
	"net":      services.ModeNet,
	"host":     services.ModeHost,
	"sensor":   services.ModeSensor,
	"battery":  services.ModeBattery,
	"proc":     services.ModeProcess,
	"gpu":      services.ModeGPU,
	"docker":   services.ModeDocker,
	"virt":     services.ModeVirt,
	"swap":     services.ModeSwap,
	"netproto": services.ModeNetProto,
	"diskinfo": services.ModeDiskInfo,
	"loadmisc": services.ModeLoadMisc,
}

func main() {
	modeFlag := flag.String("mode", "all", "Stats mode (comma-separated for multiple: cpu,mem,disk)")
	formatFlag := flag.String("format", "json", "Output format: json, list")
	outFlag := flag.String("out", "", "Output file path")
	timeoutFlag := flag.Duration("timeout", 5*time.Second, "Timeout for external commands")
	cpuSamplingFlag := flag.Duration("cpu-sampling", 100*time.Millisecond, "Sampling interval for CPU percent calculation")
	topProcsFlag := flag.Int("top-procs", 10, "Number of top processes to show")
	benchmarkFlag := flag.Bool("benchmark", false, "Show timing metrics for each stat collection")
	flag.Parse()

	// Apply configuration
	config.CommandTimeout = *timeoutFlag
	config.CPUSamplingInterval = *cpuSamplingFlag
	config.TopProcessesCount = *topProcsFlag
	config.BenchmarkMode = *benchmarkFlag

	modeStrings := strings.Split(strings.ToLower(*modeFlag), ",")
	modeMap := make(map[services.Mode]bool)

	for _, m := range modeStrings {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		mode, ok := modes[m]
		if !ok {
			fmt.Printf("Unknown mode: %s\n", m)
			os.Exit(1)
		}
		modeMap[mode] = true
	}

	if modeMap[services.ModeAll] {
		modeMap = map[services.Mode]bool{services.ModeAll: true}
	}

	services.OutputFormat = strings.ToLower(*formatFlag)

	stats, err := services.NewSystemStatsFromMap(modeMap)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	output := stats.ToPrint()

	if *outFlag != "" {
		err := os.WriteFile(*outFlag, []byte(output), 0644)
		if err != nil {
			fmt.Printf("Error writing to file: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println(output)
	}
}
