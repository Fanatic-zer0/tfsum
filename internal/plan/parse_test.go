package plan

import (
	"testing"
)

func TestCheckFormatVersion(t *testing.T) {
	cases := []struct {
		version string
		wantErr bool
	}{
		{"1.2", false},
		{"1.0", false},
		{"2.0", true},
		{"", true},
	}
	for _, tc := range cases {
		err := checkFormatVersion(tc.version)
		if (err != nil) != tc.wantErr {
			t.Errorf("checkFormatVersion(%q) error = %v, wantErr %v", tc.version, err, tc.wantErr)
		}
	}
}

func TestParseRiskLevel(t *testing.T) {
	cases := []struct {
		in   string
		want RiskLevel
		ok   bool
	}{
		{"none", RiskNone, true},
		{"low", RiskLow, true},
		{"medium", RiskMedium, true},
		{"high", RiskHigh, true},
		{"critical", RiskCritical, true},
		{"bogus", RiskNone, false},
	}
	for _, tc := range cases {
		got, ok := ParseRiskLevel(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("ParseRiskLevel(%q) = (%v, %v), want (%v, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
