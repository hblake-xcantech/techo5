//go:build nervos_echo && !dot && !spot && !panel

package nervoscontext

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNervosTrustedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readTrusted(link); err == nil {
		t.Fatal("symlink accepted")
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readTrusted(path); err == nil {
		t.Fatal("public-readable file accepted")
	}
	if os.Getuid() != 0 {
		os.Chmod(path, 0600)
		if _, err := readTrusted(path); err == nil {
			t.Fatal("non-root owner accepted")
		}
	}
}
func TestNervosLiveFileRevisions(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	write := func(name string, value any) {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		temp := filepath.Join(dir, "next")
		if err = os.WriteFile(temp, data, 0600); err != nil {
			t.Fatal(err)
		}
		if err = os.Rename(temp, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	write("binding.json", testBinding)
	write("revision.json", watermark{testBinding, 0})
	write("projection.json", sample(1))
	refresh := func(f *Feature) {
		f.refreshFiles(func(name string) ([]byte, error) { return os.ReadFile(filepath.Join(dir, name)) }, func(b Binding, rev uint64) error { return saveWatermark(dir, b, rev) }, testNow)
	}
	f := &Feature{}
	refresh(f)
	p, ok := f.Snapshot(testNow)
	if !ok || p.SourceRevision != 1 {
		t.Fatal("initial live revision missing")
	}
	write("projection.json", sample(2))
	refresh(f)
	p, ok = f.Snapshot(testNow)
	if !ok || p.SourceRevision != 2 {
		t.Fatal("live revision not swapped")
	}
	bad := sample(3)
	bad.Visibility = "private"
	write("projection.json", bad)
	refresh(f)
	p, ok = f.Snapshot(testNow)
	if !ok || p.SourceRevision != 2 {
		t.Fatal("last good lost")
	}
	if _, ok = f.Snapshot(testNow.Add(time.Minute)); ok {
		t.Fatal("expiry ignored")
	}
	restarted := &Feature{}
	refresh(restarted)
	if _, ok = restarted.Snapshot(testNow); ok {
		t.Fatal("replay after restart")
	}
	write("projection.json", sample(4))
	refresh(restarted)
	if _, ok = restarted.Snapshot(testNow); !ok {
		t.Fatal("fresh revision failed")
	}
	os.Remove(filepath.Join(dir, "binding.json"))
	refresh(restarted)
	if _, ok = restarted.Snapshot(testNow); ok {
		t.Fatal("removed binding did not disable")
	}
}
