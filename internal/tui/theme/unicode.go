package theme

type BorderStyle struct {
	Horizontal  string
	Vertical    string
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	TopT        string
	BottomT     string
	LeftT       string
	RightT      string
	Cross       string
}

var RoundedBorder = BorderStyle{
	Horizontal:  "─",
	Vertical:    "│",
	TopLeft:     "╭",
	TopRight:    "╮",
	BottomLeft:  "╰",
	BottomRight: "╯",
	TopT:        "┬",
	BottomT:     "┴",
	LeftT:       "├",
	RightT:      "┤",
	Cross:       "┼",
}

var SharpBorder = BorderStyle{
	Horizontal:  "─",
	Vertical:    "│",
	TopLeft:     "┌",
	TopRight:    "┐",
	BottomLeft:  "└",
	BottomRight: "┘",
	TopT:        "┬",
	BottomT:     "┴",
	LeftT:       "├",
	RightT:      "┤",
	Cross:       "┼",
}

var DoubleBorder = BorderStyle{
	Horizontal:  "═",
	Vertical:    "║",
	TopLeft:     "╔",
	TopRight:    "╗",
	BottomLeft:  "╚",
	BottomRight: "╝",
	TopT:        "╦",
	BottomT:     "╩",
	LeftT:       "╠",
	RightT:      "╣",
	Cross:       "╬",
}

var ThickBorder = BorderStyle{
	Horizontal:  "━",
	Vertical:    "┃",
	TopLeft:     "┏",
	TopRight:    "┓",
	BottomLeft:  "┗",
	BottomRight: "┛",
	TopT:        "┳",
	BottomT:     "┻",
	LeftT:       "┣",
	RightT:      "┫",
	Cross:       "╋",
}

var (
	Dot          = "●"
	Diamond      = "◆"
	ArrowRight   = "→"
	ArrowLeft    = "←"
	ArrowUp      = "↑"
	ArrowDown    = "↓"
	Bullet       = "•"
	Check        = "✓"
	Cross2       = "✗"
	Star         = "★"
	Heart        = "♥"
	Circle       = "○"
	CircleFilled = "●"
	Percent      = "%%"
	Pipe         = "│"
	Slash        = "/"
	Dash         = "—"
	Dots         = "···"
	Sparkle      = "✦"
	Lightning    = "◇"
	Ellipsis     = "…"

	ToolIconGlob     = "Q"
	ToolIconGrep     = "Q"
	ToolIconRead     = "D"
	ToolIconWrite    = "W"
	ToolIconTodo     = "T"
	ToolIconSubAgent = "S"
	ToolIconMessage  = "M"
	ToolIconList     = "Q"
	ToolIconDefault  = "●"

	ToolStatusPending  = "○"
	ToolStatusRunning  = "◌"
	ToolStatusComplete = "●"
	ToolStatusFailed   = "×"
)

var TitleBlock = []string{
	`██████╗ ███████╗███████╗██╗   ██╗██╗████████╗███████╗`,
	`██╔══██╗██╔════╝██╔════╝██║   ██║██║╚══██╔══╝██╔════╝`,
	`██████╔╝█████╗  ███████╗██║   ██║██║   ██║   █████╗  `,
	`██╔══██╗██╔══╝  ╚════██║██║   ██║██║   ██║   ██╔══╝  `,
	`██║  ██║███████╗███████║╚██████╔╝██║   ██║   ███████╗`,
	`╚═╝  ╚═╝╚══════╝╚══════╝ ╚═════╝ ╚═╝   ╚═╝   ╚══════╝`,
}

var TitleBlockSmall = []string{
	`V E S V A I`,
}

func LerpColor(r1, g1, b1, r2, g2, b2 int, t float64) (int, int, int) {
	r := int(float64(r1) + (float64(r2)-float64(r1))*t)
	g := int(float64(g1) + (float64(g2)-float64(g1))*t)
	b := int(float64(b1) + (float64(b2)-float64(b1))*t)
	return r, g, b
}
