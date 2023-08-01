package ddm

import (
	"testing"
)

func TestPathSplit(t *testing.T) {
	for _, v := range []struct {
		path         string
		expectedType string
		expectedDecl string
		expectedErr  bool
	}{
		{
			path:         "foo/bar",
			expectedType: "foo",
			expectedDecl: "bar",
			expectedErr:  false,
		},
		{
			path:         "foobar",
			expectedType: "foo",
			expectedDecl: "bar",
			expectedErr:  true,
		},
		{
			path:         "foo/bar/baz",
			expectedType: "foo",
			expectedDecl: "bar/baz",
			expectedErr:  false,
		},
	} {
		t.Run("parse-"+v.path, func(t *testing.T) {
			rType, rDecl, err := parseDeclarationPath(v.path)
			if err != nil && !v.expectedErr {
				t.Errorf("expected no error, but go one: %v", err)
			}
			if err == nil && v.expectedErr {
				t.Error("expected error, but did not get one")
			}
			if err == nil {
				if have, want := rDecl, v.expectedDecl; have != want {
					t.Errorf("have: %v, want: %v", have, want)
				}
				if have, want := rType, v.expectedType; have != want {
					t.Errorf("have: %v, want: %v", have, want)
				}
			}
		})
	}
}
