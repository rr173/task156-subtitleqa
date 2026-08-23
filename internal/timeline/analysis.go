package timeline

import (
	"fmt"
	"sort"
	"strings"

	"task156-subtitleqa/internal/config"
	"task156-subtitleqa/internal/model"
)

// Analyze derives findings from a media's segments. validSpeakers maps the ids
// of speakers that genuinely belong to the media; a segment referencing any
// other speaker id is reported as unknown_speaker. The function is a pure
// transform, so calling it repeatedly with the same input yields the same
// findings.
func Analyze(segs []model.Segment, cfg config.Thresholds, validSpeakers map[string]bool) []Finding {
	var findings []Finding
	for _, seg := range segs {
		findings = append(findings, analyzeOne(seg, cfg, validSpeakers)...)
	}

	ordered := make([]model.Segment, len(segs))
	copy(ordered, segs)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].StartMs != ordered[j].StartMs {
			return ordered[i].StartMs < ordered[j].StartMs
		}
		return ordered[i].Index < ordered[j].Index
	})

	for i := 1; i < len(ordered); i++ {
		prev, cur := ordered[i-1], ordered[i]
		if prev.EndMs <= prev.StartMs {
			continue // already flagged as non-positive duration
		}
		if cur.StartMs < prev.EndMs-cfg.OverlapToleranceMs {
			overlap := prev.EndMs - cur.StartMs
			findings = append(findings, Finding{
				SegmentID: cur.ID,
				Rule:      RuleOverlap,
				Severity:  SeverityError,
				Message:   fmt.Sprintf("segment #%d overlaps previous by %d ms", cur.Index, overlap),
			})
		} else if gap := cur.StartMs - prev.EndMs; gap > cfg.GapThresholdMs {
			findings = append(findings, Finding{
				SegmentID: cur.ID,
				Rule:      RuleGap,
				Severity:  SeverityInfo,
				Message:   fmt.Sprintf("segment #%d starts %d ms after previous end (silent gap)", cur.Index, gap),
			})
		}
	}
	return findings
}

func analyzeOne(seg model.Segment, cfg config.Thresholds, validSpeakers map[string]bool) []Finding {
	var out []Finding
	// A segment that names a speaker not registered for the media is an
	// attribution anomaly: the subtitle references a marker that does not
	// belong to the asset. Report it explicitly rather than silently passing.
	if seg.SpeakerID != "" && !validSpeakers[seg.SpeakerID] {
		out = append(out, Finding{
			SegmentID: seg.ID,
			Rule:      RuleUnknownSpeaker,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("segment #%d references unknown speaker %q", seg.Index, seg.SpeakerID),
		})
	}
	if seg.EndMs <= seg.StartMs {
		out = append(out, Finding{
			SegmentID: seg.ID,
			Rule:      RuleNonPositiveDuration,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("segment #%d has non-positive duration (%d ms)", seg.Index, seg.EndMs-seg.StartMs),
		})
		return out
	}
	text := strings.TrimSpace(seg.Text)
	runes := RuneCount(seg.Text)
	if text == "" {
		if seg.IsDescriptive {
			out = append(out, Finding{
				SegmentID: seg.ID,
				Rule:      RuleDescriptiveMissing,
				Severity:  SeverityError,
				Message:   fmt.Sprintf("descriptive segment #%d has no text", seg.Index),
			})
		} else {
			out = append(out, Finding{
				SegmentID: seg.ID,
				Rule:      RuleEmpty,
				Severity:  SeverityError,
				Message:   fmt.Sprintf("segment #%d is empty", seg.Index),
			})
		}
		return out
	}
	if runes > cfg.MaxSegmentChars {
		out = append(out, Finding{
			SegmentID: seg.ID,
			Rule:      RuleLongLine,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("segment #%d has %d chars (max %d)", seg.Index, runes, cfg.MaxSegmentChars),
		})
	}
	if cps, ok := ReadingSpeedCPS(seg.Text, seg.StartMs, seg.EndMs); ok && cps > cfg.MaxCPS {
		out = append(out, Finding{
			SegmentID: seg.ID,
			Rule:      RuleOverspeed,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("segment #%d reads at %.1f cps (max %.1f)", seg.Index, cps, cfg.MaxCPS),
		})
	}
	return out
}

// Summarize counts findings by rule and severity for quick dashboards.
func Summarize(findings []Finding) map[string]int {
	byRule := map[string]int{}
	for _, f := range findings {
		byRule[f.Rule]++
	}
	return byRule
}
