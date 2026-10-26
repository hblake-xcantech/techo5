//go:build nervos_echo && !dot && !spot && !panel

package display

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HuskerMinion/techo5/echod/internal/config"
	"github.com/HuskerMinion/techo5/echod/internal/feature/nervoscontext"
	"github.com/HuskerMinion/techo5/echod/internal/feature/timer"
)

func TestNervosForegroundAndManualChoice(t *testing.T) {
	config.Use(filepath.Join(t.TempDir(), "config.json"))
	_ = config.Set().Screen().ClockStyle(styleNervosContext)
	t.Cleanup(func() { _ = config.Set().Screen().ClockStyle(""); _ = config.Set().Screen().Skin("") })
	base := scene{phase: "idle"}
	if !contextEligible(base) {
		t.Fatal("opt-in not available")
	}
	for name, change := range map[string]func(*scene){
		"voice": func(s *scene) { s.phase = "listening" }, "cue": func(s *scene) { s.cue = "thinking" },
		"drawer": func(s *scene) { s.showDrawer = true }, "volume": func(s *scene) { s.showVolume = true },
		"photos":       func(s *scene) { s.slideshow = image.NewRGBA(image.Rect(0, 0, 1, 1)) },
		"timer":        func(s *scene) { s.timers = []timer.Countdown{{Name: "Synthetic timer"}} },
		"notification": func(s *scene) { s.showNotification = true }, "night": func(s *scene) { s.nightDim = true },
	} {
		t.Run(name, func(t *testing.T) {
			s := base
			change(&s)
			if contextEligible(s) {
				t.Fatal("context took foreground")
			}
		})
	}
	_ = config.Set().Screen().ClockStyle(styleAnalog)
	if contextEligible(base) {
		t.Fatal("manual clock ignored")
	}
	_ = config.Set().Screen().ClockStyle(styleNervosContext)
	_ = config.Set().Screen().Skin(skinEInk)
	if contextEligible(base) {
		t.Fatal("manual skin ignored")
	}
	if nervosClockFallback(base).style.style() != styleClassic {
		t.Fatal("no empty/expired fallback")
	}
}
func TestNervosNativePreview(t *testing.T) {
	config.Use(filepath.Join(t.TempDir(), "config.json"))
	previewFont(t)
	_ = config.Set().Screen().Theme("NervOS")
	applyTheme(current())
	t.Cleanup(func() { _ = config.Set().Screen().Theme(""); applyTheme(current()) })
	at := time.Date(2026, 10, 26, 14, 7, 0, 0, time.UTC)
	cases := map[string]nervoscontext.Projection{
		"normal": {Summary: "The workbench is ready for the next step.", Why: "Shared context from the room operator."},
		"long":   {Summary: "The next assembly step is ready. Keep the parts on the workbench while the current check finishes; the room operator will share a fresh update when it is complete.", Why: "This synthetic room update expires automatically; no action has been performed."},
	}
	for name, p := range cases {
		img := image.NewRGBA(image.Rect(0, 0, 960, 480))
		r := newRenderer(img)
		if !r.contextPage(scene{now: at, phase: "idle"}, p) {
			t.Fatalf("%s text exceeds native layout", name)
		}
		if dir := os.Getenv("SHOW_PREVIEW"); dir != "" {
			file, err := os.Create(filepath.Join(dir, "nervos-"+name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			if err = png.Encode(file, img); err != nil {
				t.Fatal(err)
			}
			file.Close()
		}
	}
	if dir := os.Getenv("SHOW_PREVIEW"); dir != "" {
		_ = config.Set().Screen().ClockStyle(styleNervosContext)
		t.Cleanup(func() { _ = config.Set().Screen().ClockStyle("") })
		for name, s := range map[string]scene{"empty": {now: at, phase: "idle"}, "voice": {now: at, phase: "thinking", heard: "What is the next step?"}} {
			img := image.NewRGBA(image.Rect(0, 0, 960, 480))
			newRenderer(img).draw(s)
			file, err := os.Create(filepath.Join(dir, "nervos-"+name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			if err = png.Encode(file, img); err != nil {
				t.Fatal(err)
			}
			file.Close()
		}
	}
	r := newRenderer(image.NewRGBA(image.Rect(0, 0, 960, 480)))
	if r.contextPage(scene{now: at}, nervoscontext.Projection{Summary: strings.Repeat("wide ", 200), Why: "Synthetic overflow"}) {
		t.Fatal("silently truncated essential context")
	}
}

func TestNervosPaletteFollowsChoice(t *testing.T) {
	config.Use(filepath.Join(t.TempDir(), "config.json"))
	t.Cleanup(func() { _ = config.Set().Screen().Theme("") })
	var first uint32
	for i, name := range []string{"NervOS", "Ember"} {
		_ = config.Set().Screen().Theme(name)
		img := image.NewRGBA(image.Rect(0, 0, 960, 480))
		r := newRenderer(img)
		if !r.contextPage(scene{now: time.Unix(1800000000, 0)}, nervoscontext.Projection{Summary: "Synthetic shared context.", Why: "Synthetic explanation."}) {
			t.Fatal("did not render")
		}
		got := img.RGBAAt(0, 0)
		if got != current().colors[roleGround] {
			t.Fatal("manual palette ignored")
		}
		red, _, _, _ := got.RGBA()
		if i == 0 {
			first = red
		} else if first == red {
			t.Fatal("palette did not change")
		}
		if config.Get().Screen.Theme != name {
			t.Fatal("renderer changed the setting")
		}
	}
}
