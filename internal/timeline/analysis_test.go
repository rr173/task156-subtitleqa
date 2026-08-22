package timeline

import (
	"testing"

	"task156-subtitleqa/internal/config"
	"task156-subtitleqa/internal/model"
)

func TestAnalyzeOverlapGapOverspeed(t *testing.T) {
	cfg := config.Default()
	segs := []model.Segment{
		{ID: "s1", Index: 1, StartMs: 0, EndMs: 3000, Text: "第一行字幕"},
		{ID: "s2", Index: 2, StartMs: 2500, EndMs: 6000, Text: "与上一行重叠"},
		{ID: "s3", Index: 3, StartMs: 9000, EndMs: 10000, Text: "三十个字符的超长超速示例行文字用来触发警告"},
	}
	findings := Analyze(segs, cfg, map[string]bool{})
	got := map[string]bool{}
	for _, f := range findings {
		got[f.Rule] = true
	}
	if !got[RuleOverlap] {
		t.Error("expected overlap finding")
	}
	if !got[RuleGap] {
		t.Error("expected gap finding between 6000 and 9000")
	}
	if !got[RuleOverspeed] {
		t.Error("expected overspeed finding")
	}
}

func TestAnalyzeEmptyAndUnknownSpeaker(t *testing.T) {
	cfg := config.Default()
	segs := []model.Segment{
		{ID: "s1", Index: 1, StartMs: 0, EndMs: 2000, Text: "   ", SpeakerID: "ghost"},
	}
	findings := Analyze(segs, cfg, map[string]bool{"ghost": false})
	var empty, unknown bool
	for _, f := range findings {
		if f.Rule == RuleEmpty {
			empty = true
		}
		if f.Rule == RuleUnknownSpeaker {
			unknown = true
		}
	}
	if !empty || !unknown {
		t.Errorf("expected empty and unknown_speaker findings, got %+v", findings)
	}
}

func TestReadingSpeedCPS(t *testing.T) {
	cps, ok := ReadingSpeedCPS("十二个字", 0, 1000)
	if !ok || cps != 4 {
		t.Errorf("expected 4 cps, got %v ok=%v", cps, ok)
	}
	if _, ok := ReadingSpeedCPS("x", 1000, 1000); ok {
		t.Error("expected false for non-positive duration")
	}
}
