package util

import (
	"fmt"
	"math"
)

func FormatHumanSize(totalSize int64) string {
	if totalSize == 0 {
		return "0 B/s"
	}
	const k float64 = 1000
	size := []string{"B", "KB", "MB", "GB"}
	i := math.Floor(math.Log(float64(totalSize)) / math.Log(k))
	return fmt.Sprintf("%.2f %s", float64(totalSize)/math.Pow(k, i), size[int(i)])
}

func FormatSpeedRate(totalSize, totalSeconds int64) string {
	if totalSeconds <= 0 {
		totalSeconds = 1
	}
	if totalSize <= 0 {
		return "0 B"
	}
	const k float64 = 1000
	size := []string{"B", "KB", "MB", "GB"}
	base := float64(totalSize) / float64(totalSeconds)
	i := math.Floor(math.Log(base) / math.Log(k))
	return fmt.Sprintf("%.2f %s/s", float64(base)/math.Pow(k, i), size[int(i)])
}
