package render

import (
	"fmt"
	"io"

	"tfsum/internal/color"
)

// Error renders a helpful, color-coded error report. title is a short
// summary (e.g. "Terraform plan failed"), status describes the outcome
// (e.g. "Terraform exited with code 1."), detail is the raw
// Terraform/tfsum error text, and suggestions are actionable next steps.
// It never prints a raw Go stack trace.
func Error(w io.Writer, theme *color.Theme, title, status, detail string, suggestions []string) {
	fmt.Fprintf(w, "%s %s\n\n", theme.Error.Render(theme.Symbols.Cross), theme.Error.Render(title))
	if status != "" {
		fmt.Fprintf(w, "%s\n\n", status)
	}
	if detail != "" {
		fmt.Fprintln(w, "Error:")
		for _, line := range splitLines(detail) {
			fmt.Fprintf(w, "  %s\n", line)
		}
		fmt.Fprintln(w)
	}
	if len(suggestions) > 0 {
		fmt.Fprintln(w, "Suggested next steps:")
		for _, s := range suggestions {
			fmt.Fprintf(w, "  %s %s\n", theme.Symbols.Bullet, s)
		}
	}
}

// Warning renders a single color-coded warning line.
func Warning(w io.Writer, theme *color.Theme, msg string) {
	fmt.Fprintf(w, "%s %s\n", theme.Warning.Render("!"), theme.Warning.Render(msg))
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
