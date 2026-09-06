package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/muesli/termenv"
)

// renderTable writes the session table to w (stdout in production).
// showDate prepends a DATE column. The unpriced-model footnote goes to errW
// (stderr in production). plain disables Unicode borders and color styling
// in favor of ASCII borders, for piping into cut/awk/grep. Ports render_table.
func renderTable(w, errW io.Writer, rows []Row, showDate, plain bool) {
	if len(rows) == 0 {
		fmt.Fprint(w, "No sessions found.\n")
		return
	}

	var headers []string
	if showDate {
		headers = append(headers, "DATE")
	}
	headers = append(headers, "PROJECT", "SESSION", "TIME", "DUR", "HARNESS", "MODEL", "TOKENS", "COST")

	body := make([][]string, 0, len(rows))
	for _, r := range rows {
		var cells []string
		if showDate {
			cells = append(cells, r.Date.Format("Mon 01-02"))
		}
		span := fmt.Sprintf("%s-%s", r.Start.Format("15:04"), r.End.Format("15:04"))
		cost := fmt.Sprintf("$%.2f", r.Cost)
		if !r.Priced {
			cost += "?"
		}
		name := r.Name
		if r.Active {
			name += " *"
		}
		cells = append(cells,
			r.Project,
			name,
			span,
			humanizeDuration(r.End.Sub(r.Start)),
			r.Harness,
			humanizeModels(r.Models),
			humanizeTokens(r.Tokens),
			cost,
		)
		body = append(body, cells)
	}

	var totalTokens int64
	var totalCost float64
	allPriced := true
	var totalTime time.Duration
	var unpricedModels []string
	seenUnpriced := map[string]bool{}
	for _, r := range rows {
		totalTokens += r.Tokens
		totalCost += r.Cost
		if !r.Priced {
			allPriced = false
			for _, m := range r.Unpriced {
				if !seenUnpriced[m] {
					seenUnpriced[m] = true
					unpricedModels = append(unpricedModels, m)
				}
			}
		}
		totalTime += r.End.Sub(r.Start)
	}

	label := fmt.Sprintf("%d sessions", len(rows))
	if showDate {
		label = fmt.Sprintf("%d session-days", len(rows))
	}

	total := make([]string, len(headers))
	total[0] = "TOTAL"
	total[indexOf(headers, "SESSION")] = label
	total[indexOf(headers, "DUR")] = humanizeDuration(totalTime)
	total[indexOf(headers, "TOKENS")] = humanizeTokens(totalTokens)
	totalCostStr := fmt.Sprintf("$%.2f", totalCost)
	if !allPriced {
		totalCostStr += "?"
	}
	total[indexOf(headers, "COST")] = totalCostStr

	totalRowIdx := len(body)

	// Use a renderer tied to the destination rather than Lipgloss's global
	// renderer. That keeps in-memory writers (and pipes) deterministic even
	// when the invoking terminal supports or forces color.
	renderer := lipgloss.NewRenderer(w)
	if plain || !isTerminalWriter(w) {
		renderer.SetColorProfile(termenv.Ascii)
	}
	border := lipgloss.RoundedBorder()
	borderStyle := renderer.NewStyle().Foreground(lipgloss.Color("62"))
	headerStyle := renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Padding(0, 1)
	cellStyle := renderer.NewStyle().Padding(0, 1)
	totalStyle := renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Padding(0, 1)
	if plain {
		border = lipgloss.ASCIIBorder()
		borderStyle = renderer.NewStyle()
		headerStyle = renderer.NewStyle().Padding(0, 1)
		totalStyle = renderer.NewStyle().Padding(0, 1)
	}

	t := table.New().
		Border(border).
		BorderStyle(borderStyle).
		Headers(headers...).
		Rows(body...).
		Row(total...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			if row == totalRowIdx {
				return totalStyle
			}
			return cellStyle
		})

	fmt.Fprintln(w, t.Render())

	if !allPriced {
		sort.Strings(unpricedModels)
		fmt.Fprintf(errW, "\n? = includes an unpriced model (%s); cost is a lower bound.\n", strings.Join(unpricedModels, ", "))
	}
}

// indexOf returns the index of s in headers.
func indexOf(headers []string, s string) int {
	for i, h := range headers {
		if h == s {
			return i
		}
	}
	return -1
}
