// Package config holds the tunable quality thresholds used by the timeline and
// QA analysis. They are intentionally centralised so the same numbers drive the
// CLI, the HTTP API and the smoke test.
package config

// Thresholds controls when the analysis engine raises findings.
type Thresholds struct {
	// MaxCPS is the maximum comfortable reading speed (characters per second).
	// Above this a segment is flagged as overspeed.
	MaxCPS float64
	// MaxSegmentChars is the maximum rune count for a single subtitle line.
	MaxSegmentChars int
	// GapThresholdMs: a silence longer than this between consecutive segments is a gap.
	GapThresholdMs int64
	// OverlapToleranceMs: end/start overlap below this is tolerated (always 0 here).
	OverlapToleranceMs int64
}

// Default returns the production thresholds.
func Default() Thresholds {
	return Thresholds{
		MaxCPS:            12.0,
		MaxSegmentChars:  84,
		GapThresholdMs:   1000,
		OverlapToleranceMs: 0,
	}
}
