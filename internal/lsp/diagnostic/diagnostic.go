package diagnostic

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	Severity int    `json:"severity"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
	Range    Range  `json:"range"`
}

func (d Diagnostic) Line() int {
	return d.Range.Start.Line
}

func (d Diagnostic) Column() int {
	return d.Range.Start.Character
}

func (d Diagnostic) EndLine() int {
	return d.Range.End.Line
}

func (d Diagnostic) EndColumn() int {
	return d.Range.End.Character
}
