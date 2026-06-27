package ddev

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// envelope is the wrapper ddev puts around -j output.
type envelope struct {
	Raw json.RawMessage `json:"raw"`
}

// rawPayload extracts the "raw" field from ddev -j output. ddev may emit
// several JSON lines; the data line is the one with a non-null raw.
func rawPayload(stdout []byte) (json.RawMessage, error) {
	for _, line := range bytes.Split(stdout, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var e envelope
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		if len(e.Raw) > 0 && !bytes.Equal(e.Raw, []byte("null")) {
			return e.Raw, nil
		}
	}
	return nil, errors.New("no raw payload in ddev json output")
}

// ParseList parses `ddev list -j`.
func ParseList(stdout []byte) ([]Project, error) {
	raw, err := rawPayload(stdout)
	if err != nil {
		return nil, err
	}
	var projects []Project
	if err := json.Unmarshal(raw, &projects); err != nil {
		return nil, fmt.Errorf("decode ddev list: %w", err)
	}
	return projects, nil
}

// ParseDescribe parses `ddev describe <name> -j`.
func ParseDescribe(stdout []byte) (*Describe, error) {
	raw, err := rawPayload(stdout)
	if err != nil {
		return nil, err
	}
	var d Describe
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("decode ddev describe: %w", err)
	}
	return &d, nil
}
