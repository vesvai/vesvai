package styles

// Tokyo Night Storm - Balanced dark theme
// Official palette: https://github.com/folke/tokyonight.nvim
func TokyoNightStorm() Theme {
	return Theme{
		Background:       c(0x24283B), // bg
		Surface:          c(0x1F2335), // bg_dark
		Foreground:       c(0xC0CAF5), // fg
		Accent:           c(0x7AA2F7), // blue
		AccentDim:        c(0x3D59A1), // blue0
		LogoBlink:        c(0xBB9AF7), // magenta
		Border:           c(0x292E42), // bg_highlight
		InputBg:          c(0x1F2335), // bg_dark
		InputText:        c(0xC0CAF5), // fg
		Placeholder:      c(0x565F89), // comment
		Cursor:           c(0xC0CAF5), // fg
		Selection:        c(0x2E3C64), // bg_visual
		Chip:             c(0xFF9E64), // orange
		Mention:          c(0x1DC9A0), // teal
		StatusBar:        c(0x1F2335), // bg_statusline
		Hint:             c(0x737AA2), // dark5
		Warning:          c(0xE0AF68), // yellow
		TextDim:          c(0x545C7E), // dark3
		Muted:            c(0x565F89), // comment
		Success:          c(0x9ECE6A), // green
		Error:            c(0xF7768E), // red
		Running:          c(0x2AC3DE), // blue1
		ThinkingDim:      c(0x292E42), // bg_highlight
		ThinkingGlow:     c(0x7DCFFF), // cyan
		Reasoning:        c(0x565F89), // comment
		Subagent:         c(0xBB9AF7), // magenta
		CodeBg:           c(0x1F2335), // bg_dark
		CodeBorder:       c(0x292E42), // bg_highlight
		CodeText:         c(0xC0CAF5), // fg
		StatusBg:         c(0x1F2335), // bg_statusline
		HelpBg:           c(0x1A1B26), // bg_dark1
		UserLabel:        c(0x9ECE6A), // green
		UserBg:           c(0x1F2335), // bg_dark
		AssistantLabel:   c(0x7AA2F7), // blue
		TokenKeyword:     c(0xBB9AF7), // magenta
		TokenName:        c(0xC0CAF5), // fg
		TokenFunction:    c(0x7AA2F7), // blue
		TokenString:      c(0x9ECE6A), // green
		TokenNumber:      c(0xFF9E64), // orange
		TokenComment:     c(0x565F89), // comment
		TokenOperator:    c(0x89DDFF), // blue5
		TokenPunctuation: c(0xA9B1D6), // fg_dark
		TokenType:        c(0x2AC3DE), // blue1
		TokenConstant:    c(0xFF9E64), // orange
		TokenGeneric:     c(0x73DACA), // green1
	}
}

