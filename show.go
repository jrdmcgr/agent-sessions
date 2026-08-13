package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// sessionRecord is the per-session JSON emitted by `sessions show`. It is the
// contract both consumers (note-archiver, memory-logger) bind to. Field order
// here is the emitted key order. See docs/plans/002-extract-archive-session.md.
type sessionRecord struct {
	Harness         string          `json:"harness"`
	SessionID       string          `json:"session_id"`
	Path            string          `json:"path"`
	CWD             string          `json:"cwd"`
	Project         string          `json:"project"`
	GitBranch       string          `json:"git_branch"`
	Provider        string          `json:"provider"`
	Model           string          `json:"model"`
	Models          []string        `json:"models"`
	Slug            string          `json:"slug"`
	Version         string          `json:"version"`
	CustomTitle     string          `json:"custom_title"`
	AITitle         string          `json:"ai_title"`
	Name            string          `json:"name"`
	Summary         string          `json:"summary"`
	StartedAt       string          `json:"started_at"`
	EndedAt         string          `json:"ended_at"`
	MessageCount    int             `json:"message_count"`
	RenderableCount int             `json:"renderable_count"`
	Usage           Usage           `json:"usage"`
	Tokens          int64           `json:"tokens"`
	Cost            float64         `json:"cost"`
	Priced          bool            `json:"priced"`
	Messages        []messageRecord `json:"messages,omitempty"`
}

// messageRecord is one renderable user/assistant message in a sessionRecord.
type messageRecord struct {
	UUID   string  `json:"uuid"`
	Role   string  `json:"role"`
	TS     string  `json:"ts"`
	Model  string  `json:"model,omitempty"`
	Blocks []Block `json:"blocks"`
}

// detectHarness reads a transcript and decides which harness wrote it. Ports
// archive-session's detect_harness: pi opens with a `session` header carrying a
// `version`, or a `message` entry whose message has a role but no top-level
// `uuid`; a top-level `uuid`/`sessionId` marks Claude Code. Defaults to Claude.
func detectHarness(path string) string {
	for _, entry := range iterJSONL(path) {
		switch getString(entry, "type") {
		case "session":
			if _, ok := entry["version"]; ok {
				return HarnessPi
			}
		case "message":
			if msg := getMap(entry, "message"); msg != nil {
				if _, hasRole := msg["role"]; hasRole {
					if _, hasUUID := entry["uuid"]; !hasUUID {
						return HarnessPi
					}
				}
			}
		}
		if getString(entry, "uuid") != "" || getString(entry, "sessionId") != "" {
			return HarnessClaude
		}
	}
	return HarnessClaude
}

// readSessionFile detects the harness of a single transcript and parses it.
func readSessionFile(path string) *Session {
	if detectHarness(path) == HarnessPi {
		return readPiSession(path)
	}
	return readClaudeSession(path)
}

// isCommandNoise reports whether a user body is just a slash-command wrapper
// (e.g. /clear output). Ports archive-session's is_command_noise. Distinct from
// NoisePrefixes, which fallbackName uses for a broader skip.
func isCommandNoise(text string) bool {
	s := strings.TrimSpace(text)
	return strings.HasPrefix(s, "<command-name>") ||
		strings.HasPrefix(s, "<local-command-") ||
		strings.HasPrefix(s, "<command-message>")
}

// tagRe-free tag stripping: extract_summary removes any <...> spans and
// collapses whitespace. We avoid regexp to keep the dependency surface flat.
func stripTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// eventBody joins an event's text blocks with spaces.
func eventBody(ev Event) string {
	var parts []string
	for _, b := range ev.Blocks {
		if b.Type == "text" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, " ")
}

// extractSummary returns the first substantive user message as a one-line
// summary: skip command-noise, strip tags, collapse whitespace, truncate 120.
// Ports archive-session's extract_summary.
func extractSummary(s *Session) string {
	for _, ev := range s.Events {
		if ev.Role != "user" {
			continue
		}
		body := strings.TrimSpace(eventBody(ev))
		if body == "" || isCommandNoise(body) {
			continue
		}
		body = strings.Join(strings.Fields(stripTags(body)), " ")
		if body != "" {
			return truncateRunes(body, 120)
		}
	}
	return ""
}

// truncateRunes truncates to n runes, appending an ellipsis when it cuts.
// Mirrors archive-session's truncate (n-1 chars + "…").
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// renderableBlocks drops empty text blocks; tool_use blocks always survive.
func renderableBlocks(blocks []Block) []Block {
	var out []Block
	for _, b := range blocks {
		if b.Type == "text" {
			if strings.TrimSpace(b.Text) == "" {
				continue
			}
			out = append(out, b)
		} else {
			out = append(out, b)
		}
	}
	return out
}

