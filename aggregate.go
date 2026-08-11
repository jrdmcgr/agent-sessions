package main

import (
	"path/filepath"
	"strings"
	"time"
)

// dayBucket accumulates one calendar day's worth of a session's events.
type dayBucket struct {
	first, last time.Time
	usage       Usage
	cost        float64
	priced      bool
	models      []string // raw models, order of first appearance
	seenModel   map[string]bool
	messages    int
}

// sessionDays splits a session into one Row per calendar day of activity.
// days is the set of selected days (keys are dayOf values); nil means all
// days. now is used for the Active flag. Rows are returned sorted by date
// ascending (Python relies on dict ordering; we sort explicitly — the final
// global sort in main makes this equivalent).
func sessionDays(s *Session, days map[time.Time]bool, now time.Time) []Row {
	order := []time.Time{}
	buckets := map[time.Time]*dayBucket{}

	for _, event := range s.Events {
		if event.TS.IsZero() {
			continue
		}
		day := dayOf(event.TS)
		if days != nil && !days[day] {
			continue
		}
		b, ok := buckets[day]
		if !ok {
			b = &dayBucket{priced: true, seenModel: map[string]bool{}}
			buckets[day] = b
			order = append(order, day)
		}
		if b.first.IsZero() || event.TS.Before(b.first) {
			b.first = event.TS
		}
		if b.last.IsZero() || event.TS.After(b.last) {
			b.last = event.TS
		}
		b.messages++
		b.usage.Add(event.Usage)
		if event.Model != "" && !b.seenModel[event.Model] {
			b.seenModel[event.Model] = true
			b.models = append(b.models, event.Model)
		}
		if event.Usage.Any() {
			cost := event.Cost
			if cost == nil {
				cost = price(event.Model, event.Usage)
			}
			if cost == nil {
				b.priced = false
			} else {
				b.cost += *cost
			}
		}
	}

	name := s.Name
	if name == "" {
		name = fallbackName(s)
	}
	project := filepath.Base(s.CWD)
	if project == "" || project == "/" || project == "." {
		project = s.CWD
	}

	rows := make([]Row, 0, len(order))
	for _, day := range order {
		b := buckets[day]
		models := make([]string, len(b.models))
		for i, m := range b.models {
			models[i] = shortModel(m)
		}
		rows = append(rows, Row{
			Date:     day,
			Harness:  s.Harness,
			ID:       s.ID,
			Name:     name,
			Project:  project,
			CWD:      s.CWD,
			Start:    b.first,
			End:      b.last,
			Models:   models,
			Tokens:   b.usage.Total(),
			Usage:    b.usage,
			Cost:     b.cost,
			Priced:   b.priced,
			Messages: b.messages,
			Active:   now.Sub(b.last) < ActiveWindow,
			Path:     s.Path,
		})
	}
	return rows
}

// fallbackName returns the first meaningful user prompt as a name: skip
// non-user events, trim whitespace, skip empty text and text starting with
// any NoisePrefixes entry, collapse all whitespace runs to single spaces
// (strings.Fields + strings.Join), truncate to 48 bytes. If nothing
// qualifies, return "(unnamed)". Ports fallback_name.
func fallbackName(s *Session) string {
	for _, event := range s.Events {
		if event.Role != "user" {
			continue
		}
		text := strings.TrimSpace(event.Text)
		if text == "" {
			continue
		}
		noisy := false
		for _, prefix := range NoisePrefixes {
			if strings.HasPrefix(text, prefix) {
				noisy = true
				break
			}
		}
		if noisy {
			continue
		}
		collapsed := strings.Join(strings.Fields(text), " ")
		if len(collapsed) > 48 {
			collapsed = collapsed[:48]
		}
		return collapsed
	}
	return "(unnamed)"
}
