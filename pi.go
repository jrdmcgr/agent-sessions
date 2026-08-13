package main

import (
	"path/filepath"
	"strings"
)

// readPiSession parses one pi transcript. Returns nil if the file yields no
// message events. Ports read_pi_session.
func readPiSession(path string) *Session {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	sessionID := stem
	if idx := strings.Index(stem, "_"); idx >= 0 {
		sessionID = stem[idx+1:]
	}

	var cwd, name, model, provider, gitBranch string
	var events []Event

	for _, entry := range iterJSONL(path) {
		etype := getString(entry, "type")
		ts := parseTS(entry["timestamp"])

		switch etype {
		case "session":
			if v := getString(entry, "cwd"); v != "" {
				cwd = v
			}
			if v := getString(entry, "id"); v != "" {
				sessionID = v
			}
		case "model_change":
			if v := getString(entry, "modelId"); v != "" {
				model = v
			}
			if v := getString(entry, "provider"); v != "" {
				provider = v
			}
		case "session_info":
			if v := getString(entry, "name"); v != "" {
				name = v
			}
		case "custom":
			// git-branch entries are stamped by the pi extension (see plan 002);
			// last one wins so GitBranch tracks mid-session changes.
			if getString(entry, "customType") == "git-branch" {
				if v := getString(getMap(entry, "data"), "branch"); v != "" {
					gitBranch = v
				}
			}
		case "message":
			msg := getMap(entry, "message")
			if v := getString(msg, "model"); v != "" {
				model = v
			}
			if v := getString(msg, "provider"); v != "" {
				provider = v
			}
			usage := Usage{}
			var cost *float64
			if raw := getMap(msg, "usage"); raw != nil {
				usage = Usage{
					Input:      getInt64(raw, "input"),
					Output:     getInt64(raw, "output"),
					CacheRead:  getInt64(raw, "cacheRead"),
					CacheWrite: getInt64(raw, "cacheWrite"),
				}
				if costMap := getMap(raw, "cost"); costMap != nil {
					if total, ok := costMap["total"].(float64); ok {
						cost = &total
					}
				}
			}
			events = append(events, Event{
				TS:     ts,
				TSRaw:  getString(entry, "timestamp"),
				UUID:   getString(entry, "id"),
				Model:  model,
				Usage:  usage,
				Cost:   cost,
				Role:   getString(msg, "role"),
				Text:   firstText(msg["content"]),
				Blocks: piContentBlocks(msg["content"]),
			})
		}
	}

	if len(events) == 0 {
		return nil
	}

	if cwd == "" {
		cwd = decodeSlug(filepath.Base(filepath.Dir(path)))
	}

	return &Session{
		Harness:     HarnessPi,
		ID:          sessionID,
		Path:        path,
		CWD:         cwd,
		Name:        name,
		CustomTitle: name, // pi's only title is the session_info name
		Provider:    provider,
		GitBranch:   gitBranch,
		Events:      events,
	}
}
