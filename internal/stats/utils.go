package stats

import (
	"context"
	"os/exec"
	"strconv"

	"system-stats/internal/config"
)

func runCommandWithTimeout(name string, arg ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, arg...)
	return cmd.Output()
}

func bytesToHuman(bytes uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return formatFloat(float64(bytes)/TB) + " TB"
	case bytes >= GB:
		return formatFloat(float64(bytes)/GB) + " GB"
	case bytes >= MB:
		return formatFloat(float64(bytes)/MB) + " MB"
	case bytes >= KB:
		return formatFloat(float64(bytes)/KB) + " KB"
	default:
		return strconv.FormatUint(bytes, 10) + " B"
	}
}

func formatFloat(f float64) string {
	if f == float64(int(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', 2, 64)
}

func formatDuration(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	mins := (seconds % 3600) / 60
	secs := seconds % 60

	if days > 0 {
		return daysStr(days, hours, mins, secs)
	} else if hours > 0 {
		return hoursStr(hours, mins, secs)
	} else if mins > 0 {
		return minsStr(mins, secs)
	}
	return strconv.FormatUint(secs, 10) + "s"
}

func daysStr(days, hours, mins, secs uint64) string {
	return strconv.FormatUint(days, 10) + "d " +
		strconv.FormatUint(hours, 10) + "h " +
		strconv.FormatUint(mins, 10) + "m " +
		strconv.FormatUint(secs, 10) + "s"
}

func hoursStr(hours, mins, secs uint64) string {
	return strconv.FormatUint(hours, 10) + "h " +
		strconv.FormatUint(mins, 10) + "m " +
		strconv.FormatUint(secs, 10) + "s"
}

func minsStr(mins, secs uint64) string {
	return strconv.FormatUint(mins, 10) + "m " +
		strconv.FormatUint(secs, 10) + "s"
}
