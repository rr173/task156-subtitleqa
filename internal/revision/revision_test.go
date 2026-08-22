package revision

import (
	"strings"
	"testing"

	"task156-subtitleqa/internal/model"
)

func TestPatchAppliesOnMatchingVersion(t *testing.T) {
	seg := model.Segment{ID: "s1", MediaID: "m1", Version: 1, Text: "旧文本", StartMs: 0, EndMs: 2000}
	text := "新文本"
	patched, rev, ok := Patch(seg, model.EditSegmentRequest{Actor: "a", BaseVersion: 1, Text: &text})
	if !ok {
		t.Fatal("expected ok")
	}
	if patched.Version != 2 || patched.Text != "新文本" {
		t.Errorf("unexpected patched: %+v", patched)
	}
	if rev.BaseVersion != 1 || rev.Actor != "a" {
		t.Errorf("unexpected revision: %+v", rev)
	}
}

func TestPatchRejectsStaleVersion(t *testing.T) {
	seg := model.Segment{ID: "s1", Version: 3, Text: "当前"}
	text := "stale"
	_, _, ok := Patch(seg, model.EditSegmentRequest{Actor: "b", BaseVersion: 1, Text: &text})
	if ok {
		t.Fatal("expected rejection for stale base version")
	}
}

func TestDescribe(t *testing.T) {
	before := model.Segment{StartMs: 0, EndMs: 2000, Text: "a", SpeakerID: "s1"}
	after := model.Segment{StartMs: 0, EndMs: 3000, Text: "b", SpeakerID: "s1"}
	d := Describe(before, after)
	if !strings.Contains(d, "timing") || !strings.Contains(d, "text changed") {
		t.Errorf("unexpected description: %q", d)
	}
}
