package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultWaitTimeout is the per-member readiness budget when unset.
const DefaultWaitTimeout = 120 * time.Second

// Duration wraps time.Duration so YAML uses "90s" style strings.
type Duration time.Duration

func (d Duration) Duration() time.Duration { return time.Duration(d) }

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid wait_timeout %q: %w", s, err)
	}
	if parsed <= 0 {
		return fmt.Errorf("invalid wait_timeout %q: must be positive", s)
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) MarshalYAML() (any, error) { return time.Duration(d).String(), nil }

// Group is a named, ordered set of ddev project names.
type Group struct {
	Name        string   `yaml:"name"`
	WaitTimeout Duration `yaml:"wait_timeout,omitempty"`
	Members     []string `yaml:"members"`
}

// Dir returns the groups directory, honoring $DPILOT_HOME then $HOME.
func Dir() (string, error) {
	base := os.Getenv("DPILOT_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".dpilot")
	}
	return filepath.Join(base, "groups"), nil
}

var (
	// nameRE is the rule for new group names: a filename-safe token that can
	// never be taken for a flag, a dotfile, or a path component.
	nameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	// memberRE mirrors ddev's own project-name rule (RFC 1123 hostname), so a
	// member can never be parsed by ddev as a flag.
	memberRE = regexp.MustCompile(`^([A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?\.)*[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?$`)
)

// ValidateName is the rule for creating a group. Groups created by earlier
// releases may have looser names; those files stay usable (see pathSafe).
func ValidateName(name string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf("invalid group name %q: use letters, digits, '.', '_' or '-', starting with a letter or digit", name)
	}
	return nil
}

// pathSafe is the rule for addressing an existing group file: it only has to
// stay inside the groups directory.
func pathSafe(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("invalid group name %q", name)
	}
	return nil
}

func path(name string) (string, error) {
	if err := pathSafe(name); err != nil {
		return "", err
	}
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, name+".yaml"), nil
}

// Validate checks invariants.
func (g *Group) Validate() error {
	if err := pathSafe(g.Name); err != nil {
		return err
	}
	// 0 means "unset, use the default" (it is what omitempty writes back);
	// an explicit non-positive value is rejected when the YAML is decoded.
	if g.WaitTimeout < 0 {
		return errors.New("wait_timeout must not be negative")
	}
	seen := map[string]bool{}
	for _, m := range g.Members {
		if !memberRE.MatchString(m) {
			return fmt.Errorf("invalid member name %q: must be a ddev project name", m)
		}
		if seen[m] {
			return fmt.Errorf("duplicate member %q", m)
		}
		seen[m] = true
	}
	return nil
}

// Load reads, applies the default timeout, and validates a group.
func Load(name string) (*Group, error) {
	p, err := path(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("group %q does not exist", name)
		}
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var g Group
	if err := dec.Decode(&g); err != nil {
		return nil, fmt.Errorf("parse group %q: %w", name, err)
	}
	// The filename is the group's identity; a stale name: inside a copied or
	// renamed file must not retarget Save.
	g.Name = name
	if g.WaitTimeout == 0 {
		g.WaitTimeout = Duration(DefaultWaitTimeout)
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return &g, nil
}

// Save writes a group to disk, creating the directory.
func Save(g *Group) error {
	if err := g.Validate(); err != nil {
		return err
	}
	d, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	p, err := path(g.Name)
	if err != nil {
		return err
	}
	out, err := yaml.Marshal(g)
	if err != nil {
		return err
	}
	return os.WriteFile(p, out, 0o644)
}

// Exists reports whether a group file exists.
func Exists(name string) (bool, error) {
	p, err := path(name)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// Delete removes a group file.
func Delete(name string) error {
	p, err := path(name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("group %q does not exist", name)
		}
		return err
	}
	return nil
}

// List returns sorted group names.
func List() ([]string, error) {
	d, err := Dir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(d)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}
	sort.Strings(names)
	return names, nil
}
