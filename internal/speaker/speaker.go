// Package speaker provides speaker-marker helpers: building the set of valid
// speaker ids for a media, computing per-speaker on-screen coverage and finding
// segments that reference a speaker not registered for the media.
package speaker

import "task156-subtitleqa/internal/model"

// ValidSet returns the ids of speakers that belong to the media.
func ValidSet(speakers []model.Speaker) map[string]bool {
	set := make(map[string]bool, len(speakers))
	for _, sp := range speakers {
		set[sp.ID] = true
	}
	return set
}

// Coverage maps each speaker id to its total on-screen duration (ms) across the
// supplied segments. Segments with non-positive duration are ignored.
func Coverage(segs []model.Segment, speakers []model.Speaker) map[string]int64 {
	out := make(map[string]int64, len(speakers))
	for _, sp := range speakers {
		out[sp.ID] = 0
	}
	for _, seg := range segs {
		if seg.SpeakerID == "" {
			continue
		}
		d := seg.EndMs - seg.StartMs
		if d <= 0 {
			continue
		}
		out[seg.SpeakerID] += d
	}
	return out
}

// OrphanSegments returns segments whose speaker_id is set but is not in the
// media's registered speaker set.
func OrphanSegments(segs []model.Segment, speakers []model.Speaker) []model.Segment {
	valid := ValidSet(speakers)
	var out []model.Segment
	for _, seg := range segs {
		if seg.SpeakerID != "" && !valid[seg.SpeakerID] {
			out = append(out, seg)
		}
	}
	return out
}

// LabelOf resolves a speaker id to its display label, falling back to the id.
func LabelOf(speakers []model.Speaker, id string) string {
	for _, sp := range speakers {
		if sp.ID == id {
			return sp.Label
		}
	}
	return id
}