// Tokyo Night Night - Darker variant
func TokyoNightNight() Theme {
	return Theme{
		Background:       c(0x1A1B26), // bg
		Surface:          c(0x16161E), // bg_dark
		Foreground:       c(0xC0CAF5), // fg
		Accent:           c(0x7AA2F7), // blue
		AccentDim:        c(0x3D59A1), // blue0
		LogoBlink:        c(0xBB9AF7), // magenta
		Border:           c(0x292E42), // bg_highlight
		InputBg:          c(0x16161E), // bg_dark
		InputText:        c(0xC0CAF5), // fg
		Placeholder:      c(0x565F89), // comment
		Cursor:           c(0xC0CAF5), // fg
		Selection:        c(0x2E3C64), // bg_visual
		Chip:             c(0xFF9E64), // orange
		Mention:          c(0x1DC9A0), // teal
		StatusBar:        c(0x16161E), // bg_statusline
		Hint:             c(0x737AA2), // dark5
		Warning:          c(0xE0AF68), // yellow
		TextDim:          c(0x545C7E), // dark3
		Muted:            c(0x565F89), // comment
		Success:          c(0x9ECE6A), // green
		Error:            c(0xF7768E), // red
		Running:          c(0x2AC3DE), // blue1
		ThinkingDim:      c(0x292E42), // bg_highlight
		ThinkingGlow:     c(0x7DCFFF), // cyan
		Reasoning:        c(0x565F89), // comment
		Subagent:         c(0xBB9AF7), // magenta
		CodeBg:           c(0x16161E), // bg_dark
		CodeBorder:       c(0x292E42), // bg_highlight
		CodeText:         c(0xC0CAF5), // fg
		StatusBg:         c(0x16161E), // bg_statusline
		HelpBg:           c(0x16161E), // bg_dark
		UserLabel:        c(0x9ECE6A), // green
		UserBg:           c(0x16161E), // bg_dark
		AssistantLabel:   c(0x7AA2F7), // blue
		TokenKeyword:     c(0xBB9AF7), // magenta
		TokenName:        c(0xC0CAF5), // fg
		TokenFunction:    c(0x7AA2F7), // blue
		TokenString:      c(0x9ECE6A), // green
		TokenNumber:      c(0xFF9E64), // orange
		TokenComment:     c(0x565F89), // comment
		TokenOperator:    c(0x89DDFF), // blue5
		TokenPunctuation: c(0xA9B1D6), // fg_dark
		TokenType:        c(0x2AC3DE), // blue1
		TokenConstant:    c(0xFF9E64), // orange
		TokenGeneric:     c(0x73DACA), // green1
	}
}

// Tokyo Night Day - Light variant
func TokyoNightDay() Theme {
	return Theme{
		Background:       c(0xE1E2E7), // bg
		Surface:          c(0xD5D6DB), // bg_dark
		Foreground:       c(0x3760BF), // fg
		Accent:           c(0x2E7DE9), // blue
		AccentDim:        c(0x8990B3), // fg_dim
		LogoBlink:        c(0x843CCE), // magenta
		Border:           c(0xC4C8DA), // bg_highlight
		InputBg:          c(0xD5D6DB), // bg_dark
		InputText:        c(0x3760BF), // fg
		Placeholder:      c(0x848CB5), // comment
		Cursor:           c(0x3760BF), // fg
		Selection:        c(0xB6BFE2), // bg_visual
		Chip:             c(0x895705), // orange
		Mention:          c(0x0F4B6E), // teal
		StatusBar:        c(0xD5D6DB), // bg_statusline
		Hint:             c(0x8990B3), // fg_dim
		Warning:          c(0x895705), // yellow
		TextDim:          c(0x848CB5), // comment
		Muted:            c(0x848CB5), // comment
		Success:          c(0x388E3C), // green
		Error:            c(0xF52A35), // red
		Running:          c(0x0F4B6E), // blue1
		ThinkingDim:      c(0xC4C8DA), // bg_highlight
		ThinkingGlow:     c(0x0F4B6E), // cyan
		Reasoning:        c(0x848CB5), // comment
		Subagent:         c(0x843CCE), // magenta
		CodeBg:           c(0xD5D6DB), // bg_dark
		CodeBorder:       c(0xC4C8DA), // bg_highlight
		CodeText:         c(0x3760BF), // fg
		StatusBg:         c(0xD5D6DB), // bg_statusline
		HelpBg:           c(0x3760BF), // bg_dark
		UserLabel:        c(0x388E3C), // green
		UserBg:           c(0xD5D6DB), // bg_dark
		AssistantLabel:   c(0x2E7DE9), // blue
		TokenKeyword:     c(0x843CCE), // magenta
		TokenName:        c(0x3760BF), // fg
		TokenFunction:    c(0x2E7DE9), // blue
		TokenString:      c(0x388E3C), // green
		TokenNumber:      c(0x895705), // orange
		TokenComment:     c(0x848CB5), // comment
		TokenOperator:    c(0x0F4B6E), // blue5
		TokenPunctuation: c(0x6173AE), // fg_dark
		TokenType:        c(0x0F4B6E), // blue1
		TokenConstant:    c(0x895705), // orange
		TokenGeneric:     c(0x388E3C), // green1
	}
}
