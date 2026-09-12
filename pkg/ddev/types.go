package ddev

import "strings"

// ProjectStatus mirrors ddev's project status strings.
type ProjectStatus string

const (
	StatusRunning ProjectStatus = "running"
	StatusStopped ProjectStatus = "stopped"
	StatusPaused  ProjectStatus = "paused"
	// StatusMissing is synthesized by dpilot when a member is absent from ddev.
	StatusMissing ProjectStatus = "missing"
)

// Project is one entry from `ddev list -j`.
type Project struct {
	Name    string        `json:"name"`
	Status  ProjectStatus `json:"status"`
	Type    string        `json:"type"`
	AppRoot string        `json:"approot"`
}

// Describe is the payload of `ddev describe <name> -j`. ddev's project-level
// status already reflects container health, so dpilot parses no per-service
// fields (an optional service such as xhgui is legitimately stopped in a
// healthy project).
type Describe struct {
	Name   string        `json:"name"`
	Status ProjectStatus `json:"status"`
}

// Ready reports whether the project is running.
func (d *Describe) Ready() bool {
	return d.Status == StatusRunning
}

// Tone is how a status should be coloured; the CLI and TUI map it to their
// own colour systems so both follow ddev's rules from one place.
type Tone int

const (
	ToneGood Tone = iota // green
	ToneWarn             // yellow
	ToneBad              // red
)

// Tone mirrors ddev's FormatSiteStatus: paused is yellow; stopped, missing
// and unhealthy are red; anything else (running included) is green.
func (s ProjectStatus) Tone() Tone {
	switch {
	case strings.Contains(string(s), string(StatusPaused)):
		return ToneWarn
	case s == StatusStopped, s == StatusMissing, s == "unhealthy", s == "exited":
		return ToneBad
	default:
		return ToneGood
	}
}

// Label is the status word to display: ddev shows a running project as "OK".
func (s ProjectStatus) Label() string {
	if s == StatusRunning {
		return "OK"
	}
	return string(s)
}
