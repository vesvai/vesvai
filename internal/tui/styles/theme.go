package styles

import "github.com/gdamore/tcell/v2"

type Theme struct {
	Name             string
	Background       tcell.Color
	Surface          tcell.Color
	Foreground       tcell.Color
	Accent           tcell.Color
	AccentDim        tcell.Color
	LogoBlink        tcell.Color
	Border           tcell.Color
	InputBg          tcell.Color
	InputText        tcell.Color
	Placeholder      tcell.Color
	Cursor           tcell.Color
	Selection        tcell.Color
	Chip             tcell.Color
	Mention          tcell.Color
	StatusBar        tcell.Color
	Hint             tcell.Color
	Warning          tcell.Color
	TextDim          tcell.Color
	Muted            tcell.Color
	Success          tcell.Color
	Error            tcell.Color
	Running          tcell.Color
	ThinkingDim      tcell.Color
	ThinkingGlow     tcell.Color
	Reasoning        tcell.Color
	Subagent         tcell.Color
	CodeBg           tcell.Color
	CodeBorder       tcell.Color
	CodeText         tcell.Color
	StatusBg         tcell.Color
	HelpBg           tcell.Color
	UserLabel        tcell.Color
	UserBg           tcell.Color
	AssistantLabel   tcell.Color
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

func (t Theme) Base() tcell.Style {
	return tcell.StyleDefault.Background(t.Background).Foreground(t.Foreground)
}
