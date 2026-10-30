package color

import (
	"bytes"
	"os"
	"testing"
)

func TestParseMode(t *testing.T) {
	valid := []string{"auto", "always", "never"}
	for _, v := range valid {
		if _, err := ParseMode(v); err != nil {
			t.Errorf("ParseMode(%q) unexpected error: %v", v, err)
		}
	}
	if _, err := ParseMode("bogus"); err == nil {
		t.Error("expected error for invalid --color value")
	}
}

func TestNewColorNeverDisables(t *testing.T) {
	var buf bytes.Buffer
	theme := New(ModeNever, false, &buf)
	if theme.Enabled {
		t.Error("ModeNever should always disable color")
	}
}

func TestNewNoColorFlagDisables(t *testing.T) {
	var buf bytes.Buffer
	theme := New(ModeAuto, true, &buf)
	if theme.Enabled {
		t.Error("--no-color flag should disable color regardless of mode")
	}
}

func TestNewRespectsNoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	theme := New(ModeAuto, false, &buf)
	if theme.Enabled {
		t.Error("NO_COLOR env var should disable color in auto mode")
	}
}

func TestNewAlwaysForcesColorOnNonTTY(t *testing.T) {
	var buf bytes.Buffer
	theme := New(ModeAlways, false, &buf)
	if !theme.Enabled {
		t.Error("--color=always should force color even when output isn't a TTY")
	}
}

func TestNewAutoDisablesOnNonTTYBuffer(t *testing.T) {
	var buf bytes.Buffer
	theme := New(ModeAuto, false, &buf)
	if theme.Enabled {
		t.Error("auto mode should disable color when writing to a non-TTY buffer")
	}
}

func TestDetectUTF8RespectsLocale(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "en_US.UTF-8")
	if !detectUTF8() {
		t.Error("expected UTF-8 locale to be detected")
	}

	os.Setenv("LANG", "C")
	if detectUTF8() {
		t.Error("expected non-UTF-8 locale to be detected as ASCII")
	}
}

func TestSymbolsASCIIFallback(t *testing.T) {
	s := symbolsFor(false)
	if s.ModuleMarker != ">" || s.BoxVertical != "|" {
		t.Errorf("expected ASCII symbols, got %+v", s)
	}
}

func TestSymbolsUnicode(t *testing.T) {
	s := symbolsFor(true)
	if s.ModuleMarker == ">" || s.BoxVertical == "|" {
		t.Errorf("expected unicode symbols, got %+v", s)
	}
}
