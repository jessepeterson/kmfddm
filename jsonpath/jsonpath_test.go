package jsonpath

import (
	"reflect"
	"testing"

	"github.com/valyala/fastjson"
)

const test1 = `{"foo": {"bar": {"baz": true, "qux": false}}}`

func TestUnhandled(t *testing.T) {
	v, err := fastjson.ParseBytes([]byte(test1))
	if err != nil {
		t.Fatal(err)
	}
	mux := NewPathMux()
	mux.HandleFunc(".foo.bar.baz", func(string, *fastjson.Value) ([]string, error) { return nil, nil })
	unhandled, err := mux.JSONPath("", v)
	if err != nil {
		t.Fatal(err)
	}
	if have, want := unhandled, []string{".foo.bar.qux"}; !reflect.DeepEqual(have, want) {
		t.Errorf("have: %q, want %q", have, want)
	}
}

// valCaptureHandler just collects the objects it has handled
type valCaptureHandler struct {
	vals []struct {
		path string
		v    *fastjson.Value
	}
}

func (h *valCaptureHandler) JSONPath(path string, v *fastjson.Value) ([]string, error) {
	h.vals = append(h.vals, struct {
		path string
		v    *fastjson.Value
	}{path: path, v: v})
	return nil, nil
}

// collapse reduces the collected objects to a map and returns it.
// potentially overwriting multiple handlers at the same path.
func (h *valCaptureHandler) collapse() map[string]*fastjson.Value {
	ret := make(map[string]*fastjson.Value)
	for _, v := range h.vals {
		ret[v.path] = v.v
	}
	return ret
}

func TestBasicWildcard(t *testing.T) {
	v, err := fastjson.ParseBytes([]byte(test1))
	if err != nil {
		t.Fatal(err)
	}
	mux := NewPathMux()
	cap := new(valCaptureHandler)
	mux.Handle(".foo.bar.", cap)
	_, err = mux.JSONPath("", v)
	if err != nil {
		t.Fatal(err)
	}
	col := cap.collapse()
	if have, want := col[".foo.bar.baz"].Type(), fastjson.TypeTrue; have != want {
		t.Errorf("have: %q, want %q", have, want)
	}
	if have, want := col[".foo.bar.qux"].Type(), fastjson.TypeFalse; have != want {
		t.Errorf("have: %q, want %q", have, want)
	}
}

func TestWildcardWithNormal(t *testing.T) {
	v, err := fastjson.ParseBytes([]byte(test1))
	if err != nil {
		t.Fatal(err)
	}
	mux := NewPathMux()
	cap := new(valCaptureHandler)
	cap2 := new(valCaptureHandler)
	mux.Handle(".foo.bar.", cap)
	mux.Handle(".foo.bar.baz", cap2)
	_, err = mux.JSONPath("", v)
	if err != nil {
		t.Fatal(err)
	}
	if have, want := cap2.collapse()[".foo.bar.baz"].Type(), fastjson.TypeTrue; have != want {
		t.Errorf("have: %q, want %q", have, want)
	}
	if have, want := cap.collapse()[".foo.bar.qux"].Type(), fastjson.TypeFalse; have != want {
		t.Errorf("have: %q, want %q", have, want)
	}
}

// TestEmbedded tests two PathMuxes embedded one-inside-the-other.
func TestEmbedded(t *testing.T) {
	v, err := fastjson.ParseBytes([]byte(test1))
	if err != nil {
		t.Fatal(err)
	}
	muxOuter := NewPathMux()
	muxInner := NewPathMux()
	cap := new(valCaptureHandler)
	muxOuter.Handle(".foo", StripAndAddPrefix(".foo", muxInner))
	muxInner.Handle(".bar.", AddPrefix(".foo", cap))
	unhandled, err := muxOuter.JSONPath("", v)
	if err != nil {
		t.Fatal(err)
	}
	if len(unhandled) > 0 {
		t.Errorf("unhandled should be empty: %q", unhandled)
	}
	col := cap.collapse()
	if have, want := col[".foo.bar.baz"].Type(), fastjson.TypeTrue; have != want {
		t.Errorf("have: %q, want %q", have, want)
	}
	if have, want := col[".foo.bar.qux"].Type(), fastjson.TypeFalse; have != want {
		t.Errorf("have: %q, want %q", have, want)
	}
}

func TestHandle(t *testing.T) {
	m := NewPathMux()

	m.HandleFunc(".foo.bar.baz", func(string, *fastjson.Value) ([]string, error) { return nil, nil })

	// verify we've properly setup our pathHandlers and have the handler attached
	pH := m.pathHandlers[""].pathHandlers["foo"].pathHandlers["bar"].pathHandlers["baz"]

	if have, want := len(pH.handlers), 1; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}
}
