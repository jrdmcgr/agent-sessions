package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writePiFixture(t *testing.T, dir, slug, filename, content string) string {
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

func piFloatPtr(f float64) *float64 { return &f }

// TestReadPiSessionFull exercises the ID/cwd/model-stickiness/name/usage/cost/
// events behaviors together against one realistic fixture.
func TestReadPiSessionFull(t *testing.T) {
	dir := t.TempDir()
	content := `{"type":"session","cwd":"/Users/jared/proj","id":"abc123","timestamp":"2026-08-05T14:00:00Z"}
{"type":"model_change","modelId":"claude-opus-4","timestamp":"2026-08-05T14:00:01Z"}
{"type":"session_info","name":"My Session","timestamp":"2026-08-05T14:00:02Z"}
{"type":"message","timestamp":"2026-08-05T14:00:03Z","message":{"role":"assistant","model":"claude-opus-4","content":"first reply","usage":{"input":10,"output":20,"cacheRead":5,"cacheWrite":1,"cost":{"total":0.0123}}}}
{"type":"message","timestamp":"2026-08-05T14:00:04Z","message":{"role":"assistant","content":"second reply","usage":{"input":3,"output":4,"cacheRead":0,"cacheWrite":0}}}
{"type":"message","timestamp":"2026-08-05T14:00:05Z","message":{"role":"user","content":[{"type":"tool_use","text":"ignore me"},{"type":"text","text":"hello world"}]}}
`
	path := writePiFixture(t, dir, "someslug", "2026-08-05_sess1.jsonl", content)

	got := readPiSession(path)
	if got == nil {
		t.Fatal("expected non-nil session")
	}

	want := &Session{
		Harness: HarnessPi,
		ID:      "abc123",
		Path:    path,
		CWD:     "/Users/jared/proj",
		Name:    "My Session",
		Events: []Event{
			{
				TS:    time.Date(2026, 8, 5, 14, 0, 3, 0, time.UTC).In(time.Local),
				Model: "claude-opus-4",
				Usage: Usage{Input: 10, Output: 20, CacheRead: 5, CacheWrite: 1},
				Cost:  piFloatPtr(0.0123),
				Role:  "assistant",
				Text:  "first reply",
			},
			{
				TS:    time.Date(2026, 8, 5, 14, 0, 4, 0, time.UTC).In(time.Local),
				Model: "claude-opus-4", // sticky: inherited, message had no model
				Usage: Usage{Input: 3, Output: 4, CacheRead: 0, CacheWrite: 0},
				Cost:  nil,
				Role:  "assistant",
				Text:  "second reply",
			},
			{
				TS:    time.Date(2026, 8, 5, 14, 0, 5, 0, time.UTC).In(time.Local),
				Model: "claude-opus-4", // sticky: no usage/model on this message either
				Usage: Usage{},
				Cost:  nil,
				Role:  "user",
				Text:  "hello world",
			},
		},
	}

	assertPiSessionEqual(t, got, want)
}

// TestReadPiSessionEnriched covers the Phase-1 additions: per-message UUID
// (pi entry "id"), normalized content Blocks (text + tool_use with the
// pi->Claude name/arg remap, thinking dropped), sticky Provider, and
// CustomTitle from session_info. GitBranch comes from a "git-branch" custom
// entry (last wins).
func TestReadPiSessionEnriched(t *testing.T) {
	dir := t.TempDir()
	content := `{"type":"session","cwd":"/Users/jared/proj","id":"abc123","timestamp":"2026-08-05T14:00:00Z"}
{"type":"session_info","name":"My Session","timestamp":"2026-08-05T14:00:01Z"}
{"type":"custom","customType":"git-branch","data":{"branch":"feature-x"},"timestamp":"2026-08-05T14:00:02Z"}
{"type":"message","id":"m1","timestamp":"2026-08-05T14:00:03Z","message":{"role":"user","content":"hello"}}
{"type":"message","id":"m2","timestamp":"2026-08-05T14:00:04Z","message":{"role":"assistant","provider":"anthropic","model":"claude-opus-4","content":[{"type":"thinking","thinking":"drop me"},{"type":"text","text":"on it"},{"type":"toolCall","id":"t1","name":"read","arguments":{"path":"/etc/hosts"}}]}}
{"type":"custom","customType":"git-branch","data":{"branch":"main"},"timestamp":"2026-08-05T14:00:05Z"}
`
	path := writePiFixture(t, dir, "someslug", "2026-08-05_enrich.jsonl", content)

	got := readPiSession(path)
	if got == nil {
		t.Fatal("expected non-nil session")
	}
	if got.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", got.Provider, "anthropic")
	}
	if got.CustomTitle != "My Session" {
		t.Errorf("CustomTitle = %q, want %q", got.CustomTitle, "My Session")
	}
	if got.GitBranch != "main" {
		t.Errorf("GitBranch = %q, want %q (last git-branch entry wins)", got.GitBranch, "main")
	}
	if len(got.Events) != 2 {
		t.Fatalf("Events len = %d, want 2", len(got.Events))
	}
	if got.Events[0].UUID != "m1" {
		t.Errorf("Events[0].UUID = %q, want %q", got.Events[0].UUID, "m1")
	}
	wantUser := []Block{{Type: "text", Text: "hello"}}
	assertBlocksEqual(t, "user", got.Events[0].Blocks, wantUser)
	wantAsst := []Block{
		{Type: "text", Text: "on it"},
		{Type: "tool_use", Name: "Read", Input: map[string]any{"file_path": "/etc/hosts"}},
	}
	assertBlocksEqual(t, "assistant", got.Events[1].Blocks, wantAsst)
}

