package diff

import (
	"reflect"
	"testing"
)

func TestCompute(t *testing.T) {
	cases := []struct {
		name            string
		before, after   any
		unknown         any
		beforeS, afterS any
		want            []AttributeDiff
	}{
		{
			name:   "scalar modified",
			before: map[string]any{"size": "small"},
			after:  map[string]any{"size": "large"},
			want: []AttributeDiff{
				{Path: "size", Before: "small", After: "large", ChangeType: Modified},
			},
		},
		{
			name:   "attribute added",
			before: map[string]any{},
			after:  map[string]any{"new_attr": "x"},
			want: []AttributeDiff{
				{Path: "new_attr", Before: nil, After: "x", ChangeType: Added},
			},
		},
		{
			name:   "attribute removed",
			before: map[string]any{"old_attr": "x"},
			after:  map[string]any{},
			want: []AttributeDiff{
				{Path: "old_attr", Before: "x", After: nil, ChangeType: Removed},
			},
		},
		{
			name:   "unchanged attribute omitted",
			before: map[string]any{"same": "x"},
			after:  map[string]any{"same": "x"},
			want:   nil,
		},
		{
			name: "nested map dot notation",
			before: map[string]any{
				"settings": map[string]any{"backup": map[string]any{"retention_days": float64(7)}},
			},
			after: map[string]any{
				"settings": map[string]any{"backup": map[string]any{"retention_days": float64(30)}},
			},
			want: []AttributeDiff{
				{Path: "settings.backup.retention_days", Before: float64(7), After: float64(30), ChangeType: Modified},
			},
		},
		{
			name: "list index notation",
			before: map[string]any{
				"ingress": []any{map[string]any{"cidr_blocks": []any{"10.0.0.0/16"}}},
			},
			after: map[string]any{
				"ingress": []any{map[string]any{"cidr_blocks": []any{"0.0.0.0/0"}}},
			},
			want: []AttributeDiff{
				{Path: "ingress[0].cidr_blocks[0]", Before: "10.0.0.0/16", After: "0.0.0.0/0", ChangeType: Modified},
			},
		},
		{
			name:   "map key dot notation",
			before: map[string]any{"tags": map[string]any{"Environment": "staging"}},
			after:  map[string]any{"tags": map[string]any{"Environment": "production"}},
			want: []AttributeDiff{
				{Path: "tags.Environment", Before: "staging", After: "production", ChangeType: Modified},
			},
		},
		{
			name:    "unknown value",
			before:  map[string]any{},
			after:   map[string]any{"id": nil},
			unknown: map[string]any{"id": true},
			want: []AttributeDiff{
				{Path: "id", Before: nil, After: nil, Unknown: true, ChangeType: Modified},
			},
		},
		{
			name:    "sensitive value redacted",
			before:  map[string]any{"password": "old-secret"},
			after:   map[string]any{"password": "new-secret"},
			beforeS: map[string]any{"password": true},
			afterS:  map[string]any{"password": true},
			want: []AttributeDiff{
				{Path: "password", Before: nil, After: nil, Sensitive: true, ChangeType: Modified},
			},
		},
		{
			name:   "null before and after produce no diff",
			before: nil,
			after:  nil,
			want:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Compute(tc.before, tc.after, tc.unknown, tc.beforeS, tc.afterS)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Compute() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestComputeNaturalSort(t *testing.T) {
	before := map[string]any{}
	after := map[string]any{
		"list": []any{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"},
	}
	got := Compute(before, after, nil, nil, nil)
	if len(got) != 11 {
		t.Fatalf("expected 11 diffs, got %d", len(got))
	}
	if got[9].Path != "list[9]" || got[10].Path != "list[10]" {
		t.Errorf("expected numeric natural ordering, got paths %q then %q", got[9].Path, got[10].Path)
	}
}
