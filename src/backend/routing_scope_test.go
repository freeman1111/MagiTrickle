package magitrickle

import (
	"reflect"
	"testing"
)

func TestPickBypassMarks(t *testing.T) {
	policyMarks := map[string]uint32{
		"Policy0":     0xffffaaa,
		"Germany-AWG": 0xffffaaa,
		"Policy4":     0xffffaad,
		"noMT":        0xffffaad,
	}

	marks, missing := pickBypassMarks([]string{"noMT", "unknown", "Policy0"}, policyMarks)

	if expected := []uint32{0xffffaad, 0xffffaaa}; !reflect.DeepEqual(marks, expected) {
		t.Errorf("marks = %#x, want %#x", marks, expected)
	}
	if expected := []string{"unknown"}; !reflect.DeepEqual(missing, expected) {
		t.Errorf("missing = %v, want %v", missing, expected)
	}
}
