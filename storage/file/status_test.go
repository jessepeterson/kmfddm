package file

import (
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
)

func TestMergeStatusValues(t *testing.T) {
	a := []ddm.StatusValue{
		{
			Path: ".a",
		},
		{
			Path: ".d",
		},
	}
	b := []ddm.StatusValue{
		{
			Path: ".b",
		},
		{
			Path: ".c",
		},
		{
			Path: ".d",
		},
	}
	if containsValue(b, a[1]) < 0 {
		t.Error("should contain value")
	}
	c := mergeStatusValues(a, b)
	if have, want := len(c), 4; have != want {
		t.Errorf("have: %v, want: %v, %q, %q", have, want, a, b)
	}
	// fmt.Println(c)
}
