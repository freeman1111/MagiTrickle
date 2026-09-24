package magitrickle

import (
	"reflect"
	"testing"

	"magitrickle/internal/interfaces"
)

func TestPickBypassMarks(t *testing.T) {
	policies := []interfaces.Policy{
		{ID: "Policy0", Description: "Germany-AWG", Mark: 0xffffaaa},
		{ID: "Policy4", Description: "noMT", Mark: 0xffffaad},
	}

	marks, missing := pickBypassMarks([]string{"NOMT", "unknown", "Policy0"}, policies)

	if expected := []uint32{0xffffaad, 0xffffaaa}; !reflect.DeepEqual(marks, expected) {
		t.Errorf("marks = %#x, want %#x", marks, expected)
	}
	if expected := []string{"unknown"}; !reflect.DeepEqual(missing, expected) {
		t.Errorf("missing = %v, want %v", missing, expected)
	}
}
