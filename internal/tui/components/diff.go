package components

import "strings"

type DiffLine struct {
	Kind    byte
	OldLine int
	NewLine int
	Text    string
}

type DiffHunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int

	Lines []DiffLine
}

func ComputeDiff(oldText, newText string) []DiffHunk {
	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	ops := diffLines(oldLines, newLines)

	const context = 3

	var hunks []DiffHunk

	for i := 0; i < len(ops); {
		if ops[i].Kind == ' ' {
			i++
			continue
		}

		start := i - context
		if start < 0 {
			start = 0
		}

		end := i
		for end < len(ops) && ops[end].Kind != ' ' {
			end++
		}

		end += context
		if end > len(ops) {
			end = len(ops)
		}

		if len(hunks) > 0 {
			prev := &hunks[len(hunks)-1]
			prevEnd := prev.OldStart + prev.OldCount - 1
			_ = prevEnd
		}

		hunk := buildHunk(ops[start:end])
		hunks = append(hunks, hunk)
		i = end
	}

	return mergeHunks(hunks, context)
}

type diffOp struct {
	Kind byte
	Text string
}

func diffLines(oldLines, newLines []string) []diffOp {
	n := len(oldLines)
	m := len(newLines)

	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}

	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case oldLines[i] == newLines[j]:
			ops = append(ops, diffOp{Kind: ' ', Text: oldLines[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, diffOp{Kind: '-', Text: oldLines[i]})
			i++
		default:
			ops = append(ops, diffOp{Kind: '+', Text: newLines[j]})
			j++
		}
	}
	for i < n {
		ops = append(ops, diffOp{Kind: '-', Text: oldLines[i]})
		i++
	}
	for j < m {
		ops = append(ops, diffOp{Kind: '+', Text: newLines[j]})
		j++
	}
	return ops
}

func buildHunk(ops []diffOp) DiffHunk {
	var hunk DiffHunk

	oldLine := 1
	newLine := 1

	for _, op := range ops {
		if op.Kind != '+' {
			hunk.OldStart = oldLine
			break
		}
	}

	for _, op := range ops {
		if op.Kind != '-' {
			hunk.NewStart = newLine
			break
		}
	}

	if hunk.OldStart == 0 {
		hunk.OldStart = oldLine
	}
	if hunk.NewStart == 0 {
		hunk.NewStart = newLine
	}

	for _, op := range ops {
		line := DiffLine{Kind: op.Kind, Text: op.Text}
		switch op.Kind {
		case '-':
			line.OldLine = oldLine
			oldLine++
			hunk.OldCount++
		case '+':
			line.NewLine = newLine
			newLine++
			hunk.NewCount++
		default:
			line.OldLine = oldLine
			line.NewLine = newLine
			oldLine++
			newLine++
			hunk.OldCount++
			hunk.NewCount++
		}
		hunk.Lines = append(hunk.Lines, line)
	}

	return hunk
}

func mergeHunks(hunks []DiffHunk, context int) []DiffHunk {
	if len(hunks) < 2 {
		return hunks
	}
	_ = context
	return hunks
}

func HasDiff(hunks []DiffHunk) bool {
	for _, hunk := range hunks {
		for _, line := range hunk.Lines {
			if line.Kind != ' ' {
				return true
			}
		}
	}
	return false
}
