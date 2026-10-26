//go:build nervos_echo && !dot && !spot && !panel

package display

import (
	"image"
	"image/draw"

	"github.com/HuskerMinion/techo5/echod/internal/feature/nervoscontext"
)

const styleNervosContext = "nervos-context"

// Register during variable initialization, before display.init constructs the existing HA select.
var nervosStyleRegistered = func() bool {
	clockStyles = append(clockStyles, struct{ label, value string }{"NervOS context", styleNervosContext})
	themes = append(themes, preset("NervOS", 0x293b46, 0x91a99a, 0xedf0ed, 0x91a99a, 0x91a99a))
	return true
}()

func nervosClockFallback(s scene) scene {
	if s.style.style() == styleNervosContext {
		s.style.chosen = true
		s.style.kind = styleClassic
	}
	return s
}

func contextEligible(s scene) bool {
	return clockStyle() == styleNervosContext && skin() == "" && s.phase == "idle" && s.cue == "" && s.slideshow == nil && !s.showDrawer && !s.showVolume && !s.strip && !s.callButton && !s.showReminder && !s.showNotification && !s.showAnnouncement && !s.announceRecording && s.popup == nil && s.nextUp == nil && len(s.timers) == 0 && len(s.glance) == 0 && s.alarms.Next == nil && !s.nightDim
}
func (r *renderer) nervosContext(s scene) bool {
	if !contextEligible(s) {
		return false
	}
	p, ok := nervoscontext.Get().Snapshot(s.now)
	if !ok {
		return false
	}
	return r.contextPage(s, p)
}

// Context has no tap zones. Existing idle tap-to-talk and settings gestures remain upstream-owned.
// NerveTheme preset, stable typography and no animation or Presence sound changes.
func (r *renderer) contextPage(s scene, p nervoscontext.Projection) bool {
	palette := current()
	cloud, slate, signal := palette.colors[roleText], palette.colors[roleGround], palette.colors[roleDim]
	summary := r.wrapLines(r.body, p.Summary, r.w-2*r.margin)
	why := r.wrapLines(r.tiny, p.Why, r.w-2*r.margin)
	// Font changes are human-owned. Yield to the clock if this font cannot show the whole text.
	if len(summary) > 4 || len(why) > 2 {
		return false
	}
	draw.Draw(r.dst, r.dst.Rect, image.NewUniform(slate), image.Point{}, draw.Src)
	r.text(r.tiny, "NERVOS  /  SHARED ROOM", r.margin, r.s(76), signal)
	clock := clockText(s.now)
	r.text(r.tiny, clock, r.w-r.margin-r.width(r.tiny, clock), r.s(76), cloud)
	draw.Draw(r.dst, image.Rect(r.margin, r.s(104), r.w-r.margin, r.s(105)), image.NewUniform(signal), image.Point{}, draw.Src)
	for i, line := range summary {
		r.text(r.body, line, r.margin, r.s(159+i*48), cloud)
	}
	for i, line := range why {
		r.text(r.tiny, line, r.margin, r.s(365+i*33), signal)
	}
	return true
}
