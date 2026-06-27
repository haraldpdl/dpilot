package ddev

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
