package main

import (
	"reflect"
	"testing"
)

// assertBlocksEqual compares two block slices field by field, tolerating the
// nil-vs-empty-map distinction on Input (a text block has no Input).
func assertBlocksEqual(t *testing.T, label string, got, want []Block) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: Blocks len = %d, want %d (got=%+v)", label, len(got), len(want), got)
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Type != w.Type {
			t.Errorf("%s: Blocks[%d].Type = %q, want %q", label, i, g.Type, w.Type)
		}
		if g.Text != w.Text {
			t.Errorf("%s: Blocks[%d].Text = %q, want %q", label, i, g.Text, w.Text)
		}
		if g.Name != w.Name {
			t.Errorf("%s: Blocks[%d].Name = %q, want %q", label, i, g.Name, w.Name)
		}
		if len(g.Input) != 0 || len(w.Input) != 0 {
			if !reflect.DeepEqual(g.Input, w.Input) {
				t.Errorf("%s: Blocks[%d].Input = %v, want %v", label, i, g.Input, w.Input)
			}
		}
	}
}

// TestPiContentBlocks covers the pi->Claude normalization: text passes
// through, toolCall becomes tool_use with the name and arg-key remap, unknown
// tools keep their raw name, and thinking blocks are dropped.
func TestPiContentBlocks(t *testing.T) {
	content := []any{
		map[string]any{"type": "thinking", "thinking": "drop me"},
		map[string]any{"type": "text", "text": "hi"},
		map[string]any{"type": "toolCall", "name": "edit", "arguments": map[string]any{"path": "/f", "old": "a"}},
		map[string]any{"type": "toolCall", "name": "customtool", "arguments": map[string]any{"x": float64(1)}},
	}
	got := piContentBlocks(content)
	want := []Block{
		{Type: "text", Text: "hi"},
		{Type: "tool_use", Name: "Edit", Input: map[string]any{"file_path": "/f", "old": "a"}},
		{Type: "tool_use", Name: "customtool", Input: map[string]any{"x": float64(1)}},
	}
	assertBlocksEqual(t, "pi", got, want)
}

// TestPiContentBlocksString: a bare string becomes one text block.
func TestPiContentBlocksString(t *testing.T) {
	got := piContentBlocks("just text")
	assertBlocksEqual(t, "pi-string", got, []Block{{Type: "text", Text: "just text"}})
}

// TestClaudeContentBlocks: text and tool_use pass through; thinking and
// tool_result are dropped.
func TestClaudeContentBlocks(t *testing.T) {
	content := []any{
		map[string]any{"type": "thinking", "thinking": "drop"},
		map[string]any{"type": "text", "text": "hello"},
		map[string]any{"type": "tool_use", "name": "Bash", "input": map[string]any{"command": "ls"}},
		map[string]any{"type": "tool_result", "content": "output"},
	}
	got := claudeContentBlocks(content)
	want := []Block{
		{Type: "text", Text: "hello"},
		{Type: "tool_use", Name: "Bash", Input: map[string]any{"command": "ls"}},
	}
	assertBlocksEqual(t, "claude", got, want)
}
