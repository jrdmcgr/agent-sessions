package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// subagentsDir returns the directory Claude Code writes a session's spawned
// subagent transcripts to: a sibling directory named after the session file
// (extension stripped), holding "agent-<id>.jsonl" + "agent-<id>.meta.json"
// pairs. Does not check existence.
func subagentsDir(sessionPath string) string {
	base := strings.TrimSuffix(sessionPath, filepath.Ext(sessionPath))
	return filepath.Join(base, "subagents")
}

// readSubagents reads every agent-*.jsonl in a session's subagents
// directory (if any) into one SubagentTranscript each, sorted by filename
// (which is also spawn order: Claude Code names them sequentially). Returns
// nil if the directory is absent or holds nothing readable.
func readSubagents(sessionPath string) []SubagentTranscript {
	matches, err := filepath.Glob(filepath.Join(subagentsDir(sessionPath), "agent-*.jsonl"))
	if err != nil || len(matches) == 0 {
		return nil
	}
	sort.Strings(matches)

	var out []SubagentTranscript
	for _, path := range matches {
		if t := readSubagentTranscript(path); t != nil {
			out = append(out, *t)
		}
	}
	return out
}

// readSubagentTranscript parses one subagent transcript file plus its
// sibling .meta.json for identifying fields (name/description/agentType set
// by the tool call that spawned it). Returns nil if the transcript carries
// no user/assistant turns. Subagent transcript lines carry the same
// "message.usage" shape as a top-level Claude Code transcript (they are, in
// effect, isSidechain turns written to their own file instead of inline), so
// the per-line extraction mirrors readClaudeSession's.
func readSubagentTranscript(path string) *SubagentTranscript {
	id := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "agent-"), ".jsonl")
	t := &SubagentTranscript{ID: id, Path: path, Priced: true}

	metaPath := strings.TrimSuffix(path, ".jsonl") + ".meta.json"
	if meta := readSubagentMeta(metaPath); meta != nil {
		t.Name = getString(meta, "name")
		t.Description = getString(meta, "description")
		t.AgentType = getString(meta, "agentType")
	}

	seenModel := map[string]bool{}
	seenUnpriced := map[string]bool{}
	for _, entry := range iterJSONL(path) {
		switch getString(entry, "type") {
		case "assistant", "user":
		default:
			continue // fork-context-ref, attachment, etc. carry no usage
		}
		t.Messages++

		if ts := parseTS(entry["timestamp"]); !ts.IsZero() {
			if t.Start.IsZero() || ts.Before(t.Start) {
				t.Start = ts
			}
			if t.End.IsZero() || ts.After(t.End) {
				t.End = ts
			}
		}

		msg := getMap(entry, "message")
		model := getString(msg, "model")
		if model == "<synthetic>" {
			model = ""
		}
		if model != "" && !seenModel[model] {
			seenModel[model] = true
			t.Models = append(t.Models, model)
		}

		raw := getMap(msg, "usage")
		if raw == nil {
			continue
		}
		usage := Usage{
			Input:      getInt64(raw, "input_tokens"),
			Output:     getInt64(raw, "output_tokens"),
			CacheRead:  getInt64(raw, "cache_read_input_tokens"),
			CacheWrite: getInt64(raw, "cache_creation_input_tokens"),
		}
		t.Usage.Add(usage)
		if !usage.Any() {
			continue
		}
		if cost := price(model, usage); cost != nil {
			t.Cost += *cost
		} else {
			t.Priced = false
			if !seenUnpriced[model] {
				seenUnpriced[model] = true
				t.Unpriced = append(t.Unpriced, model)
			}
		}
	}

	if t.Messages == 0 {
		return nil
	}
	return t
}

func readSubagentMeta(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}

// subagentDay is the calendar day a subagent transcript's activity belongs
// to, for folding its cost into the right day bucket. Zero time if the
// transcript carries no timestamps.
func subagentDay(t SubagentTranscript) time.Time {
	if t.Start.IsZero() {
		return time.Time{}
	}
	return dayOf(t.Start)
}
