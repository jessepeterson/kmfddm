package storage

import "time"

type StatusError struct {
	Path      string      `json:"path"`
	Error     interface{} `json:"error"`
	Timestamp time.Time   `json:"timestamp"`
}

type StatusValue struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}
