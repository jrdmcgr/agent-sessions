package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func piStubReader(path string) *Session {
	return &Session{Harness: "pi-stub"}
}

func claudeStubReader(path string) *Session {
	return &Session{Harness: "claude-stub"}
}

func buildDiscoverTree(t *testing.T) (piRoot, claudeRoot string) {
	t.Helper()
	tmp := t.TempDir()
	piRoot = filepath.Join(tmp, "pi")
	claudeRoot = filepath.Join(tmp, "claude")

	mustDir := func(p string) {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustFile := func(p string) {
		mustDir(filepath.Dir(p))
		if err := os.WriteFile(p, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustFile(filepath.Join(piRoot, "slugA", "a.jsonl"))
	mustFile(filepath.Join(piRoot, "slugA", "old.jsonl"))
	mustFile(filepath.Join(piRoot, "nested", "deep", "x.jsonl")) // too deep
	mustFile(filepath.Join(piRoot, "top.jsonl"))                 // too shallow
	mustFile(filepath.Join(claudeRoot, "slugB", "b.jsonl"))

	now := time.Now()
	old := now.Add(-3 * 24 * time.Hour)
	if err := os.Chtimes(filepath.Join(piRoot, "slugA", "a.jsonl"), now, now); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(piRoot, "slugA", "old.jsonl"), old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(claudeRoot, "slugB", "b.jsonl"), now, now); err != nil {
		t.Fatal(err)
	}

	return piRoot, claudeRoot
}

func TestDiscoverInZeroCutoff(t *testing.T) {
	piRoot, claudeRoot := buildDiscoverTree(t)

	got := discoverIn(piRoot, claudeRoot, piStubReader, claudeStubReader, time.Time{})

	wantPaths := []string{
		filepath.Join(piRoot, "slugA", "a.jsonl"),
		filepath.Join(piRoot, "slugA", "old.jsonl"),
		filepath.Join(claudeRoot, "slugB", "b.jsonl"),
	}
	if len(got) != len(wantPaths) {
		t.Fatalf("got %d files, want %d: %+v", len(got), len(wantPaths), got)
	}
	for i, w := range wantPaths {
		if got[i].path != w {
			t.Errorf("file %d: got path %q, want %q", i, got[i].path, w)
		}
	}

	// pi files carry the pi stub reader, b.jsonl carries the claude stub reader.
	for i, f := range got[:2] {
		if s := f.reader(""); s.Harness != "pi-stub" {
			t.Errorf("file %d (%s): reader = %q, want pi-stub", i, f.path, s.Harness)
		}
	}
	if s := got[2].reader(""); s.Harness != "claude-stub" {
		t.Errorf("claude file: reader = %q, want claude-stub", s.Harness)
	}
}

func TestDiscoverInCutoffExcludesOld(t *testing.T) {
	piRoot, claudeRoot := buildDiscoverTree(t)

	yesterday := time.Now().Add(-24 * time.Hour)
	cutoff := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.Local)

	got := discoverIn(piRoot, claudeRoot, piStubReader, claudeStubReader, cutoff)

	for _, f := range got {
		if filepath.Base(f.path) == "old.jsonl" {
			t.Fatalf("old.jsonl should have been excluded by cutoff, got %+v", got)
		}
	}
	if len(got) != 2 {
		t.Fatalf("got %d files, want 2 (a.jsonl, b.jsonl): %+v", len(got), got)
	}
}

func TestDiscoverInNonexistentRoots(t *testing.T) {
	tmp := t.TempDir()
	got := discoverIn(
		filepath.Join(tmp, "no-pi"),
		filepath.Join(tmp, "no-claude"),
		piStubReader, claudeStubReader,
		time.Time{},
	)
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %+v", got)
	}
}

func TestDecodeSlug(t *testing.T) {
	cases := map[string]string{
		"-Users-JaredMcGuire-Code-dotfiles": "/Users/JaredMcGuire/Code/dotfiles",
		"": "/",
	}
	for in, want := range cases {
		if got := decodeSlug(in); got != want {
			t.Errorf("decodeSlug(%q) = %q, want %q", in, got, want)
		}
	}
}