// TestReadPiSessionIDFallbackNoUnderscore covers behavior 1: a stem with no
// "_" uses the whole stem as the ID, and no session entry means it is never
// overridden.
func TestReadPiSessionIDFallbackNoUnderscore(t *testing.T) {
	dir := t.TempDir()
	content := `{"type":"message","timestamp":"2026-08-05T14:00:00Z","message":{"role":"user","content":"hi"}}
`
	path := writePiFixture(t, dir, "-Users-jared-proj", "sessiononly.jsonl", content)

	got := readPiSession(path)
	if got == nil {
		t.Fatal("expected non-nil session")
	}
	if got.ID != "sessiononly" {
		t.Errorf("ID = %q, want %q", got.ID, "sessiononly")
	}
}

// TestReadPiSessionCWDFallback covers behavior 2: no session entry sets cwd,
// so it falls back to decodeSlug of the parent directory name.
func TestReadPiSessionCWDFallback(t *testing.T) {
	dir := t.TempDir()
	content := `{"type":"message","timestamp":"2026-08-05T14:00:00Z","message":{"role":"user","content":"hi"}}
`
	path := writePiFixture(t, dir, "-Users-jared-proj", "2026-08-05_xyz.jsonl", content)

	got := readPiSession(path)
	if got == nil {
		t.Fatal("expected non-nil session")
	}
	want := "/Users/jared/proj"
	if got.CWD != want {
		t.Errorf("CWD = %q, want %q", got.CWD, want)
	}
	if got.ID != "xyz" {
		t.Errorf("ID = %q, want %q", got.ID, "xyz")
	}
}

// TestReadPiSessionEmpty covers behavior 7: a file with only session/
// model_change entries (no message entries) returns nil.
func TestReadPiSessionEmpty(t *testing.T) {
	dir := t.TempDir()
	content := `{"type":"session","cwd":"/Users/jared/proj","id":"abc123","timestamp":"2026-08-05T14:00:00Z"}
{"type":"model_change","modelId":"claude-opus-4","timestamp":"2026-08-05T14:00:01Z"}
`
	path := writePiFixture(t, dir, "someslug", "2026-08-05_sess2.jsonl", content)

	got := readPiSession(path)
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

// TestReadPiSessionUnreadable covers behavior 7's other half: an unreadable
// (nonexistent) file returns nil.
func TestReadPiSessionUnreadable(t *testing.T) {
	got := readPiSession("/nonexistent/dir/2026-08-05_nope.jsonl")
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func assertPiSessionEqual(t *testing.T, got, want *Session) {
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
