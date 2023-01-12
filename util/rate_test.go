package util

import "testing"

func TestFormatHumanSize(t *testing.T) {
	t.Log(FormatHumanSize(1024))
	t.Log(FormatHumanSize(1024 * 1024))
	t.Log(FormatHumanSize(1024 * 1024 * 1024))

	t.Log(FormatSpeedRate(1024, 1))
	t.Log(FormatSpeedRate(1024*1024, 2))
	t.Log(FormatSpeedRate(1024*1024*1024, 3))
}
