package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// decodeSlug converts "-Users-JaredMcGuire-Code-dotfiles" to
// "/Users/JaredMcGuire/Code/dotfiles" — strip leading/trailing '-', replace
// '-' with '/', prepend "/". Lossy for paths containing real dashes; that is
// accepted. Ports decode_slug.
func decodeSlug(slug string) string {
	return "/" + strings.ReplaceAll(strings.Trim(slug, "-"), "-", "/")
}

// sessionReader parses one transcript file into a Session (or nil).
type sessionReader func(path string) *Session

// discoveredFile pairs a transcript path with the reader that understands it.
type discoveredFile struct {
	path   string
	reader sessionReader
}

// discoverIn scans piRoot and claudeRoot for "*/*.jsonl" files (exactly one
// directory deep, sorted lexically by full path within each root, pi root's
// files before claude root's) whose mtime is >= cutoff. A zero cutoff means
// no filtering. Roots that don't exist are skipped; files that can't be
// stat'ed are skipped. Every file under piRoot is paired with piReader in
// the result; every file under claudeRoot is paired with claudeReader.
func discoverIn(piRoot, claudeRoot string, piReader, claudeReader sessionReader, cutoff time.Time) []discoveredFile {
	var out []discoveredFile
	roots := []struct {
		dir    string
		reader sessionReader
	}{
		{piRoot, piReader},
		{claudeRoot, claudeReader},
	}
	for _, r := range roots {
		matches, err := filepath.Glob(filepath.Join(r.dir, "*", "*.jsonl"))
		if err != nil {
			continue
		}
		for _, path := range matches {
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			if !cutoff.IsZero() && info.ModTime().Before(cutoff) {
				continue
			}
			out = append(out, discoveredFile{path: path, reader: r.reader})
		}
	}
	return out
}
