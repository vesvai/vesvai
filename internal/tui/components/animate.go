package components

import (
	"math"
	"time"

	"github.com/gdamore/tcell/v2"
)

func glowT(elapsed time.Duration) float64 {
	return 0.5 + 0.5*math.Sin(float64(elapsed)/float64(time.Second)*2*math.Pi)
}

func lerpColor(a, b tcell.Color, t float64) tcell.Color {
	if uint64(a)&uint64(tcell.ColorIsRGB) == 0 || uint64(b)&uint64(tcell.ColorIsRGB) == 0 {
		if t < 0.5 {
			return a
		}
		return b
	}
	ar, ag, ab := a.RGB()
	br, bg, bb := b.RGB()
	return tcell.NewRGBColor(
		ar+int32(float64(br-ar)*t),
		ag+int32(float64(bg-ag)*t),
		ab+int32(float64(bb-ab)*t),
	)
}

var spinnerFrames = []rune{'◐', '◓', '◑', '◒'}

func spinnerAt(elapsed time.Duration) rune {
	return spinnerFrames[int(elapsed/120*time.Millisecond)%len(spinnerFrames)]
}
