package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// decodeRecord runs `sessions show` over a fixture and decodes the JSON.
func decodeRecord(t *testing.T, argv []string) (sessionRecord, int) {
	t.Helper()
	var out, errBuf bytes.Buffer
	code := runShow(argv, &out, &errBuf)
	var rec sessionRecord
	if out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &rec); err != nil {
			t.Fatalf("invalid JSON: %v\noutput: %s\nstderr: %s", err, out.String(), errBuf.String())
		}
	}
	return rec, code
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestShowClaudeRecord: metadata, counts, summary, and messages[] for a Claude
// transcript. The command-noise user turn is excluded from renderable_count and
// messages[]; the empty-text assistant turn too; message_count counts them.
func TestShowClaudeRecord(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "projects", "-Users-x-Code-proj")
	content := `{"type":"custom-title","customTitle":"My Title"}
{"type":"user","uuid":"u0","cwd":"/Users/x/Code/proj","gitBranch":"main","version":"1.2.3","slug":"the-slug","sessionId":"sid","timestamp":"2026-08-05T10:00:00Z","message":{"content":"<command-name>/clear</command-name>"}}
{"type":"user","uuid":"u1","sessionId":"sid","timestamp":"2026-08-05T10:01:00Z","message":{"content":"Please <b>refactor</b> the parser"}}
{"type":"assistant","uuid":"a1","sessionId":"sid","timestamp":"2026-08-05T10:02:00Z","message":{"model":"claude-sonnet-4-5","content":[{"type":"text","text":"on it"},{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}
{"type":"assistant","uuid":"a2","sessionId":"sid","timestamp":"2026-08-05T10:03:00Z","message":{"model":"claude-sonnet-4-5","content":[{"type":"text","text":"   "}]}}
`
	path := writeFile(t, dir, "sid.jsonl", content)

	rec, code := decodeRecord(t, []string{path, "--messages"})
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	local := func(u string) string {
		return parseTS(u).Format("2006-01-02T15:04:05")
	}
	checks := map[string]struct{ got, want string }{
		"harness":      {rec.Harness, "claude"},
		"session_id":   {rec.SessionID, "sid"},
		"project":      {rec.Project, "proj"},
		"git_branch":   {rec.GitBranch, "main"},
		"provider":     {rec.Provider, "anthropic"},
		"slug":         {rec.Slug, "the-slug"},
		"version":      {rec.Version, "1.2.3"},
		"custom_title": {rec.CustomTitle, "My Title"},
		"name":         {rec.Name, "My Title"},
		"summary":      {rec.Summary, "Please refactor the parser"},
		"started_at":   {rec.StartedAt, local("2026-08-05T10:00:00Z")},
		"ended_at":     {rec.EndedAt, local("2026-08-05T10:03:00Z")},
		"model":        {rec.Model, "sonnet-4-5"},
	}
	for field, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", field, c.got, c.want)
		}
	}
	if rec.MessageCount != 4 {
		t.Errorf("message_count = %d, want 4 (all user/assistant events)", rec.MessageCount)
	}
	if rec.RenderableCount != 2 {
		t.Errorf("renderable_count = %d, want 2 (noise + empty-text dropped)", rec.RenderableCount)
	}
	if len(rec.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(rec.Messages))
	}
	if rec.Messages[0].UUID != "u1" || rec.Messages[0].Role != "user" {
		t.Errorf("messages[0] = %+v, want uuid u1 / user", rec.Messages[0])
	}
	assertBlocksEqual(t, "asst", rec.Messages[1].Blocks, []Block{
		{Type: "text", Text: "on it"},
		{Type: "tool_use", Name: "Bash", Input: map[string]any{"command": "ls"}},
	})
	if rec.Messages[1].Model != "sonnet-4-5" {
		t.Errorf("messages[1].model = %q, want sonnet-4-5", rec.Messages[1].Model)
	}
}

// TestShowOmitsMessagesByDefault: without --messages the array is absent.
func TestShowOmitsMessagesByDefault(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "projects", "-Users-x-Code-proj")
	content := `{"type":"user","uuid":"u1","sessionId":"sid","timestamp":"2026-08-05T10:01:00Z","message":{"content":"hi"}}
`
	path := writeFile(t, dir, "sid.jsonl", content)

	var out, errBuf bytes.Buffer
	if code := runShow([]string{path}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d, stderr=%s", code, errBuf.String())
	}
	if strings.Contains(out.String(), "\"messages\"") {
		t.Errorf("default output should omit messages[]:\n%s", out.String())
	}
	if rec, _ := decodeRecord(t, []string{path}); rec.RenderableCount != 1 {
		t.Errorf("renderable_count = %d, want 1", rec.RenderableCount)
	}
}

// TestShowPiHarnessDetection: a pi transcript is detected and its toolCall
// blocks are normalized to Claude spelling.
func TestShowPiHarnessDetection(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "--Users-x-proj--")
	content := `{"type":"session","version":3,"cwd":"/Users/x/proj","id":"pid","timestamp":"2026-08-05T14:00:00Z"}
{"type":"message","id":"m1","timestamp":"2026-08-05T14:00:01Z","message":{"role":"user","content":"read the file"}}
{"type":"message","id":"m2","timestamp":"2026-08-05T14:00:02Z","message":{"role":"assistant","provider":"anthropic","model":"claude-opus-4","content":[{"type":"toolCall","name":"read","arguments":{"path":"/etc/hosts"}}]}}
{"type":"message","id":"m3","timestamp":"2026-08-05T14:00:03Z","message":{"role":"toolResult","content":[{"type":"text","text":"file contents"}]}}
`
	path := writeFile(t, dir, "2026-08-05_pid.jsonl", content)

	rec, code := decodeRecord(t, []string{path, "--messages"})
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if rec.Harness != "pi" {
		t.Fatalf("harness = %q, want pi", rec.Harness)
	}
	// toolResult is its own pi message entry: counted, but not conversation.
	if rec.MessageCount != 3 {
		t.Errorf("message_count = %d, want 3 (incl. toolResult)", rec.MessageCount)
	}
	if rec.RenderableCount != 2 {
		t.Errorf("renderable_count = %d, want 2 (toolResult excluded)", rec.RenderableCount)
	}
	if len(rec.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2 (toolResult excluded)", len(rec.Messages))
	}
	for _, m := range rec.Messages {
		if m.Role == "toolResult" {
			t.Errorf("toolResult leaked into messages[]")
		}
	}
	assertBlocksEqual(t, "pi-asst", rec.Messages[1].Blocks, []Block{
		{Type: "tool_use", Name: "Read", Input: map[string]any{"file_path": "/etc/hosts"}},
	})
}

// TestShowMissingFile: a nonexistent path exits non-zero without a record.
func TestShowMissingFile(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := runShow([]string{"/no/such/file.jsonl"}, &out, &errBuf); code == 0 {
		t.Errorf("expected non-zero exit for missing file")
	}
}
