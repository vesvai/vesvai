package styles

import "github.com/gdamore/tcell/v2"

type Theme struct {
	Name        string
	Background  tcell.Color
	Foreground  tcell.Color
	Accent      tcell.Color
	AccentDim   tcell.Color
	LogoBlink   tcell.Color
	Border      tcell.Color
	InputBg     tcell.Color
	InputText   tcell.Color
	Placeholder tcell.Color
	Cursor      tcell.Color
	Selection   tcell.Color
	Chip        tcell.Color
	Mention     tcell.Color
	StatusBar   tcell.Color
	Hint        tcell.Color
	Warning     tcell.Color
}

func (t Theme) Base() tcell.Style {
	return tcell.StyleDefault.Background(t.Background).Foreground(t.Foreground)
}
