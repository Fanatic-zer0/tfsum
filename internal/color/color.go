// Package color builds tfsum's terminal theme: a lipgloss-based style set
// and a Unicode/ASCII symbol set, selected according to color mode,
// NO_COLOR, TTY detection, and UTF-8 locale support.
package color

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Mode is the --color flag value.
type Mode string

const (
	ModeAuto   Mode = "auto"
	ModeAlways Mode = "always"
	ModeNever  Mode = "never"
)

// ParseMode validates a --color flag value.
func ParseMode(s string) (Mode, error) {
	switch Mode(s) {
	case ModeAuto, ModeAlways, ModeNever:
		return Mode(s), nil
	default:
		return "", fmt.Errorf("invalid --color value %q (expected auto, always, or never)", s)
	}
}

// Theme holds every style and symbol tfsum's renderers use, pre-resolved
// for the active color profile and UTF-8 support.
type Theme struct {
	Enabled bool
	Unicode bool
	Symbols Symbols

	Create    lipgloss.Style
	Update    lipgloss.Style
	Replace   lipgloss.Style
	Destroy   lipgloss.Style
	Drift     lipgloss.Style
	Unknown   lipgloss.Style
	Sensitive lipgloss.Style

	RiskLow      lipgloss.Style
	RiskMedium   lipgloss.Style
	RiskHigh     lipgloss.Style
	RiskCritical lipgloss.Style

	Header   lipgloss.Style
	Address  lipgloss.Style
	AttrPath lipgloss.Style
	OldValue lipgloss.Style
	NewValue lipgloss.Style
	Dim      lipgloss.Style
	Warning  lipgloss.Style
	Error    lipgloss.Style
}

// New builds a Theme for output writer w, honoring mode, the --no-color
// flag, and the NO_COLOR environment variable.
func New(mode Mode, noColorFlag bool, w io.Writer) *Theme {
	r := lipgloss.NewRenderer(w)

	disabled := noColorFlag || mode == ModeNever ||
		(mode != ModeAlways && os.Getenv("NO_COLOR") != "")

	switch {
	case disabled:
		r.SetColorProfile(termenv.Ascii)
	case mode == ModeAlways:
		// Force color even when stdout isn't a TTY (e.g. CI log capture).
		if r.ColorProfile() == termenv.Ascii {
			r.SetColorProfile(termenv.TrueColor)
		}
	default:
		// auto: keep termenv's own TTY + TERM/COLORTERM detection, which
		// already degrades truecolor -> 256 -> ANSI -> ascii.
	}

	enabled := r.ColorProfile() != termenv.Ascii
	return buildTheme(r, enabled, detectUTF8())
}

func buildTheme(r *lipgloss.Renderer, enabled, unicode bool) *Theme {
	dimColor := lipgloss.AdaptiveColor{Light: "250", Dark: "240"}
	headerColor := lipgloss.AdaptiveColor{Light: "25", Dark: "87"}

	return &Theme{
		Enabled: enabled,
		Unicode: unicode,
		Symbols: symbolsFor(unicode),

		Create:    r.NewStyle().Foreground(lipgloss.Color("2")),
		Update:    r.NewStyle().Foreground(lipgloss.Color("3")),
		Replace:   r.NewStyle().Foreground(lipgloss.Color("213")),
		Destroy:   r.NewStyle().Foreground(lipgloss.Color("9")),
		Drift:     r.NewStyle().Foreground(lipgloss.Color("6")),
		Unknown:   r.NewStyle().Foreground(lipgloss.Color("4")),
		Sensitive: r.NewStyle().Foreground(lipgloss.Color("5")),

		RiskLow:      r.NewStyle().Foreground(lipgloss.Color("2")),
		RiskMedium:   r.NewStyle().Foreground(lipgloss.Color("3")),
		RiskHigh:     r.NewStyle().Foreground(lipgloss.Color("9")),
		RiskCritical: r.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("1")),

		Header:   r.NewStyle().Bold(true).Foreground(headerColor),
		Address:  r.NewStyle().Foreground(lipgloss.Color("6")),
		AttrPath: r.NewStyle().Foreground(lipgloss.Color("33")),
		OldValue: r.NewStyle().Foreground(lipgloss.Color("1")),
		NewValue: r.NewStyle().Foreground(lipgloss.Color("2")),
		Dim:      r.NewStyle().Foreground(dimColor),
		Warning:  r.NewStyle().Foreground(lipgloss.Color("3")),
		Error:    r.NewStyle().Foreground(lipgloss.Color("1")),
	}
}

// StyleForRisk returns the style for a risk level by its uppercase label.
func (t *Theme) StyleForRisk(level string) lipgloss.Style {
	switch level {
	case "CRITICAL":
		return t.RiskCritical
	case "HIGH":
		return t.RiskHigh
	case "MEDIUM":
		return t.RiskMedium
	case "LOW":
		return t.RiskLow
	default:
		return t.Dim
	}
}

func detectUTF8() bool {
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		v := os.Getenv(key)
		if v == "" {
			continue
		}
		lower := strings.ToLower(v)
		return strings.Contains(lower, "utf-8") || strings.Contains(lower, "utf8")
	}
	// No locale information available; assume UTF-8 (true on macOS/modern
	// Linux terminals by default).
	return true
}
