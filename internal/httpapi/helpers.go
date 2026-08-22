package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"task156-subtitleqa/internal/model"
)

// --- JSON plumbing -------------------------------------------------------

func decode(r *http.Request, out any) error {
	defer r.Body.Close()
	d := json.NewDecoder(io.LimitReader(r.Body, 4<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return fmt.Errorf("invalid request body: %w", model.ErrValidation)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// writeError maps domain errors to HTTP status codes.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	var field model.FieldError
	if errors.As(err, &field) || errors.Is(err, model.ErrValidation) {
		status = http.StatusBadRequest
	}
	if errors.Is(err, model.ErrConflict) || errors.Is(err, model.ErrPublished) || errors.Is(err, model.ErrWithdrawn) {
		status = http.StatusConflict
	}
	if errors.Is(err, model.ErrNotFound) || strings.Contains(err.Error(), "not found") {
		status = http.StatusNotFound
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func errMissingQuery(names ...string) error {
	return fmt.Errorf("missing query parameter(s): %s: %w", strings.Join(names, ", "), model.ErrValidation)
}

// --- DTO conversions -----------------------------------------------------

func toCreateMedia(req struct {
	Title      string  `json:"title"`
	Language   string  `json:"language"`
	DurationMs int64   `json:"duration_ms"`
	FrameRate  float64 `json:"frame_rate"`
	SourceURL  string  `json:"source_url"`
	Checksum   string  `json:"checksum"`
}) model.CreateMediaRequest {
	return model.CreateMediaRequest{
		Title:      req.Title,
		Language:   req.Language,
		DurationMs: req.DurationMs,
		FrameRate:  req.FrameRate,
		SourceURL:  req.SourceURL,
		Checksum:   req.Checksum,
	}
}

func toCreateSpeaker(req struct {
	Label string `json:"label"`
	Color string `json:"color"`
}) model.CreateSpeakerRequest {
	return model.CreateSpeakerRequest{Label: req.Label, Color: req.Color}
}

func toSegmentInputs(inputs []struct {
	Index         int    `json:"index"`
	StartMs       int64  `json:"start_ms"`
	EndMs         int64  `json:"end_ms"`
	Text          string `json:"text"`
	SpeakerID     string `json:"speaker_id"`
	IsDescriptive bool   `json:"is_descriptive"`
	Language      string `json:"language"`
}) []model.SegmentInput {
	out := make([]model.SegmentInput, 0, len(inputs))
	for _, in := range inputs {
		out = append(out, model.SegmentInput{
			Index:         in.Index,
			StartMs:       in.StartMs,
			EndMs:         in.EndMs,
			Text:          in.Text,
			SpeakerID:     in.SpeakerID,
			IsDescriptive: in.IsDescriptive,
			Language:      in.Language,
		})
	}
	return out
}

func toEditRequest(req struct {
	Actor         string `json:"actor"`
	BaseVersion   int    `json:"base_version"`
	StartMs       *int64 `json:"start_ms"`
	EndMs         *int64 `json:"end_ms"`
	Text          *string `json:"text"`
	SpeakerID     *string `json:"speaker_id"`
	IsDescriptive *bool  `json:"is_descriptive"`
}) model.EditSegmentRequest {
	return model.EditSegmentRequest{
		Actor:         req.Actor,
		BaseVersion:   req.BaseVersion,
		StartMs:       req.StartMs,
		EndMs:         req.EndMs,
		Text:          req.Text,
		SpeakerID:     req.SpeakerID,
		IsDescriptive: req.IsDescriptive,
	}
}

func toPublish(req struct {
	Actor string `json:"actor"`
	Label string `json:"label"`
	Note  string `json:"note"`
}) model.PublishRequest {
	return model.PublishRequest{Actor: req.Actor, Label: req.Label, Note: req.Note}
}

func toWithdraw(req struct {
	Actor  string `json:"actor"`
	Reason string `json:"reason"`
}) model.WithdrawRequest {
	return model.WithdrawRequest{Actor: req.Actor, Reason: req.Reason}
}
