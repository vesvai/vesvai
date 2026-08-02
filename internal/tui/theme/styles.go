package theme

import "github.com/gdamore/tcell/v2"

type Style struct {
	Foreground    Color
	Background    Color
	Bold          bool
	Italic        bool
	Underline     bool
	Strikethrough bool
	Blink         bool
}

func NewStyle() Style {
	return Style{
		Foreground: TextPrimary,
		Background: Transparent,
	}
}

func (s Style) WithForeground(c Color) Style {
	s.Foreground = c
	return s
}

func (s Style) WithBackground(c Color) Style {
	s.Background = c
	return s
}

func (s Style) BoldOn() Style {
	s.Bold = true
	return s
}

func (s Style) ItalicOn() Style {
	s.Italic = true
	return s
}

func (s Style) UnderlineOn() Style {
	s.Underline = true
	return s
}

func (s Style) StrikethroughOn() Style {
	s.Strikethrough = true
	return s
}

func (s Style) ToTcell() tcell.Style {
	st := tcell.StyleDefault.
		Foreground(s.Foreground).
		Background(s.Background)
	if s.Bold {
		st = st.Bold(true)
	}
	if s.Italic {
		st = st.Italic(true)
	}
	if s.Underline {
		st = st.Underline(true)
	}
	if s.Strikethrough {
		st = st.StrikeThrough(true)
	}
	return st
}

