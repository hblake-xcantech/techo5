//go:build nervos_echo && !dot && !spot && !panel

package nervoscontext

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/HuskerMinion/techo5/echod/internal/component"
)

// Directory is deliberately fixed. No remote endpoint or model-accessible setter is registered.
const Directory = "/data/nervos/presentation"

type Feature struct {
	mu      sync.RWMutex
	store   *Store
	binding Binding
}

var shared = &Feature{}

func Get() *Feature { return shared }
func init()         { component.Register(component.Device, shared, component.Order(37)) }

func (f *Feature) Snapshot(now time.Time) (Projection, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.store == nil {
		return Projection{}, false
	}
	return f.store.Snapshot(now)
}
func (f *Feature) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		f.refresh(Directory, time.Now())
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// Only root-owned non-symlink files inside a root-owned private directory are accepted.
// Deployment creates the directory; this feature never enables itself by creating a binding.
func trusted(info os.FileInfo, directory bool) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == 0 && info.Mode().Perm()&0077 == 0 && info.IsDir() == directory && (directory || info.Mode().IsRegular())
}
func readTrusted(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !trusted(info, false) {
		return nil, ErrRejected
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !trusted(opened, false) {
		return nil, ErrRejected
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxBytes+1))
	if err != nil || len(data) > MaxBytes {
		return nil, ErrRejected
	}
	return data, nil
}

type watermark struct {
	Binding  Binding `json:"binding"`
	Revision uint64  `json:"revision"`
}

func saveWatermark(dir string, b Binding, revision uint64) error {
	data, err := json.Marshal(watermark{b, revision})
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".revision-")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(name, filepath.Join(dir, "revision.json")); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
func (f *Feature) refresh(dir string, now time.Time) {
	info, err := os.Lstat(dir)
	if err != nil || !trusted(info, true) {
		f.mu.Lock()
		f.store = nil
		f.mu.Unlock()
		return
	}
	f.refreshFiles(func(name string) ([]byte, error) { return readTrusted(filepath.Join(dir, name)) }, func(b Binding, rev uint64) error { return saveWatermark(dir, b, rev) }, now)
}

// The reader boundary keeps file policy separate from the atomic live-update lifecycle.
// Production supplies only readTrusted above; tests use synthetic files owned by the test runner.
func (f *Feature) refreshFiles(read func(string) ([]byte, error), persist func(Binding, uint64) error, now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := read("binding.json")
	var b Binding
	if err != nil || Decode(data, &b) != nil || !b.Valid() {
		f.store = nil
		return
	}
	if f.store == nil || f.binding != b {
		var w watermark
		data, err = read("revision.json")
		// An explicit operator-created initial watermark is required; missing/corrupt state fails closed.
		if err != nil || Decode(data, &w) != nil || w.Binding != b {
			f.store = nil
			return
		}
		f.store = New(b, w.Revision, func(rev uint64) error { return persist(b, rev) })
		f.binding = b
	}
	data, err = read("projection.json")
	if err != nil {
		return
	}
	// No rejected payload or room content is logged. A subsequent valid revision recovers live.
	_ = f.store.Apply(data, now)
}

// Name exposes no room content in component diagnostics.
func (*Feature) Name() string { return "nervoscontext" }
