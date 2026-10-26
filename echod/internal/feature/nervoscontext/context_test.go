package nervoscontext

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

var testBinding = Binding{"echo-example", "shared-room", "operator-example"}
var testNow = time.Unix(1800000000, 0)

func sample(rev uint64) Projection {
	return Projection{1, testBinding.DeviceID, testBinding.SurfaceID, testBinding.Source, rev, testNow.Unix(), testNow.Add(time.Minute).Unix(), "shared", "The workbench is ready for the next step.", "A shared update from the room operator."}
}
func encoded(p Projection) []byte { b, _ := json.Marshal(p); return b }
func TestNervosRejectsUnsafeCandidatesRetainingGood(t *testing.T) {
	for name, change := range map[string]func(*Projection){
		"private":         func(p *Projection) { p.Visibility = "private" },
		"foreign-device":  func(p *Projection) { p.DeviceID = "other-echo" },
		"foreign-surface": func(p *Projection) { p.SurfaceID = "phone" },
		"foreign-source":  func(p *Projection) { p.Source = "other-operator" },
		"stale":           func(p *Projection) { p.ExpiresAt = testNow.Unix() },
		"future":          func(p *Projection) { p.ObservedAt = testNow.Add(time.Second).Unix() },
		"long-lease":      func(p *Projection) { p.ExpiresAt = testNow.Add(time.Hour).Unix() },
		"empty":           func(p *Projection) { p.Summary = "" },
		"long":            func(p *Projection) { p.Summary = strings.Repeat("x", 161) },
		"bidi":            func(p *Projection) { p.Summary = "hello\u202etext" },
		"control":         func(p *Projection) { p.Why = "hello\ntext" },
		"version":         func(p *Projection) { p.SchemaVersion = 2 },
	} {
		t.Run(name, func(t *testing.T) {
			s := New(testBinding, 0, nil)
			if err := s.Apply(encoded(sample(1)), testNow); err != nil {
				t.Fatal(err)
			}
			p := sample(2)
			change(&p)
			if s.Apply(encoded(p), testNow) == nil {
				t.Fatal("accepted invalid candidate")
			}
			got, ok := s.Snapshot(testNow)
			if !ok || got.SourceRevision != 1 {
				t.Fatal("last good changed")
			}
			if _, ok := s.Snapshot(testNow.Add(time.Minute)); ok {
				t.Fatal("expired last good remains visible")
			}
			if err := s.Apply(encoded(sample(2)), testNow); err != nil {
				t.Fatal("rejected candidate poisoned the revision watermark", err)
			}
		})
	}
}
func TestNervosStrictJSON(t *testing.T) {
	good := string(encoded(sample(1)))
	for _, data := range []string{good + good, good[:len(good)-1] + `,"url":"https://example.invalid"}`, good[:len(good)-1] + `,"visibility":"private"}`, `null`, `[]`, strings.Repeat(" ", MaxBytes+1), `{"summary":"unfinished"`, string([]byte{0xff})} {
		s := New(testBinding, 0, nil)
		if s.Apply([]byte(data), testNow) == nil {
			t.Fatal("accepted malformed/extra fields")
		}
	}
}
func TestNervosRevisionAndPersistence(t *testing.T) {
	persisted := uint64(0)
	fail := false
	s := New(testBinding, 5, func(r uint64) error {
		if fail {
			return errors.New("disk full")
		}
		persisted = r
		return nil
	})
	if s.Apply(encoded(sample(5)), testNow) == nil {
		t.Fatal("replayed durable revision")
	}
	if err := s.Apply(encoded(sample(6)), testNow); err != nil {
		t.Fatal(err)
	}
	if persisted != 6 {
		t.Fatal("not persisted")
	}
	fail = true
	if s.Apply(encoded(sample(7)), testNow) == nil {
		t.Fatal("published without durability")
	}
	p, ok := s.Snapshot(testNow)
	if !ok || p.SourceRevision != 6 {
		t.Fatal("partial swap")
	}
	fail = false
	if err := s.Apply(encoded(sample(8)), testNow); err != nil {
		t.Fatal(err)
	}
	if s.Apply(encoded(sample(7)), testNow) == nil {
		t.Fatal("accepted out of order")
	}
	// Reconnect/restart retains the durable watermark but does not replay a previously rendered value.
	restarted := New(testBinding, persisted, nil)
	if restarted.Apply(encoded(sample(8)), testNow) == nil {
		t.Fatal("replayed on restart")
	}
	if _, ok := s.Snapshot(testNow.Add(-time.Second)); ok {
		t.Fatal("clock rollback displayed future context")
	}
}
func TestNervosConcurrentAtomicSnapshot(t *testing.T) {
	s := New(testBinding, 0, nil)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 300; j++ {
				p, ok := s.Snapshot(testNow)
				if ok && (p.SourceRevision == 0 || p.Summary != sample(1).Summary) {
					t.Error("partial value")
				}
			}
		}()
	}
	for rev := uint64(1); rev <= 100; rev++ {
		if err := s.Apply(encoded(sample(rev)), testNow); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
}
