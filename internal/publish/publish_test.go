package publish

import (
	"testing"

	"task156-subtitleqa/internal/model"
)

func TestCompareReportsChangedSubtitleFieldsAndAddedSegments(t *testing.T) {
	media := model.Media{ID: "media-1", Title: "访谈"}
	beforeJSON, err := BuildSnapshot(media, nil, []model.Segment{{ID: "s1", Index: 1, StartMs: 0, EndMs: 1000, Text: "原文"}})
	if err != nil {
		t.Fatal(err)
	}
	afterJSON, err := BuildSnapshot(media, nil, []model.Segment{
		{ID: "s1", Index: 1, StartMs: 0, EndMs: 1200, Text: "修订文"},
		{ID: "s2", Index: 2, StartMs: 1200, EndMs: 2000, Text: "新增行"},
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := ParseSnapshot(beforeJSON)
	if err != nil {
		t.Fatal(err)
	}
	after, err := ParseSnapshot(afterJSON)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := Compare(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Added) != 1 || diff.Added[0] != "s2" {
		t.Fatalf("added=%v, want s2", diff.Added)
	}
	if len(diff.Changes) < 2 {
		t.Fatalf("changes=%v, want timing and text changes", diff.Changes)
	}
}
