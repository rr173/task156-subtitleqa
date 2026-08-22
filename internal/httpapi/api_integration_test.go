package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/metrics"
	"task156-subtitleqa/internal/service"
	"task156-subtitleqa/internal/store"
)

func TestAPIImportsSegmentsAndReturnsQualitySummary(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "http.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	api := New(service.New(st), &metrics.Metrics{})
	create := httptest.NewRequest(http.MethodPost, "/api/media", bytes.NewBufferString(`{"title":"课程","duration_ms":5000}`))
	create.Header.Set("content-type", "application/json")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, create)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create media status=%d body=%s", rec.Code, rec.Body.String())
	}
	var media struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&media); err != nil {
		t.Fatal(err)
	}
	payload := `[{"index":0,"start_ms":0,"end_ms":1000,"text":""},{"index":1,"start_ms":2500,"end_ms":3500,"text":"后续字幕"}]`
	importReq := httptest.NewRequest(http.MethodPost, "/api/media/"+media.ID+"/segments", bytes.NewBufferString(payload))
	importReq.Header.Set("content-type", "application/json")
	importRec := httptest.NewRecorder()
	api.ServeHTTP(importRec, importReq)
	if importRec.Code != http.StatusCreated {
		t.Fatalf("import status=%d body=%s", importRec.Code, importRec.Body.String())
	}
	summaryReq := httptest.NewRequest(http.MethodGet, "/api/media/"+media.ID+"/quality/summary", nil).WithContext(context.Background())
	summaryRec := httptest.NewRecorder()
	api.ServeHTTP(summaryRec, summaryReq)
	if summaryRec.Code != http.StatusOK {
		t.Fatalf("summary status=%d body=%s", summaryRec.Code, summaryRec.Body.String())
	}
	var summary struct {
		Total int `json:"total"`
	}
	if err := json.NewDecoder(summaryRec.Body).Decode(&summary); err != nil {
		t.Fatal(err)
	}
	if summary.Total < 2 {
		t.Fatalf("quality total=%d, want empty-line and gap findings", summary.Total)
	}
}
