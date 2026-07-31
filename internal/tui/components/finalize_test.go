package components

import "testing"

func TestStreamingMessage_FinalizePreservesTools(t *testing.T) {
	sm := NewStreamingMessage()
	sm.AppendContent("final answer")
	sm.AddToolCall("bash", map[string]any{"command": "ls"})
	sm.CompleteToolCall("bash", "file1\n", true, 12)

	msg := sm.Finalize()
	if msg.Content != "final answer" {
		t.Errorf("content = %q, want final answer", msg.Content)
	}
	if len(msg.Tools) != 1 {
		t.Fatalf("tools = %d, want 1", len(msg.Tools))
	}
	if msg.Tools[0].Name != "bash" || msg.Tools[0].Status != ToolComplete {
		t.Errorf("tool = %+v, want bash/complete", msg.Tools[0])
	}
}

func TestStreamingMessage_FinalizeNoTools(t *testing.T) {
	sm := NewStreamingMessage()
	sm.AppendContent("just text")
	msg := sm.Finalize()
	if len(msg.Tools) != 0 {
		t.Errorf("tools = %d, want 0", len(msg.Tools))
	}
}

func TestMessageBubble_HeightIncludesTools(t *testing.T) {
	sm := NewStreamingMessage()
	sm.AppendContent("answer")
	sm.AddToolCall("read", map[string]any{"path": "x"})
	msg := sm.Finalize()

	bubble := NewMessageBubble(msg)
	plain := NewMessageBubble(NewAssistantMessage("answer"))
	h1 := bubble.Height(80)
	h2 := plain.Height(80)
	if h1 <= h2 {
		t.Errorf("height with tools = %d, plain = %d; want larger", h1, h2)
	}
}

func TestToolDisplay_PreviewLinesTruncated(t *testing.T) {
	td := NewToolDisplay("bash", nil)
	td.SetRunning()

	long := ""
	for i := 0; i < 40; i++ {
		long += "line of output text for testing truncation\n"
	}
	td.SetComplete(long, 123)

	lines := td.previewLines(100)
	if len(lines) > maxPreviewLines+1 {
		t.Errorf("preview lines = %d, want at most %d", len(lines), maxPreviewLines+1)
	}
	if lines[len(lines)-1] != "…" {
		t.Errorf("last preview line should be the ellipsis, got %q", lines[len(lines)-1])
	}

	td2 := NewToolDisplay("read", nil)
	td2.SetRunning()
	td2.SetComplete("single line", 5)
	if got := td2.previewLines(100); len(got) != 1 || got[0] != "single line" {
		t.Errorf("compact preview = %v", got)
	}
}

func TestToolDisplay_StatusGlyphNoEmoji(t *testing.T) {
	td := NewToolDisplay("bash", nil)
	for _, status := range []ToolStatus{ToolPending, ToolRunning, ToolComplete, ToolFailed} {
		td.Status = status
		g := td.statusGlyph()
		runes := []rune(g)
		if len(runes) != 1 {
			t.Errorf("status %d glyph %q should be a single rune", status, g)
		}
	}
}
