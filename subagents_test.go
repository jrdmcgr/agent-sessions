package main

import (
	"path/filepath"
	"testing"
)

// writeSubagent writes a subagent transcript + its .meta.json sibling under
// <sessionPath-without-ext>/subagents/agent-<id>.jsonl, mirroring the layout
// Claude Code writes for a Task/Agent tool call.
func writeSubagent(t *testing.T, sessionPath, id, meta, transcript string) {
	t.Helper()
	dir := subagentsDir(sessionPath)
	writeFile(t, dir, "agent-"+id+".jsonl", transcript)
	if meta != "" {
		writeFile(t, dir, "agent-"+id+".meta.json", meta)
	}
}

func TestReadSubagentsAbsentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sid.jsonl")
	if got := readSubagents(path); got != nil {
		t.Errorf("readSubagents on absent dir = %v, want nil", got)
	}
}

func TestReadSubagentTranscript(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "sid.jsonl")

	meta := `{"agentType":"fork","description":"Bug-trace batch 1","name":"bugtrace-batch1","model":"inherit"}`
	transcript := `{"type":"fork-context-ref","agentId":"a1"}
{"type":"assistant","isSidechain":true,"agentId":"a1","timestamp":"2026-09-22T12:50:00Z","message":{"model":"claude-sonnet-5","usage":{"input_tokens":2,"cache_read_input_tokens":100000,"cache_creation_input_tokens":300,"output_tokens":50}}}
{"type":"user","isSidechain":true,"agentId":"a1","timestamp":"2026-09-22T12:50:05Z","message":{"content":"tool result"}}
{"type":"assistant","isSidechain":true,"agentId":"a1","timestamp":"2026-09-22T12:51:00Z","message":{"model":"claude-sonnet-5","usage":{"input_tokens":2,"cache_read_input_tokens":150000,"output_tokens":200}}}
`
	writeSubagent(t, sessionPath, "a1", meta, transcript)

	subs := readSubagents(sessionPath)
	if len(subs) != 1 {
		t.Fatalf("got %d subagents, want 1", len(subs))
	}
	s := subs[0]

	if s.ID != "a1" {
		t.Errorf("ID = %q, want a1", s.ID)
	}
	if s.Name != "bugtrace-batch1" {
		t.Errorf("Name = %q, want bugtrace-batch1", s.Name)
	}
	if s.Description != "Bug-trace batch 1" {
		t.Errorf("Description = %q, want %q", s.Description, "Bug-trace batch 1")
	}
	if s.AgentType != "fork" {
		t.Errorf("AgentType = %q, want fork", s.AgentType)
	}
	// 2 assistant + 1 user turn -> 3 messages; only assistant turns carry usage.
	if s.Messages != 3 {
		t.Errorf("Messages = %d, want 3", s.Messages)
	}
	wantUsage := Usage{Input: 4, Output: 250, CacheRead: 250000, CacheWrite: 300}
	if s.Usage != wantUsage {
		t.Errorf("Usage = %+v, want %+v", s.Usage, wantUsage)
	}
	if !s.Priced {
		t.Errorf("Priced = false, want true")
	}
	wantCost := *price("claude-sonnet-5", wantUsage)
	if s.Cost != wantCost {
		t.Errorf("Cost = %v, want %v", s.Cost, wantCost)
	}
	if len(s.Models) != 1 || s.Models[0] != "claude-sonnet-5" {
		t.Errorf("Models = %v, want [claude-sonnet-5]", s.Models)
	}
}

func TestReadSubagentTranscriptMissingMeta(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "sid.jsonl")
	transcript := `{"type":"assistant","isSidechain":true,"agentId":"a1","timestamp":"2026-09-22T12:50:00Z","message":{"model":"claude-sonnet-5","usage":{"input_tokens":1,"output_tokens":1}}}
`
	writeSubagent(t, sessionPath, "a1", "", transcript)

	subs := readSubagents(sessionPath)
	if len(subs) != 1 {
		t.Fatalf("got %d subagents, want 1", len(subs))
	}
	if subs[0].Name != "" || subs[0].Description != "" {
		t.Errorf("expected empty Name/Description with no meta.json, got %+v", subs[0])
	}
}

func TestReadSubagentTranscriptUnpricedModel(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "sid.jsonl")
	transcript := `{"type":"assistant","isSidechain":true,"agentId":"a1","timestamp":"2026-09-22T12:50:00Z","message":{"model":"some-unknown-model","usage":{"input_tokens":5,"output_tokens":1}}}
`
	writeSubagent(t, sessionPath, "a1", "", transcript)

	subs := readSubagents(sessionPath)
	s := subs[0]
	if s.Priced {
		t.Errorf("Priced = true, want false")
	}
	if len(s.Unpriced) != 1 || s.Unpriced[0] != "some-unknown-model" {
		t.Errorf("Unpriced = %v, want [some-unknown-model]", s.Unpriced)
	}
}

func TestReadSubagentTranscriptNoMessagesReturnsNil(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "sid.jsonl")
	transcript := `{"type":"fork-context-ref","agentId":"a1"}
{"type":"attachment","agentId":"a1","attachment":{}}
`
	writeSubagent(t, sessionPath, "a1", "", transcript)

	if subs := readSubagents(sessionPath); subs != nil {
		t.Errorf("readSubagents = %v, want nil (no user/assistant turns)", subs)
	}
}

func TestReadSubagentsMultipleSortedByID(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "sid.jsonl")
	line := `{"type":"assistant","isSidechain":true,"timestamp":"2026-09-22T12:50:00Z","message":{"model":"claude-sonnet-5","usage":{"input_tokens":1,"output_tokens":1}}}
`
	writeSubagent(t, sessionPath, "b2", `{"description":"second"}`, line)
	writeSubagent(t, sessionPath, "a1", `{"description":"first"}`, line)

	subs := readSubagents(sessionPath)
	if len(subs) != 2 {
		t.Fatalf("got %d subagents, want 2", len(subs))
	}
	if subs[0].Description != "first" || subs[1].Description != "second" {
		t.Errorf("order = %q, %q; want first, second", subs[0].Description, subs[1].Description)
	}
}

func TestClaudeSessionPicksUpSubagents(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "projects", "-Users-x-Code-proj")
	main := `{"type":"user","uuid":"u1","cwd":"/Users/x/Code/proj","sessionId":"sid","timestamp":"2026-09-22T12:00:00Z","message":{"content":"go run the batches"}}
{"type":"assistant","uuid":"a1","sessionId":"sid","timestamp":"2026-09-22T12:00:10Z","message":{"model":"claude-sonnet-5","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":10,"output_tokens":5}}}
`
	sessionPath := writeFile(t, dir, "sid.jsonl", main)

	sub := `{"type":"assistant","isSidechain":true,"timestamp":"2026-09-22T12:00:20Z","message":{"model":"claude-sonnet-5","usage":{"input_tokens":1000000,"output_tokens":0}}}
`
	writeSubagent(t, sessionPath, "a1", `{"description":"batch 1"}`, sub)

	s := readClaudeSession(sessionPath)
	if s == nil {
		t.Fatal("readClaudeSession returned nil")
	}
	if len(s.Subagents) != 1 {
		t.Fatalf("got %d subagents, want 1", len(s.Subagents))
	}
	if s.Subagents[0].Description != "batch 1" {
		t.Errorf("Description = %q, want %q", s.Subagents[0].Description, "batch 1")
	}
}
