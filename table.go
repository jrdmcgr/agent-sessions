package main

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	tableBorderColor = lipgloss.Color("62")
	tableHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Padding(0, 1)
	tableCellStyle   = lipgloss.NewStyle().Padding(0, 1)
	tableTotalStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Padding(0, 1)
)

// renderTable writes the session table to w (stdout in production).
// showDate prepends a DATE column. The unpriced-model footnote goes to errW
// (stderr in production). Ports render_table.
func renderTable(w, errW io.Writer, rows []Row, showDate bool) {
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
	for _, r := range rows {
		totalTokens += r.Tokens
		totalCost += r.Cost
		if !r.Priced {
			allPriced = false
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

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(tableBorderColor)).
		Headers(headers...).
		Rows(body...).
		Row(total...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return tableHeaderStyle
			}
			if row == totalRowIdx {
				return tableTotalStyle
			}
			return tableCellStyle
		})

	fmt.Fprintln(w, t.Render())

	if !allPriced {
		fmt.Fprint(errW, "\n? = includes an unpriced model; cost is a lower bound.\n")
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
