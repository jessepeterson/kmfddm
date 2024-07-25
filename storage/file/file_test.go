package file

import (
	"context"
	"hash"
	"reflect"
	"testing"

	"github.com/cespare/xxhash"
	"github.com/jessepeterson/kmfddm/storage/test"
)

func TestFile(t *testing.T) {
	s, err := New(t.TempDir(), func() hash.Hash { return xxhash.New() })
	if err != nil {
		t.Fatal(err)
	}

	test.TestBasic(t, s, context.Background())
	test.TestBasicStatus(t, "../test", s, context.Background())
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
