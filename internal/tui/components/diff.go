package components

import "strings"

type DiffLine struct {
	Kind byte
	Text string
}

func ComputeDiff(oldText, newText string) []DiffLine {
	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	start := 0
	for start < len(oldLines) && start < len(newLines) && oldLines[start] == newLines[start] {
		start++
	}
	oe, ne := len(oldLines), len(newLines)
	for oe > start && ne > start && oldLines[oe-1] == newLines[ne-1] {
		oe--
		ne--
	}

	var out []DiffLine
	for i := start; i < oe; i++ {
		out = append(out, DiffLine{Kind: '-', Text: oldLines[i]})
	}
	for i := start; i < ne; i++ {
		out = append(out, DiffLine{Kind: '+', Text: newLines[i]})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func HasDiff(d []DiffLine) bool {
	for _, l := range d {
		if l.Kind != ' ' {
			return true
		}
	}
	return false
}

func DiffText(d []DiffLine) []string {
	out := make([]string, 0, len(d))
	for _, l := range d {
		out = append(out, string(l.Kind)+" "+l.Text)
	}
	return out
}
