package file

import (
	"reflect"
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

func TestSliceOps(t *testing.T) {
	a := []string{"a", "b", "c"}
	if contains(a, "b") < 0 {
		t.Error("should have found")
	}
	if contains(a, "d") >= 0 {
		t.Error("should not have found")
	}

	if changed, _ := assureIn(a, "b"); changed {
		t.Error("should not have changed")
	}

	if _, out := assureIn(a, "d"); !reflect.DeepEqual(out, []string{"a", "b", "c", "d"}) {
		t.Error("not equal")
	}

	if _, out := assureOut(a, "b"); !reflect.DeepEqual(out, []string{"a", "c"}) {
		t.Errorf("not equal")
	}

	if _, out := assureOut(a, "a"); !reflect.DeepEqual(out, []string{"b", "c"}) {
		t.Errorf("not equal")
	}
}
