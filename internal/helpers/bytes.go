package helpers

import "strconv"

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
)

// BytesToHuman преобразует байты в человекочитаемый формат
func BytesToHuman(bytes uint64) string {
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
