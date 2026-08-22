package timeline

import "unicode/utf8"

// ReadingSpeedCPS computes the characters-per-second reading speed of a segment.
// For CJK text each rune is one character; the same metric works for latin text.
// It returns false when the duration is non-positive so callers can skip it.
func ReadingSpeedCPS(text string, startMs, endMs int64) (float64, bool) {
	durationSec := float64(endMs-startMs) / 1000.0
	if durationSec <= 0 {
		return 0, false
	}
	runes := utf8.RuneCountInString(text)
	return float64(runes) / durationSec, true
}

// RuneCount returns the number of runes in s (CJK-friendly length).
func RuneCount(s string) int { return utf8.RuneCountInString(s) }
