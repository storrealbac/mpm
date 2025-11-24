package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Table represents a styled table
type Table struct {
	headers []string
	rows    [][]string
	widths  []int
}

// NewTable creates a new table
func NewTable(headers ...string) *Table {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = lipgloss.Width(h)
	}
	return &Table{
		headers: headers,
		rows:    [][]string{},
		widths:  widths,
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells ...string) {
	if len(cells) != len(t.headers) {
		return // Skip invalid rows
	}

	// Update column widths
	for i, cell := range cells {
		w := lipgloss.Width(cell)
		if w > t.widths[i] {
			t.widths[i] = w
		}
	}

	t.rows = append(t.rows, cells)
}

// Render renders the table with a clean, borderless style
func (t *Table) Render() string {
	if len(t.headers) == 0 {
		return ""
	}

	var sb strings.Builder
	columnGap := 4

	// Headers
	for i, header := range t.headers {
		// Style the header
		styled := TableHeaderStyle.Render(header)
		sb.WriteString(styled)
		// Target width for this column (max content width)
		targetWidth := t.widths[i]

		// Padding needed
		padding := targetWidth - lipgloss.Width(header) + columnGap

		// Let's simplify: Just pad with spaces.
		if i < len(t.headers)-1 {
			sb.WriteString(strings.Repeat(" ", padding))
		}
	}
	sb.WriteString("\n\n")

	// Rows
	for _, row := range t.rows {
		for i, cell := range row {
			sb.WriteString(cell)

			if i < len(row)-1 {
				w := lipgloss.Width(cell)
				padding := t.widths[i] - w + columnGap
				// Adjust for header style padding difference if headers are wider?
				// The headers have extra padding from TableHeaderStyle.
				// Let's make it simple: The column width should be the max of (Header Width, Cell Widths).
				// But Header is styled differently.

				// Let's just use a simple alignment.
				if padding > 0 {
					sb.WriteString(strings.Repeat(" ", padding))
				}
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// CreateStatusBadge creates a professional status indicator
func CreateStatusBadge(status string) string {
	status = strings.ToUpper(status)
	switch status {
	case "INSTALLED", "OK", "SUCCESS":
		return SuccessBadge.Render(status)
	case "MISSING", "ERROR", "FAILED":
		return ErrorBadge.Render(status)
	case "OUTDATED", "WARNING", "PENDING":
		return WarningBadge.Render(status)
	default:
		return InfoBadge.Render(status)
	}
}

// CreateProgressBar creates a polished progress bar
func CreateProgressBar(current, total int, width int) string {
	if total == 0 {
		return ""
	}

	percent := float64(current) / float64(total)
	if percent > 1.0 {
		percent = 1.0
	}

	filled := int(percent * float64(width))
	empty := width - filled

	// Standard filled bar
	filledStyle := lipgloss.NewStyle().Foreground(primaryColor)
	emptyStyle := lipgloss.NewStyle().Foreground(grayColor)

	bar := filledStyle.Render(strings.Repeat("=", filled))
	bar += emptyStyle.Render(strings.Repeat("-", empty))

	// Clean percentage display
	percentText := fmt.Sprintf(" %d/%d (%.0f%%)", current, total, percent*100)
	percentStyle := lipgloss.NewStyle().Foreground(grayColor)

	return "[" + bar + "]" + percentStyle.Render(percentText)
}
