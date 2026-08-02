package theme

import "github.com/gdamore/tcell/v2"

type Color = tcell.Color

var (
	BgPrimary    = tcell.NewRGBColor(0x0a, 0x0a, 0x0a)
	BgSecondary  = tcell.NewRGBColor(0x14, 0x14, 0x14)
	BgTertiary   = tcell.NewRGBColor(0x1e, 0x1e, 0x1e)
	BgOverlay    = tcell.NewRGBColor(0x12, 0x12, 0x12)

	BorderDefault = tcell.NewRGBColor(0x33, 0x33, 0x33)
	BorderFocus   = tcell.NewRGBColor(0x55, 0x55, 0x55)
	BorderMuted   = tcell.NewRGBColor(0x28, 0x28, 0x28)

	AccentCyan    = tcell.NewRGBColor(0x58, 0xA6, 0xFF)
	AccentGreen   = tcell.NewRGBColor(0x3F, 0xB9, 0x50)
	AccentAmber   = tcell.NewRGBColor(0xD4, 0xA5, 0x37)
	AccentMagenta = tcell.NewRGBColor(0xF7, 0x78, 0xBA)
	AccentRed     = tcell.NewRGBColor(0xF8, 0x51, 0x49)
	AccentPurple  = tcell.NewRGBColor(0xBC, 0x8C, 0xFF)
	AccentGold    = tcell.NewRGBColor(0xD4, 0xA5, 0x37)

	TextPrimary   = tcell.NewRGBColor(0xe0, 0xd5, 0xc0)
	TextSecondary = tcell.NewRGBColor(0x77, 0x77, 0x77)
	TextDim       = tcell.NewRGBColor(0x55, 0x55, 0x55)
	TextMuted     = tcell.NewRGBColor(0x66, 0x66, 0x66)
	TextLink      = tcell.NewRGBColor(0x58, 0xA6, 0xFF)

	TitleColor    = tcell.NewRGBColor(0xe0, 0xd5, 0xc0)
	SubtitleColor = tcell.NewRGBColor(0x77, 0x77, 0x77)

	White       = tcell.ColorWhite
	Black       = tcell.ColorBlack
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
	TitleColor,
)

var AccentGradient = NewGradient(
	AccentCyan,
	AccentGreen,
)

var WarmGradient = NewGradient(
	AccentAmber,
	AccentMagenta,
)
