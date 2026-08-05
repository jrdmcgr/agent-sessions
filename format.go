package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// humanizeTokens: >=1e9 -> "1.2B" (one decimal); >=1e6 -> "3.4M" (one
// decimal); >=1e3 -> "56k" (no decimal, rounded like Python's %.0f i.e.
// banker's-free: use strconv.FormatFloat(f, 'f', 0, 64)); else the plain
// integer. Ports humanize_tokens.
func humanizeTokens(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return strconv.FormatFloat(float64(n)/1_000, 'f', 0, 64) + "k"
	default:
		return strconv.FormatInt(n, 10)
	}
}

// humanizeDuration: total whole minutes (floor). <60 -> "42m"; else
// "3h07m" (minutes zero-padded to 2). Ports humanize_duration.
func humanizeDuration(d time.Duration) string {
	minutes := int64(d / time.Minute)
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%dh%02dm", minutes/60, minutes%60)
}

// humanizeModels: empty -> "-". Join the first two with ","; if that string
// exceeds 22 bytes, cut to 21 bytes and append "~". If more than two models,
// append "+N" where N = len-2 (after any truncation of the head). Ports
// humanize_models.
func humanizeModels(models []string) string {
	if len(models) == 0 {
		return "-"
	}
	head := strings.Join(models[:min(2, len(models))], ",")
	if len(head) > 22 {
		head = head[:21] + "~"
	}
	if len(models) <= 2 {
		return head
	}
	return fmt.Sprintf("%s+%d", head, len(models)-2)
}
