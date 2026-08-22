// Package timeline implements the domain analysis that turns raw subtitle
// segments into quality findings: time-axis overlap, silent gaps, reading-speed
// (overspeed) violations, empty or over-long lines and descriptive-subtitle
// problems. It is a pure function of the input segments and the configured
// thresholds, so it is trivially testable and idempotent.
package timeline

// Rule identifiers raised by the analysis engine.
const (
	RuleOverlap              = "overlap"
	RuleGap                  = "gap"
	RuleOverspeed            = "overspeed"
	RuleEmpty                = "empty"
	RuleLongLine             = "long_line"
	RuleNonPositiveDuration  = "non_positive_duration"
	RuleDescriptiveMissing   = "descriptive_missing"
	RuleUnknownSpeaker       = "unknown_speaker"
)

// Severity levels for findings.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// Finding is one derived quality issue.
type Finding struct {
	SegmentID string
	Rule      string
	Severity  string
	Message   string
}
