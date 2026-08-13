package main

import (
	"path/filepath"
	"strings"
)

// readClaudeSession parses one Claude Code transcript. Returns nil if the
// file yields no user/assistant events.
func readClaudeSession(path string) *Session {
	var customTitle, aiTitle, cwd, gitBranch, slug, version string
	sessionID := filepath.Base(path)
	sessionID = strings.TrimSuffix(sessionID, filepath.Ext(sessionID))

	var events []Event
	for _, entry := range iterJSONL(path) {
		etype := getString(entry, "type")
		switch etype {
		case "custom-title":
			if t := getString(entry, "customTitle"); t != "" {
				customTitle = t
			}
			continue
		case "ai-title":
			if t := getString(entry, "aiTitle"); t != "" {
				aiTitle = t
			}
			continue
		case "user", "assistant":
		default:
			continue
		}
		if getBool(entry, "isSidechain") {
			continue
		}

		if cwd == "" {
			cwd = getString(entry, "cwd")
		}
		if gitBranch == "" {
			gitBranch = getString(entry, "gitBranch")
		}
		if slug == "" {
			slug = getString(entry, "slug")
		}
		if version == "" {
			version = getString(entry, "version")
		}
		if id := getString(entry, "sessionId"); id != "" {
			sessionID = id
		}

		msg := getMap(entry, "message")
		model := getString(msg, "model")
		if model == "<synthetic>" {
			model = ""
		}
		var usage Usage
		if raw := getMap(msg, "usage"); raw != nil {
			usage = Usage{
				Input:      getInt64(raw, "input_tokens"),
				Output:     getInt64(raw, "output_tokens"),
				CacheRead:  getInt64(raw, "cache_read_input_tokens"),
				CacheWrite: getInt64(raw, "cache_creation_input_tokens"),
			}
		}

		role := "assistant"
		if etype == "user" {
			role = "user"
		}

		events = append(events, Event{
			TS:     parseTS(entry["timestamp"]),
			UUID:   getString(entry, "uuid"),
			Model:  model,
			Usage:  usage,
			Cost:   nil,
			Role:   role,
			Text:   firstText(msg["content"]),
			Blocks: claudeContentBlocks(msg["content"]),
		})
	}

	if len(events) == 0 {
		return nil
	}

	if cwd == "" {
		cwd = decodeSlug(filepath.Base(filepath.Dir(path)))
	}

	name := customTitle
	if name == "" {
		name = aiTitle
	}

	return &Session{
		Harness:     HarnessClaude,
		ID:          sessionID,
		Path:        path,
		CWD:         cwd,
		Name:        name,
		CustomTitle: customTitle,
		AITitle:     aiTitle,
		Slug:        slug,
		Version:     version,
		Provider:    "anthropic",
		GitBranch:   gitBranch,
		Events:      events,
	}
}
