// Package metrics tracks lightweight request and operation counters for the
// health dashboard. Counters are atomics, safe for concurrent HTTP handlers.
package metrics

import "sync/atomic"

// Metrics holds the operational counters.
type Metrics struct {
	requests  atomic.Int64
	sampleWrites atomic.Int64
	analyses  atomic.Int64
	edits     atomic.Int64
	publishes atomic.Int64
	conflicts atomic.Int64
}

// Request counts an HTTP request.
func (m *Metrics) Request() { m.requests.Add(1) }

// SampleWrite counts a segment import batch.
func (m *Metrics) SampleWrite() { m.sampleWrites.Add(1) }

// Analysis counts a quality recomputation.
func (m *Metrics) Analysis() { m.analyses.Add(1) }

// Edit counts a successful segment edit.
func (m *Metrics) Edit() { m.edits.Add(1) }

// Publish counts a published version.
func (m *Metrics) Publish() { m.publishes.Add(1) }

// Conflict counts a rejected concurrent edit.
func (m *Metrics) Conflict() { m.conflicts.Add(1) }

// Snapshot returns a copy of the counters.
func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"requests":  m.requests.Load(),
		"sample_writes": m.sampleWrites.Load(),
		"analyses":  m.analyses.Load(),
		"edits":     m.edits.Load(),
		"publishes": m.publishes.Load(),
		"conflicts": m.conflicts.Load(),
	}
}
