// Package httpapi exposes the JSON HTTP interface of the subtitle proofreading
// workbench. Every route lives under /api; the root serves the embedded web UI.
// Handlers are thin: validation and business rules live in the service layer.
package httpapi

import (
	"net/http"

	"task156-subtitleqa/internal/demo"
	"task156-subtitleqa/internal/metrics"
	"task156-subtitleqa/internal/service"
	"task156-subtitleqa/internal/webui"
)

// API wires the service, metrics and router together.
type API struct {
	svc     *service.Service
	metrics *metrics.Metrics
	mux     *http.ServeMux
}

// New builds an API over a service.
func New(svc *service.Service, m *metrics.Metrics) *API {
	a := &API{svc: svc, metrics: m, mux: http.NewServeMux()}
	a.routes()
	return a
}

// ServeHTTP implements http.Handler and counts every request.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.metrics.Request()
	a.mux.ServeHTTP(w, r)
}

func (a *API) routes() {
	a.mux.Handle("/", webui.Handler())
	a.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	a.mux.HandleFunc("GET /readyz", a.ready)
	a.mux.HandleFunc("GET /api/metrics", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, a.metrics.Snapshot())
	})
	a.mux.HandleFunc("GET /api/stats", a.stats)

	a.mux.HandleFunc("POST /api/media", a.createMedia)
	a.mux.HandleFunc("GET /api/media", a.listMedia)
	a.mux.HandleFunc("GET /api/media/{id}", a.getMedia)

	a.mux.HandleFunc("POST /api/media/{id}/speakers", a.createSpeaker)
	a.mux.HandleFunc("GET /api/media/{id}/speakers", a.listSpeakers)

	a.mux.HandleFunc("POST /api/media/{id}/segments", a.importSegments)
	a.mux.HandleFunc("GET /api/media/{id}/segments", a.listSegments)
	a.mux.HandleFunc("GET /api/segments/{id}", a.getSegment)
	a.mux.HandleFunc("GET /api/segments/{id}/revisions", a.segmentRevisions)
	a.mux.HandleFunc("POST /api/segments/{id}/edit", a.editSegment)

	a.mux.HandleFunc("GET /api/media/{id}/quality", a.quality)
	a.mux.HandleFunc("GET /api/media/{id}/quality/summary", a.qualitySummary)
	a.mux.HandleFunc("POST /api/media/{id}/quality/recompute", a.recomputeQuality)

	a.mux.HandleFunc("POST /api/media/{id}/publish", a.publish)
	a.mux.HandleFunc("GET /api/media/{id}/versions", a.listVersions)
	a.mux.HandleFunc("GET /api/media/{id}/conflicts", a.listConflicts)
	a.mux.HandleFunc("GET /api/versions/{id}", a.getVersion)
	a.mux.HandleFunc("POST /api/versions/{id}/withdraw", a.withdraw)
	a.mux.HandleFunc("GET /api/versions/compare", a.compareVersions)

	a.mux.HandleFunc("POST /api/demo", a.seedDemo)
}

func (a *API) ready(w http.ResponseWriter, r *http.Request) {
	if err := a.svc.Recover(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *API) stats(w http.ResponseWriter, r *http.Request) {
	data, err := a.svc.Counts(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (a *API) createMedia(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title      string  `json:"title"`
		Language   string  `json:"language"`
		DurationMs int64   `json:"duration_ms"`
		FrameRate  float64 `json:"frame_rate"`
		SourceURL  string  `json:"source_url"`
		Checksum   string  `json:"checksum"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	item, err := a.svc.CreateMedia(r.Context(), toCreateMedia(req))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) listMedia(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.ListMedia(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) getMedia(w http.ResponseWriter, r *http.Request) {
	item, err := a.svc.GetMedia(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) createSpeaker(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Label string `json:"label"`
		Color string `json:"color"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	item, err := a.svc.CreateSpeaker(r.Context(), r.PathValue("id"), toCreateSpeaker(req))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) listSpeakers(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.ListSpeakers(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) importSegments(w http.ResponseWriter, r *http.Request) {
	var inputs []struct {
		Index         int    `json:"index"`
		StartMs       int64  `json:"start_ms"`
		EndMs         int64  `json:"end_ms"`
		Text          string `json:"text"`
		SpeakerID     string `json:"speaker_id"`
		IsDescriptive bool   `json:"is_descriptive"`
		Language      string `json:"language"`
	}
	if err := decode(r, &inputs); err != nil {
		writeError(w, err)
		return
	}
	items, err := a.svc.ImportSegments(r.Context(), r.PathValue("id"), toSegmentInputs(inputs))
	if err != nil {
		writeError(w, err)
		return
	}
	a.metrics.SampleWrite()
	a.metrics.Analysis()
	writeJSON(w, http.StatusCreated, items)
}

func (a *API) listSegments(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.ListSegments(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) getSegment(w http.ResponseWriter, r *http.Request) {
	item, err := a.svc.GetSegment(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) segmentRevisions(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.Revisions(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) editSegment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Actor         string `json:"actor"`
		BaseVersion   int    `json:"base_version"`
		StartMs       *int64 `json:"start_ms"`
		EndMs         *int64 `json:"end_ms"`
		Text          *string `json:"text"`
		SpeakerID     *string `json:"speaker_id"`
		IsDescriptive *bool  `json:"is_descriptive"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	item, conflict, err := a.svc.EditSegment(r.Context(), r.PathValue("id"), toEditRequest(req))
	if err != nil {
		if conflict != nil {
			a.metrics.Conflict()
			writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "conflict": conflict})
			return
		}
		writeError(w, err)
		return
	}
	a.metrics.Edit()
	a.metrics.Analysis()
	writeJSON(w, http.StatusOK, item)
}

func (a *API) quality(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.Quality(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) qualitySummary(w http.ResponseWriter, r *http.Request) {
	item, err := a.svc.QualitySummary(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) recomputeQuality(w http.ResponseWriter, r *http.Request) {
	if err := a.svc.RecomputeQuality(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	a.metrics.Analysis()
	writeJSON(w, http.StatusOK, map[string]string{"status": "recomputed"})
}

func (a *API) publish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Actor string `json:"actor"`
		Label string `json:"label"`
		Note  string `json:"note"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	item, err := a.svc.Publish(r.Context(), r.PathValue("id"), toPublish(req))
	if err != nil {
		writeError(w, err)
		return
	}
	a.metrics.Publish()
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) listVersions(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.ListVersions(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) getVersion(w http.ResponseWriter, r *http.Request) {
	item, err := a.svc.GetVersion(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) withdraw(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Actor  string `json:"actor"`
		Reason string `json:"reason"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := a.svc.Withdraw(r.Context(), r.PathValue("id"), toWithdraw(req)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "withdrawn"})
}

func (a *API) compareVersions(w http.ResponseWriter, r *http.Request) {
	v1 := r.URL.Query().Get("v1")
	v2 := r.URL.Query().Get("v2")
	if v1 == "" || v2 == "" {
		writeError(w, errMissingQuery("v1", "v2"))
		return
	}
	diff, err := a.svc.CompareVersions(r.Context(), v1, v2)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, diff)
}

func (a *API) listConflicts(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.ListConflicts(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) seedDemo(w http.ResponseWriter, r *http.Request) {
	result, err := demo.Seed(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
