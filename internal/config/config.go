package config

import (
	"time"
)

var (
	// CommandTimeout is the timeout for external commands
	CommandTimeout = 5 * time.Second

	// CPUSamplingInterval is the delay between CPU samples for percent calculation
	CPUSamplingInterval = 100 * time.Millisecond

	// TopProcessesCount is the number of top processes to show
	TopProcessesCount = 10

	// BenchmarkMode enables timing metrics output
	BenchmarkMode = false
)
