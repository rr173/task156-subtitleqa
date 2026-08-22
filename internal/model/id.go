package model

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewID builds a process-unique identifier with a readable prefix, a nanosecond
// timestamp and random entropy. It is used for primary keys across every entity
// so callers never have to invent identifiers.
func NewID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand should never fail; fall back to timestamp entropy.
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixNano(), hex.EncodeToString(buf))
}