var (
	TitleBigStyle = NewStyle().
			WithForeground(TitleColor).
			BoldOn()

	TitleSmallStyle = NewStyle().
				WithForeground(TitleColor).
				BoldOn()

	TitleBlockStyle = NewStyle().
			WithForeground(TitleColor).
			BoldOn()

	HeaderBrandStyle = NewStyle().
				WithForeground(TextDim).
				BoldOn()

	HeaderShortcutStyle = NewStyle().
				WithForeground(TextDim)

	HeaderSeparatorStyle = NewStyle().
				WithForeground(BorderDefault)

	InputBorderStyle = NewStyle().
				WithForeground(BorderDefault).
				WithBackground(BgPrimary)

	InputBorderFocusStyle = NewStyle().
				WithForeground(BorderDefault).
				WithBackground(BgPrimary)

	InputTextStyle = NewStyle().
			WithForeground(TextPrimary).
			WithBackground(BgSecondary)

	PlaceholderStyle = NewStyle().
				WithForeground(TextDim).
				WithBackground(BgSecondary)

	CursorStyle = NewStyle().
			WithForeground(AccentGold).
			BoldOn()

	MessageUserBorder = NewStyle().
				WithForeground(AccentCyan).
				WithBackground(tcell.NewRGBColor(0x12, 0x1A, 0x24))

	MessageAssistantBorder = NewStyle().
				WithForeground(AccentGreen).
				WithBackground(tcell.NewRGBColor(0x0F, 0x19, 0x12))

	MessageUserContent = NewStyle().
				WithForeground(TextPrimary).
				WithBackground(tcell.NewRGBColor(0x12, 0x1A, 0x24))

	MessageAssistantContent = NewStyle().
				WithForeground(TextPrimary).
				WithBackground(tcell.NewRGBColor(0x0F, 0x19, 0x12))

	MessageRoleLabel = NewStyle().
				WithForeground(TextSecondary).
				BoldOn()

	StatusBarStyle = NewStyle().
			WithForeground(TextDim).
			WithBackground(BgPrimary)

	ShortcutKeyStyle = NewStyle().
				WithForeground(AccentGold).
				WithBackground(BgPrimary).
				BoldOn()

	ShortcutDescStyle = NewStyle().
				WithForeground(TextDim).
				WithBackground(BgPrimary)

	TipStyle = NewStyle().
			WithForeground(AccentAmber)

	TipLabelStyle = NewStyle().
			WithForeground(AccentAmber).
			BoldOn()

	VersionStyle = NewStyle().
			WithForeground(TextDim).
			WithBackground(BgPrimary)

	MarkdownHeading1 = NewStyle().
				WithForeground(AccentCyan).
				BoldOn()

	MarkdownHeading2 = NewStyle().
				WithForeground(AccentPurple).
				BoldOn()

	MarkdownHeading3 = NewStyle().
				WithForeground(AccentMagenta).
				BoldOn()

	MarkdownHeading4 = NewStyle().
				WithForeground(AccentGreen).
				BoldOn()

	MarkdownHeading5 = NewStyle().
				WithForeground(AccentAmber).
				BoldOn()

	MarkdownHeading6 = NewStyle().
				WithForeground(TextSecondary).
				BoldOn()

	MarkdownBold = NewStyle().
			WithForeground(TextPrimary).
			BoldOn()

	MarkdownItalic = NewStyle().
			WithForeground(TextPrimary).
			ItalicOn()

	MarkdownStrikethrough = NewStyle().
				WithForeground(TextSecondary).
				StrikethroughOn()

	MarkdownCode = NewStyle().
			WithForeground(AccentAmber).
			WithBackground(BgTertiary)

	MarkdownCodeBlock = NewStyle().
				WithForeground(TextPrimary).
				WithBackground(BgSecondary)

	MarkdownBlockquote = NewStyle().
				WithForeground(TextSecondary).
				ItalicOn()

	MarkdownLink = NewStyle().
			WithForeground(TextLink).
			UnderlineOn()

	MarkdownListBullet = NewStyle().
				WithForeground(AccentCyan)

	MarkdownHR = NewStyle().
			WithForeground(BorderDefault).
			WithBackground(BgPrimary)

	MarkdownTableBorder = NewStyle().
				WithForeground(BorderDefault)

	MarkdownTableHeader = NewStyle().
				WithForeground(TextPrimary).
				BoldOn()

	MarkdownTableContent = NewStyle().
				WithForeground(TextPrimary)

	MarkdownParagraph = NewStyle().
				WithForeground(TextPrimary)

	SuccessStyle = NewStyle().
			WithForeground(AccentGreen)

	ErrorStyle = NewStyle().
			WithForeground(AccentRed)

	WarningStyle = NewStyle().
			WithForeground(AccentAmber)

	InfoStyle = NewStyle().
			WithForeground(AccentCyan)

	CommandPaletteBorder = NewStyle().
				WithForeground(BorderFocus).
				WithBackground(BgOverlay)

	CommandPaletteTitle = NewStyle().
				WithForeground(TextPrimary).
				WithBackground(BgOverlay).
				BoldOn()

	CommandPaletteSearch = NewStyle().
				WithForeground(TextPrimary).
				WithBackground(BgTertiary)

	CommandPaletteSearchPlaceholder = NewStyle().
					WithForeground(TextDim).
					WithBackground(BgTertiary)

	CommandPaletteCategory = NewStyle().
				WithForeground(AccentCyan).
				WithBackground(BgOverlay).
				BoldOn()

	CommandPaletteItem = NewStyle().
				WithForeground(TextPrimary).
				WithBackground(BgOverlay)

	CommandPaletteItemSelected = NewStyle().
					WithForeground(BgPrimary).
					WithBackground(AccentCyan).
					BoldOn()

	CommandPaletteShortcut = NewStyle().
				WithForeground(TextDim).
				WithBackground(BgOverlay)

	CommandPaletteShortcutSelected = NewStyle().
					WithForeground(BgPrimary).
					WithBackground(AccentCyan)

	CommandPaletteIcon = NewStyle().
				WithForeground(AccentAmber).
				WithBackground(BgOverlay)

	CommandPaletteIconSelected = NewStyle().
					WithForeground(BgPrimary).
					WithBackground(AccentCyan)

	ActionListBorder = NewStyle().
			WithForeground(BorderFocus).
			WithBackground(BgSecondary)

	ActionListItem = NewStyle().
			WithForeground(TextPrimary).
			WithBackground(BgSecondary)

	ActionListItemSelected = NewStyle().
				WithForeground(BgPrimary).
				WithBackground(AccentCyan).
				BoldOn()

	ActionListIcon = NewStyle().
			WithForeground(AccentAmber).
			WithBackground(BgSecondary)

	ActionListIconSelected = NewStyle().
				WithForeground(BgPrimary).
				WithBackground(AccentCyan)

	ActionListEmpty = NewStyle().
			WithForeground(TextDim).
			WithBackground(BgSecondary)

	ActionBlockStyle = NewStyle().
			WithForeground(AccentAmber).
			WithBackground(BgSecondary).
			BoldOn()

	MentionBlockStyle = NewStyle().
			WithForeground(AccentCyan).
			WithBackground(BgSecondary).
			BoldOn()

	ToolCardStyle = NewStyle().
			WithForeground(TextPrimary).
			WithBackground(BgSecondary)

	ToolCardBorderStyle = NewStyle().
				WithForeground(BorderDefault).
				WithBackground(BgSecondary)

	ToolIconStyle = NewStyle().
			WithForeground(AccentGold).
			WithBackground(BgSecondary).
			BoldOn()

	ToolNameStyle = NewStyle().
			WithForeground(TextPrimary).
			WithBackground(BgSecondary).
			BoldOn()

	ToolParamStyle = NewStyle().
			WithForeground(TextSecondary).
			WithBackground(BgSecondary)

	ToolMetaStyle = NewStyle().
			WithForeground(TextDim).
			WithBackground(BgSecondary)

	ToolExpandStyle = NewStyle().
			WithForeground(TextDim).
			WithBackground(BgSecondary)

	ToolResultPreviewStyle = NewStyle().
				WithForeground(TextSecondary).
				WithBackground(BgSecondary)

	UserMessageStyle = NewStyle().
				WithForeground(TextPrimary).
				WithBackground(BgSecondary)

	UserMessageBorderStyle = NewStyle().
				WithForeground(BorderDefault).
				WithBackground(BgSecondary)

	AssistantLabelStyle = NewStyle().
				WithForeground(TextDim).
				WithBackground(BgPrimary).
				BoldOn()

	AssistantContentStyle = NewStyle().
				WithForeground(TextPrimary).
				WithBackground(BgPrimary)
)
