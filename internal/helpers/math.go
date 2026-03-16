package helpers

import "math"

// BytesToMB конвертирует байты в мегабайты
func BytesToMB(bytes uint64) uint64 {
	return bytes / MB
}

// BytesToGB конвертирует байты в гигабайты
func BytesToGB(bytes uint64) uint64 {
	return bytes / GB
}

// RoundPrecision округляет число до указанной точности
func RoundPrecision(num float64, precision uint) float32 {
	ratio := math.Pow(10, float64(precision))
	return float32(math.Round(num*ratio) / ratio)
}
