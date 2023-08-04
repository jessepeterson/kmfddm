package storage

import "time"

type StatusError struct {
	Path      string      `json:"path"`
	Error     interface{} `json:"error"`
	Timestamp time.Time   `json:"timestamp"`
	StatusID  string      `json:"status_id,omitempty"`
}

type StatusValue struct {
	Path      string    `json:"path"`
	Value     string    `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	StatusID  string    `json:"status_id,omitempty"`
}
