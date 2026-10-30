// Package diff implements a recursive semantic diff engine over Terraform's
// JSON plan representation of before/after resource attribute values.
package diff

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// ChangeType classifies how a single attribute path changed.
type ChangeType string

const (
	Added    ChangeType = "added"
	Removed  ChangeType = "removed"
	Modified ChangeType = "modified"
)

// AttributeDiff describes a single changed attribute path between a
// resource's before and after state.
type AttributeDiff struct {
	Path       string
	Before     any
	After      any
	Unknown    bool
	Sensitive  bool
	ChangeType ChangeType
}

// Compute walks the before/after/unknown/sensitive trees produced by
// Terraform's JSON plan and returns a naturally-sorted list of changed
// attribute paths. Unchanged leaves are omitted.
func Compute(before, after, afterUnknown, beforeSensitive, afterSensitive any) []AttributeDiff {
	var out []AttributeDiff
	walk("", before, after, afterUnknown, beforeSensitive, afterSensitive, &out)
	sort.Slice(out, func(i, j int) bool { return lessPath(out[i].Path, out[j].Path) })
	return out
}

func walk(path string, before, after, unknown, beforeSens, afterSens any, out *[]AttributeDiff) {
	sensitive := isSensitive(beforeSens) || isSensitive(afterSens)

	// The whole value at this path is unknown (known only after apply).
	if b, ok := unknown.(bool); ok && b {
		*out = append(*out, AttributeDiff{
			Path:       path,
			Before:     redact(before, sensitive),
			After:      nil,
			Unknown:    true,
			Sensitive:  sensitive,
			ChangeType: Modified,
		})
		return
	}

	if afterMap, ok := after.(map[string]any); ok {
		beforeMap, _ := before.(map[string]any)
		unknownMap, _ := unknown.(map[string]any)
		beforeSensMap, _ := beforeSens.(map[string]any)
		afterSensMap, _ := afterSens.(map[string]any)
		for _, k := range unionKeys(beforeMap, afterMap) {
			walk(joinPath(path, k), mapGet(beforeMap, k), mapGet(afterMap, k),
				mapGet(unknownMap, k), mapGet(beforeSensMap, k), mapGet(afterSensMap, k), out)
		}
		return
	}

	if afterSlice, ok := after.([]any); ok {
		beforeSlice, _ := before.([]any)
		unknownSlice, _ := unknown.([]any)
		beforeSensSlice, _ := beforeSens.([]any)
		afterSensSlice, _ := afterSens.([]any)
		n := len(afterSlice)
		if len(beforeSlice) > n {
			n = len(beforeSlice)
		}
		for i := 0; i < n; i++ {
			walk(fmt.Sprintf("%s[%d]", path, i), sliceGet(beforeSlice, i), sliceGet(afterSlice, i),
				sliceGet(unknownSlice, i), sliceGet(beforeSensSlice, i), sliceGet(afterSensSlice, i), out)
		}
		return
	}

	// after is absent (nil/removed) but before may still be a container.
	if after == nil {
		if beforeMap, ok := before.(map[string]any); ok {
			unknownMap, _ := unknown.(map[string]any)
			beforeSensMap, _ := beforeSens.(map[string]any)
			afterSensMap, _ := afterSens.(map[string]any)
			for _, k := range unionKeys(beforeMap, nil) {
				walk(joinPath(path, k), mapGet(beforeMap, k), nil,
					mapGet(unknownMap, k), mapGet(beforeSensMap, k), mapGet(afterSensMap, k), out)
			}
			return
		}
		if beforeSlice, ok := before.([]any); ok {
			unknownSlice, _ := unknown.([]any)
			beforeSensSlice, _ := beforeSens.([]any)
			afterSensSlice, _ := afterSens.([]any)
			for i := range beforeSlice {
				walk(fmt.Sprintf("%s[%d]", path, i), sliceGet(beforeSlice, i), nil,
					sliceGet(unknownSlice, i), sliceGet(beforeSensSlice, i), sliceGet(afterSensSlice, i), out)
			}
			return
		}
	}

	if equalValues(before, after) {
		return
	}

	changeType := Modified
	switch {
	case before == nil && after != nil:
		changeType = Added
	case before != nil && after == nil:
		changeType = Removed
	}

	*out = append(*out, AttributeDiff{
		Path:       path,
		Before:     redact(before, sensitive),
		After:      redact(after, sensitive),
		Sensitive:  sensitive,
		ChangeType: changeType,
	})
}

func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

func unionKeys(a, b map[string]any) []string {
	seen := make(map[string]bool, len(a)+len(b))
	keys := make([]string, 0, len(a)+len(b))
	for k := range a {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for k := range b {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

func mapGet(m map[string]any, k string) any {
	if m == nil {
		return nil
	}
	return m[k]
}

func sliceGet(s []any, i int) any {
	if i < 0 || i >= len(s) {
		return nil
	}
	return s[i]
}

func isSensitive(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

func redact(value any, sensitive bool) any {
	if sensitive {
		return nil
	}
	return value
}

func equalValues(a, b any) bool {
	return reflect.DeepEqual(a, b)
}

// lessPath performs a natural sort comparison so that numeric path segments
// (list/set indexes) sort numerically rather than lexically, e.g.
// "ingress[2]" before "ingress[10]".
func lessPath(a, b string) bool {
	ta, tb := tokenizePath(a), tokenizePath(b)
	for i := 0; i < len(ta) && i < len(tb); i++ {
		if ta[i] == tb[i] {
			continue
		}
		na, aErr := strconv.Atoi(ta[i])
		nb, bErr := strconv.Atoi(tb[i])
		if aErr == nil && bErr == nil {
			return na < nb
		}
		return ta[i] < tb[i]
	}
	return len(ta) < len(tb)
}

func tokenizePath(s string) []string {
	s = strings.NewReplacer("[", ".", "]", "").Replace(s)
	return strings.Split(s, ".")
}
