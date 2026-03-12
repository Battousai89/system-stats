package utils

import "math"

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
)

func BytesToMB(bytes uint64) uint64 {
	return bytes / MB
}

func BytesToGB(bytes uint64) uint64 {
	return bytes / GB
}

func RoundPrecision(num float64, precision uint) float32 {
	ratio := math.Pow(10, float64(precision))
	return float32(math.Round(num*ratio) / ratio)
}