// buildRecord aggregates a whole session (not per-day) into the emitted record.
// withMessages controls whether the messages[] array is populated.
func buildRecord(s *Session, withMessages bool) sessionRecord {
	rec := sessionRecord{
		Harness:     s.Harness,
		SessionID:   s.ID,
		Path:        s.Path,
		CWD:         s.CWD,
		Project:     projectName(s.CWD),
		GitBranch:   s.GitBranch,
		Provider:    s.Provider,
		Slug:        s.Slug,
		Version:     s.Version,
		CustomTitle: s.CustomTitle,
		AITitle:     s.AITitle,
		Summary:     extractSummary(s),
		Models:      []string{},
		Priced:      true,
	}
	rec.Name = s.Name
	if rec.Name == "" {
		rec.Name = fallbackName(s)
	}

	seenModel := map[string]bool{}
	var first, last time.Time
	lastModel := ""
	for _, ev := range s.Events {
		if !ev.TS.IsZero() {
			if first.IsZero() || ev.TS.Before(first) {
				first = ev.TS
			}
			if last.IsZero() || ev.TS.After(last) {
				last = ev.TS
			}
		}
		rec.MessageCount++
		rec.Usage.Add(ev.Usage)
		if ev.Model != "" {
			lastModel = ev.Model
			short := shortModel(ev.Model)
			if !seenModel[short] {
				seenModel[short] = true
				rec.Models = append(rec.Models, short)
			}
		}
		if ev.Usage.Any() {
			cost := ev.Cost
			if cost == nil {
				cost = price(ev.Model, ev.Usage)
			}
			if cost == nil {
				rec.Priced = false
			} else {
				rec.Cost += *cost
			}
		}

		// Only user/assistant turns are conversation. pi records toolResult,
		// bashExecution, etc. as their own message entries (they count toward
		// message_count above, matching the table) but carry no dialogue, so
		// archive-session drops them from the rendered body and so do we.
		if ev.Role != "user" && ev.Role != "assistant" {
			continue
		}
		blocks := renderableBlocks(ev.Blocks)
		if len(blocks) == 0 {
			continue
		}
		if ev.Role == "user" && isCommandNoise(eventBody(ev)) {
			continue
		}
		rec.RenderableCount++
		if withMessages {
			mr := messageRecord{UUID: ev.UUID, Role: ev.Role, Blocks: blocks}
			if !ev.TS.IsZero() {
				mr.TS = ev.TS.Format("2006-01-02T15:04:05")
			}
			if ev.Role == "assistant" {
				mr.Model = shortModel(ev.Model)
			}
			rec.Messages = append(rec.Messages, mr)
		}
	}
	rec.Tokens = rec.Usage.Total()
	if lastModel != "" {
		rec.Model = shortModel(lastModel)
	}
	if !first.IsZero() {
		rec.StartedAt = first.Format("2006-01-02T15:04:05")
	}
	if !last.IsZero() {
		rec.EndedAt = last.Format("2006-01-02T15:04:05")
	}
	return rec
}

// projectName is basename(cwd), falling back to the whole cwd for the
// degenerate roots, matching sessionDays.
func projectName(cwd string) string {
	p := filepath.Base(cwd)
	if p == "" || p == "/" || p == "." {
		return cwd
	}
	return p
}

// runShow implements `sessions show <path> [--messages]`, emitting one
// per-session JSON record. Returns an exit code.
func runShow(argv []string, stdout, stderr io.Writer) int {
	var path string
	withMessages := false
	for _, arg := range argv {
		switch {
		case arg == "--messages":
			withMessages = true
		case arg == "-h" || arg == "--help":
			fmt.Fprintln(stdout, "usage: sessions show <transcript.jsonl> [--messages]")
			return 0
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(stderr, "unrecognized argument: %s\n", arg)
			return 2
		default:
			if path != "" {
				fmt.Fprintf(stderr, "unexpected argument: %s\n", arg)
				return 2
			}
			path = arg
		}
	}
	if path == "" {
		fmt.Fprintln(stderr, "usage: sessions show <transcript.jsonl> [--messages]")
		return 2
	}
	if _, err := os.Stat(path); err != nil {
		fmt.Fprintf(stderr, "cannot read transcript: %v\n", err)
		return 1
	}

	s := readSessionFile(path)
	if s == nil {
		fmt.Fprintf(stderr, "no conversational content in %s\n", path)
		return 1
	}
	rec := buildRecord(s, withMessages)

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rec); err != nil {
		fmt.Fprintf(stderr, "encode failed: %v\n", err)
		return 1
	}
	return 0
}
