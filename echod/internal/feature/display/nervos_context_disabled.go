//go:build !dot && !spot && (!nervos_echo || panel)

package display

func (*renderer) nervosContext(scene) bool { return false }

func nervosClockFallback(s scene) scene { return s }
