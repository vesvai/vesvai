package tui

import "github.com/gdamore/tcell/v2"

type Palette struct {
	Background  tcell.Color
	Surface     tcell.Color
	Border      tcell.Color
	BorderFocus tcell.Color

	Foreground tcell.Color
	TextDim    tcell.Color
	Muted      tcell.Color

	Accent    tcell.Color
	AccentDim tcell.Color

	UserLabel      tcell.Color
	UserBg         tcell.Color
	AssistantLabel tcell.Color

	Success tcell.Color
	Error   tcell.Color
	Warning tcell.Color
	Running tcell.Color

	Selection tcell.Color

	Mention    tcell.Color
	SkillBlock tcell.Color

	ThinkingDim  tcell.Color
	ThinkingGlow tcell.Color
	Reasoning    tcell.Color

	Subagent tcell.Color

	CodeBg     tcell.Color
	CodeBorder tcell.Color
	CodeText   tcell.Color

	StatusBg tcell.Color
	HelpBg   tcell.Color

	TokenKeyword     tcell.Color
	TokenName        tcell.Color
	TokenFunction    tcell.Color
	TokenString      tcell.Color
	TokenNumber      tcell.Color
	TokenComment     tcell.Color
	TokenOperator    tcell.Color
	TokenPunctuation tcell.Color
	TokenType        tcell.Color
	TokenConstant    tcell.Color
	TokenGeneric     tcell.Color
}

func DefaultDark() *Palette {
	return &Palette{
		Background:  tcell.NewRGBColor(0x0d, 0x11, 0x17),
		Surface:     tcell.NewRGBColor(0x16, 0x1b, 0x22),
		Border:      tcell.NewRGBColor(0x30, 0x36, 0x3d),
		BorderFocus: tcell.NewRGBColor(0x58, 0xa6, 0xff),

		Foreground: tcell.NewRGBColor(0xe6, 0xed, 0xf3),
		TextDim:    tcell.NewRGBColor(0x8b, 0x94, 0x9e),
		Muted:      tcell.NewRGBColor(0x6e, 0x76, 0x81),

		Accent:    tcell.NewRGBColor(0x58, 0xa6, 0xff),
		AccentDim: tcell.NewRGBColor(0x1f, 0x6f, 0xeb),

		UserLabel:      tcell.NewRGBColor(0x7e, 0xe7, 0x87),
		UserBg:         tcell.NewRGBColor(0x1c, 0x21, 0x28),
		AssistantLabel: tcell.NewRGBColor(0x79, 0xc0, 0xff),

		Success: tcell.NewRGBColor(0x3f, 0xb9, 0x50),
		Error:   tcell.NewRGBColor(0xf8, 0x51, 0x49),
		Warning: tcell.NewRGBColor(0xd2, 0x99, 0x22),
		Running: tcell.NewRGBColor(0x58, 0xa6, 0xff),

		Selection: tcell.NewRGBColor(0x26, 0x4f, 0x78),

		Mention: tcell.NewRGBColor(0xe3, 0xb3, 0x41),

		SkillBlock: tcell.NewRGBColor(0x7d, 0xd3, 0xfc),

		ThinkingDim:  tcell.NewRGBColor(0x15, 0x5e, 0x75),
		ThinkingGlow: tcell.NewRGBColor(0x67, 0xe8, 0xf9),
		Reasoning:    tcell.NewRGBColor(0x7c, 0x8a, 0x9e),

		CodeBg:     tcell.NewRGBColor(0x16, 0x1b, 0x22),
		CodeBorder: tcell.NewRGBColor(0x30, 0x36, 0x3d),
		CodeText:   tcell.NewRGBColor(0xe6, 0xed, 0xf3),

		StatusBg: tcell.NewRGBColor(0x0d, 0x11, 0x17),
		HelpBg:   tcell.NewRGBColor(0x0f, 0x14, 0x1a),

		TokenKeyword:     tcell.NewRGBColor(0xff, 0x7b, 0x72),
		TokenName:        tcell.NewRGBColor(0xe6, 0xed, 0xf3),
		TokenFunction:    tcell.NewRGBColor(0xd2, 0xa8, 0xff),
		TokenString:      tcell.NewRGBColor(0xa5, 0xd6, 0xff),
		TokenNumber:      tcell.NewRGBColor(0x79, 0xc0, 0xff),
		TokenComment:     tcell.NewRGBColor(0x8b, 0x94, 0x9e),
		TokenOperator:    tcell.NewRGBColor(0xff, 0x7b, 0x72),
		TokenPunctuation: tcell.NewRGBColor(0x8b, 0x94, 0x9e),
		TokenType:        tcell.NewRGBColor(0xff, 0xa6, 0x57),
		TokenConstant:    tcell.NewRGBColor(0x79, 0xc0, 0xff),
		TokenGeneric:     tcell.NewRGBColor(0x7e, 0xe7, 0x87),
	}
}

func FallbackDark() *Palette {
	return &Palette{
		Background:  tcell.ColorBlack,
		Surface:     tcell.ColorBlack,
		Border:      tcell.ColorGray,
		BorderFocus: tcell.ColorBlue,

		Foreground: tcell.ColorWhite,
		TextDim:    tcell.ColorGray,
		Muted:      tcell.ColorGray,

		Accent:    tcell.ColorBlue,
		AccentDim: tcell.ColorDarkBlue,

		UserLabel:      tcell.ColorGreen,
		UserBg:         tcell.ColorBlack,
		AssistantLabel: tcell.ColorLightBlue,

		Success: tcell.ColorGreen,
		Error:   tcell.ColorRed,
		Warning: tcell.ColorYellow,
		Running: tcell.ColorLightBlue,

		Selection: tcell.ColorBlue,

		Mention: tcell.ColorYellow,

		SkillBlock: tcell.ColorLightBlue,

		ThinkingDim:  tcell.ColorTeal,
		ThinkingGlow: tcell.ColorAqua,
		Reasoning:    tcell.ColorGray,

		Subagent: tcell.ColorFuchsia,

		CodeBg:     tcell.ColorBlack,
		CodeBorder: tcell.ColorGray,
		CodeText:   tcell.ColorWhite,

		StatusBg: tcell.ColorBlack,
		HelpBg:   tcell.ColorBlack,

		TokenKeyword:     tcell.ColorRed,
		TokenName:        tcell.ColorWhite,
		TokenFunction:    tcell.ColorFuchsia,
		TokenString:      tcell.ColorGreen,
		TokenNumber:      tcell.ColorAqua,
		TokenComment:     tcell.ColorGray,
		TokenOperator:    tcell.ColorRed,
		TokenPunctuation: tcell.ColorGray,
		TokenType:        tcell.ColorYellow,
		TokenConstant:    tcell.ColorAqua,
		TokenGeneric:     tcell.ColorGreen,
	}
}

func DetectPalette(s tcell.Screen) *Palette {
	if s.Colors() >= 1<<24 {
		return DefaultDark()
	}
	return FallbackDark()
}

func (p *Palette) TrueColor() bool {
	return uint64(p.Accent)&uint64(tcell.ColorIsRGB) != 0
}

func (p *Palette) TextStyle() tcell.Style {
	return tcell.StyleDefault.Background(p.Background).Foreground(p.Foreground)
}

func (p *Palette) DimTextStyle() tcell.Style {
	return tcell.StyleDefault.Background(p.Background).Foreground(p.TextDim)
}

func (p *Palette) Style(fg, bg tcell.Color) tcell.Style {
	return tcell.StyleDefault.Background(bg).Foreground(fg)
}
