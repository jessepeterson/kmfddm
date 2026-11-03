package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/alexedwards/flow"
	"github.com/micromdm/nanolib/log/stdlogfmt"
)

type enrollmentIDStub struct {
	ids  []string
	err  error
	sets []string
}

func (s *enrollmentIDStub) RetrieveEnrollmentIDs(_ context.Context, declarations []string, sets []string, ids []string) ([]string, error) {
	if declarations != nil || ids != nil {
		return nil, errors.New("unexpected declarations or ids")
	}
	s.sets = sets
	return s.ids, s.err
}

func TestGetSetEnrollmentsHandler(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		stub     *enrollmentIDStub
		wantCode int
		wantBody string
		wantSets []string
	}{
		{"ids", "/v1/set-enrollments/s1", &enrollmentIDStub{ids: []string{"A", "B"}}, http.StatusOK, "[\"A\",\"B\"]\n", []string{"s1"}},
		{"none", "/v1/set-enrollments/s1", &enrollmentIDStub{}, http.StatusOK, "[]\n", []string{"s1"}},
		{"error", "/v1/set-enrollments/s1", &enrollmentIDStub{err: errors.New("boom")}, http.StatusInternalServerError, "{\"error\":\"boom\"}\n", []string{"s1"}},
		{"empty set name", "/v1/set-enrollments/", &enrollmentIDStub{}, http.StatusNotFound, "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := flow.New()
			mux.Handle("/v1/set-enrollments/:id", GetSetEnrollmentsHandler(tt.stub, stdlogfmt.New()), "GET")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest("GET", tt.path, nil))
			if rec.Code != tt.wantCode {
				t.Fatalf("status: got %d, want %d", rec.Code, tt.wantCode)
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body: got %q, want %q", rec.Body.String(), tt.wantBody)
			}
			if !reflect.DeepEqual(tt.stub.sets, tt.wantSets) {
				t.Errorf("sets: got %v, want %v", tt.stub.sets, tt.wantSets)
			}
		})
	}
}
