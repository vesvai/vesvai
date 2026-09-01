package styles

func Dracula() Theme {
	return Theme{
		Background:  c(0x282A36),
		Foreground:  c(0xF8F8F2),
		Accent:      c(0xBD93F9),
		AccentDim:   c(0x6272A4),
		LogoBlink:   c(0xFF79C6),
		Border:      c(0x44475A),
		InputBg:     c(0x21222C),
		InputText:   c(0xF8F8F2),
		Placeholder: c(0x6272A4),
		Cursor:      c(0xF8F8F2),
		Selection:   c(0x44475A),
		Chip:        c(0xFFB86C),
		Mention:     c(0x8BE9FD),
		StatusBar:   c(0x1E1F29),
		Hint:        c(0x6272A4),
		Warning:     c(0xFFB86C),
	}
}
