package theme

import "github.com/gdamore/tcell/v2"

type Color = tcell.Color

var (
	BgPrimary    = tcell.NewRGBColor(0x0D, 0x11, 0x17)
	BgSecondary  = tcell.NewRGBColor(0x16, 0x1B, 0x22)
	BgTertiary   = tcell.NewRGBColor(0x21, 0x26, 0x2D)
	BgOverlay    = tcell.NewRGBColor(0x1C, 0x21, 0x28)

	BorderDefault = tcell.NewRGBColor(0x30, 0x36, 0x3D)
	BorderFocus   = tcell.NewRGBColor(0x58, 0xA6, 0xFF)
	BorderMuted   = tcell.NewRGBColor(0x21, 0x26, 0x2D)

	AccentCyan    = tcell.NewRGBColor(0x58, 0xA6, 0xFF)
	AccentGreen   = tcell.NewRGBColor(0x3F, 0xB9, 0x50)
	AccentAmber   = tcell.NewRGBColor(0xF0, 0x88, 0x3E)
	AccentMagenta = tcell.NewRGBColor(0xF7, 0x78, 0xBA)
	AccentRed     = tcell.NewRGBColor(0xF8, 0x51, 0x49)
	AccentPurple  = tcell.NewRGBColor(0xBC, 0x8C, 0xFF)

	TextPrimary   = tcell.NewRGBColor(0xE6, 0xED, 0xF3)
	TextSecondary = tcell.NewRGBColor(0x8B, 0x94, 0x9E)
	TextDim       = tcell.NewRGBColor(0x48, 0x4F, 0x58)
	TextMuted     = tcell.NewRGBColor(0x65, 0x6D, 0x76)
	TextLink      = tcell.NewRGBColor(0x58, 0xA6, 0xFF)

	White  = tcell.ColorWhite
	Black  = tcell.ColorBlack
	Transparent = tcell.ColorDefault
)

type Gradient struct {
	Colors []Color
}

func NewGradient(colors ...Color) Gradient {
	return Gradient{Colors: colors}
}

func (g Gradient) At(progress float64) Color {
	if len(g.GradientColors()) == 0 {
		return TextPrimary
	}
	if len(g.GradientColors()) == 1 {
		return g.GradientColors()[0]
	}
	if progress <= 0 {
		return g.GradientColors()[0]
	}
	if progress >= 1 {
		return g.GradientColors()[len(g.GradientColors())-1]
	}

	scale := float64(len(g.GradientColors()) - 1)
	idx := int(progress * scale)
	frac := (progress * scale) - float64(idx)

	if idx >= len(g.GradientColors())-1 {
		return g.GradientColors()[len(g.GradientColors())-1]
	}

	c1 := g.GradientColors()[idx]
	c2 := g.GradientColors()[idx+1]

	r1, g1, b1 := c1.RGB()
	r2, g2, b2 := c2.RGB()

	r := int(float64(r1) + (float64(r2)-float64(r1))*frac)
	gn := int(float64(g1) + (float64(g2)-float64(g1))*frac)
	b := int(float64(b1) + (float64(b2)-float64(b1))*frac)

	return tcell.NewRGBColor(int32(r), int32(gn), int32(b))
}

func (g Gradient) GradientColors() []Color {
	return g.Colors
}

var TitleGradient = NewGradient(
	AccentCyan,
	AccentPurple,
	AccentMagenta,
)

var AccentGradient = NewGradient(
	AccentCyan,
	AccentGreen,
)

var WarmGradient = NewGradient(
	AccentAmber,
	AccentMagenta,
)
