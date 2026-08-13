package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeClaudeFixture(t *testing.T, dir, slug, filename, content string) string {
	t.Helper()
	sessDir := filepath.Join(dir, slug)
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sessDir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertClaudeSessionEqual(t *testing.T, got, want *Session) {
	t.Helper()
	if got.Harness != want.Harness {
		t.Errorf("Harness = %q, want %q", got.Harness, want.Harness)
	}
	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.Path != want.Path {
		t.Errorf("Path = %q, want %q", got.Path, want.Path)
	}
	if got.CWD != want.CWD {
		t.Errorf("CWD = %q, want %q", got.CWD, want.CWD)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	if len(got.Events) != len(want.Events) {
		t.Fatalf("Events len = %d, want %d (got=%+v)", len(got.Events), len(want.Events), got.Events)
	}
	for i := range want.Events {
		g, w := got.Events[i], want.Events[i]
		if !g.TS.Equal(w.TS) {
			t.Errorf("Events[%d].TS = %v, want %v", i, g.TS, w.TS)
		}
		if g.Model != w.Model {
			t.Errorf("Events[%d].Model = %q, want %q", i, g.Model, w.Model)
		}
		if g.Usage != w.Usage {
			t.Errorf("Events[%d].Usage = %+v, want %+v", i, g.Usage, w.Usage)
		}
		if (g.Cost == nil) != (w.Cost == nil) {
			t.Errorf("Events[%d].Cost = %v, want %v", i, g.Cost, w.Cost)
		} else if g.Cost != nil && *g.Cost != *w.Cost {
			t.Errorf("Events[%d].Cost = %v, want %v", i, *g.Cost, *w.Cost)
		}
		if g.Role != w.Role {
			t.Errorf("Events[%d].Role = %q, want %q", i, g.Role, w.Role)
		}
		if g.Text != w.Text {
			t.Errorf("Events[%d].Text = %q, want %q", i, g.Text, w.Text)
		}
	}
}

func TestReadClaudeSession(t *testing.T) {
	t.Run("full fixture", func(t *testing.T) {
		dir := t.TempDir()
		content := `{"type":"custom-title","customTitle":"My Custom Title"}
{"type":"ai-title","aiTitle":"An AI Title"}
{"type":"assistant","isSidechain":true,"cwd":"/Users/x/Code/proj","sessionId":"should-not-win","timestamp":"2026-08-05T10:00:00Z","message":{"model":"claude-sonnet-4-5","content":"sidechain text"}}
{"type":"user","cwd":"/Users/x/Code/proj","sessionId":"uuid1","timestamp":"2026-08-05T10:01:00Z","message":{"content":"hello there"}}
{"type":"assistant","cwd":"/Users/x/Code/proj","sessionId":"uuid1","timestamp":"2026-08-05T10:02:00Z","message":{"model":"claude-sonnet-4-5","content":[{"type":"text","text":"hi back"}],"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":30,"cache_creation_input_tokens":40}}}
{"type":"assistant","cwd":"/Users/x/Code/proj","sessionId":"uuid1","timestamp":"2026-08-05T10:03:00Z","message":{"model":"<synthetic>","content":"synthetic reply"}}
`
		path := writeClaudeFixture(t, dir, "projects/-Users-x-Code-proj", "uuid1.jsonl", content)

		got := readClaudeSession(path)
		if got == nil {
			t.Fatal("readClaudeSession returned nil")
		}

		want := &Session{
			Harness: HarnessClaude,
			ID:      "uuid1",
			Path:    path,
			CWD:     "/Users/x/Code/proj",
			Name:    "My Custom Title",
			Events: []Event{
				{
					TS:    time.Date(2026, 8, 5, 10, 1, 0, 0, time.UTC).In(time.Local),
					Model: "",
					Usage: Usage{},
					Cost:  nil,
					Role:  "user",
					Text:  "hello there",
				},
				{
					TS:    time.Date(2026, 8, 5, 10, 2, 0, 0, time.UTC).In(time.Local),
					Model: "claude-sonnet-4-5",
					Usage: Usage{Input: 10, Output: 20, CacheRead: 30, CacheWrite: 40},
					Cost:  nil,
					Role:  "assistant",
					Text:  "hi back",
				},
				{
					TS:    time.Date(2026, 8, 5, 10, 3, 0, 0, time.UTC).In(time.Local),
					Model: "", // "<synthetic>" maps to ""
					Usage: Usage{},
					Cost:  nil,
					Role:  "assistant",
					Text:  "synthetic reply",
				},
			},
		}

		assertClaudeSessionEqual(t, got, want)
	})

	t.Run("enriched fields", func(t *testing.T) {
		dir := t.TempDir()
		content := `{"type":"custom-title","customTitle":"My Custom Title"}
{"type":"ai-title","aiTitle":"An AI Title"}
{"type":"user","uuid":"u1","cwd":"/Users/x/Code/proj","gitBranch":"develop","version":"1.2.3","slug":"my-slug","sessionId":"uuid1","timestamp":"2026-08-05T10:01:00Z","message":{"content":"hello there"}}
{"type":"assistant","uuid":"a1","cwd":"/Users/x/Code/proj","gitBranch":"feature","sessionId":"uuid1","timestamp":"2026-08-05T10:02:00Z","message":{"model":"claude-sonnet-4-5","content":[{"type":"thinking","thinking":"drop"},{"type":"text","text":"hi back"},{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}
`
		path := writeClaudeFixture(t, dir, "projects/-Users-x-Code-proj", "uuid1.jsonl", content)

		got := readClaudeSession(path)
		if got == nil {
			t.Fatal("readClaudeSession returned nil")
		}
		if got.CustomTitle != "My Custom Title" {
			t.Errorf("CustomTitle = %q, want %q", got.CustomTitle, "My Custom Title")
		}
		if got.AITitle != "An AI Title" {
			t.Errorf("AITitle = %q, want %q", got.AITitle, "An AI Title")
		}
		if got.Provider != "anthropic" {
			t.Errorf("Provider = %q, want %q", got.Provider, "anthropic")
		}
		if got.Slug != "my-slug" {
			t.Errorf("Slug = %q, want %q", got.Slug, "my-slug")
		}
		if got.Version != "1.2.3" {
			t.Errorf("Version = %q, want %q", got.Version, "1.2.3")
		}
		if got.GitBranch != "develop" {
			t.Errorf("GitBranch = %q, want %q (first non-empty wins)", got.GitBranch, "develop")
		}
		if got.Events[0].UUID != "u1" {
			t.Errorf("Events[0].UUID = %q, want %q", got.Events[0].UUID, "u1")
		}
		assertBlocksEqual(t, "user", got.Events[0].Blocks, []Block{{Type: "text", Text: "hello there"}})
		assertBlocksEqual(t, "assistant", got.Events[1].Blocks, []Block{
			{Type: "text", Text: "hi back"},
			{Type: "tool_use", Name: "Bash", Input: map[string]any{"command": "ls"}},
		})
	})

	t.Run("ai-title only", func(t *testing.T) {
		dir := t.TempDir()
		content := `{"type":"ai-title","aiTitle":"Only AI Title"}
{"type":"user","cwd":"/Users/x/Code/proj","sessionId":"uuid2","timestamp":"2026-08-05T10:01:00Z","message":{"content":"hi"}}
`
		path := writeClaudeFixture(t, dir, "projects/-Users-x-Code-proj", "uuid2.jsonl", content)

		got := readClaudeSession(path)
		if got == nil {
			t.Fatal("readClaudeSession returned nil")
		}
		if got.Name != "Only AI Title" {
			t.Errorf("Name = %q, want %q", got.Name, "Only AI Title")
		}
	})

	t.Run("no cwd anywhere falls back to slug", func(t *testing.T) {
		dir := t.TempDir()
		content := `{"type":"user","sessionId":"uuid3","timestamp":"2026-08-05T10:01:00Z","message":{"content":"hi"}}
`
		path := writeClaudeFixture(t, dir, "projects/-Users-x-Code-proj", "uuid3.jsonl", content)

		got := readClaudeSession(path)
		if got == nil {
			t.Fatal("readClaudeSession returned nil")
		}
		want := "/Users/x/Code/proj"
		if got.CWD != want {
			t.Errorf("CWD = %q, want %q", got.CWD, want)
		}
	})

	t.Run("empty file returns nil", func(t *testing.T) {
		dir := t.TempDir()
		path := writeClaudeFixture(t, dir, "projects/-Users-x-Code-proj", "uuid4.jsonl", "")

		got := readClaudeSession(path)
		if got != nil {
			t.Errorf("readClaudeSession = %+v, want nil", got)
		}
	})
}
