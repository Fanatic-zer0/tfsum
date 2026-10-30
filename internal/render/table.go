package render

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"tfsum/internal/color"
	"tfsum/internal/plan"
)

// resourceTable renders a single module's changed resources as a
// color-coded, bordered table (symbol, address, action, risk), falling
// back to an ASCII border when the terminal doesn't advertise UTF-8.
// Columns are sized to their content rather than stretched to maxWidth.
func resourceTable(theme *color.Theme, resources []plan.ResourceChange) string {
	border := lipgloss.RoundedBorder()
	if !theme.Unicode {
		border = lipgloss.ASCIIBorder()
	}

	t := table.New().
		Border(border).
		BorderStyle(theme.Dim).
		BorderRow(false).
		Headers("", "RESOURCE", "ACTION", "RISK")

	t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == table.HeaderRow {
			return theme.Header.Padding(0, 1)
		}
		rc := resources[row]
		switch col {
		case 0, 2:
			return actionStyle(theme, rc.ActionKind).Padding(0, 1)
		case 1:
			return theme.Address.Padding(0, 1)
		case 3:
			level := rc.Risk.Level.String()
			return theme.StyleForRisk(level).Padding(0, 1)
		default:
			return theme.Dim.Padding(0, 1)
		}
	})

	for _, rc := range resources {
		t.Row(
			actionSymbol(theme.Symbols, rc.ActionKind),
			rc.Address,
			actionLabel(rc.ActionKind),
			rc.Risk.Level.String(),
		)
	}

	return t.Render()
}
