// Package nervoscontext accepts bounded, shared-room text from a trusted local operator.
// It is not an Orbit transport, identity recognizer, or action broker.
package nervoscontext

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const MaxBytes = 4096
const MaxLifetime = 15 * time.Minute

var identifier = regexp.MustCompile(`^[a-z][a-z0-9-]{0,159}$`)
var ErrRejected = errors.New("projection rejected")

type Binding struct {
	DeviceID  string `json:"deviceId"`
	SurfaceID string `json:"surfaceId"`
	Source    string `json:"source"`
}

func (b Binding) Valid() bool {
	return identifier.MatchString(b.DeviceID) && b.SurfaceID == "shared-room" && identifier.MatchString(b.Source)
}

// Projection contains only text. No links, buttons, grants, markup or commands are interpreted.
// Source is pinned by the operator binding; the payload cannot choose another revision stream.
type Projection struct {
	SchemaVersion  int    `json:"schemaVersion"`
	DeviceID       string `json:"deviceId"`
	SurfaceID      string `json:"surfaceId"`
	Source         string `json:"source"`
	SourceRevision uint64 `json:"sourceRevision"`
	ObservedAt     int64  `json:"observedAt"`
	ExpiresAt      int64  `json:"expiresAt"`
	Visibility     string `json:"visibility"`
	Summary        string `json:"summary"`
	Why            string `json:"why"`
}

// Decode rejects unknown/duplicate keys, multiple documents and oversized input.
func Decode(data []byte, dst any) error {
	if len(data) > MaxBytes || !utf8.Valid(data) {
		return ErrRejected
	}
	d := json.NewDecoder(bytes.NewReader(data))
	first, err := d.Token()
	if err != nil || first != json.Delim('{') {
		return ErrRejected
	}
	keys := map[string]bool{}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return ErrRejected
		}
		key, ok := token.(string)
		if !ok || keys[key] {
			return ErrRejected
		}
		keys[key] = true
		var raw json.RawMessage
		if d.Decode(&raw) != nil {
			return ErrRejected
		}
	}
	if _, err := d.Token(); err != nil {
		return ErrRejected
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrRejected
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(dst) != nil {
		return ErrRejected
	}
	return nil
}

func plain(s string, max int) bool {
	if strings.TrimSpace(s) != s || s == "" || utf8.RuneCountInString(s) > max {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}

func (p Projection) valid(b Binding, now time.Time) bool {
	n := now.Unix()
	return b.Valid() && p.SchemaVersion == 1 && p.DeviceID == b.DeviceID && p.SurfaceID == b.SurfaceID && p.Source == b.Source && p.Visibility == "shared" && p.SourceRevision > 0 && p.ObservedAt > 0 && p.ObservedAt <= n && p.ExpiresAt > n && p.ExpiresAt > p.ObservedAt && p.ExpiresAt-p.ObservedAt <= int64(MaxLifetime/time.Second) && plain(p.Summary, 160) && plain(p.Why, 90)
}

// Store commits a durable high-water mark before publishing a complete immutable value.
// Rejecting a candidate never changes the last good value or its original expiry.
type Store struct {
	mu       sync.RWMutex
	binding  Binding
	revision uint64
	current  Projection
	persist  func(uint64) error
}

func New(b Binding, revision uint64, persist func(uint64) error) *Store {
	return &Store{binding: b, revision: revision, persist: persist}
}
func (s *Store) Apply(data []byte, now time.Time) error {
	var p Projection
	if Decode(data, &p) != nil || !p.valid(s.binding, now) {
		return ErrRejected
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.SourceRevision <= s.revision {
		return ErrRejected
	}
	if s.persist != nil {
		if err := s.persist(p.SourceRevision); err != nil {
			return err
		}
	}
	s.current, s.revision = p, p.SourceRevision
	return nil
}
func (s *Store) Snapshot(now time.Time) (Projection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current, s.current.SourceRevision > 0 && s.current.valid(s.binding, now)
}
