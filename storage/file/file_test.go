package file

import (
	"context"
	"hash"
	"reflect"
	"testing"

	"github.com/cespare/xxhash"
	"github.com/jessepeterson/kmfddm/storage/test"
	"github.com/jessepeterson/kmfddm/test/e2e"
)

func TestFile(t *testing.T) {
	s, err := New(t.TempDir(), func() hash.Hash { return xxhash.New() })
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	test.TestBasic(t, s, ctx)
	test.TestBasicStatus(t, "../test", s, ctx)
	t.Run("TestE2E", func(t *testing.T) {
		e2e.TestE2E(t, ctx, s)
	})

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
