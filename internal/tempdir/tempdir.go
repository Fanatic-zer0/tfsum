// Package tempdir manages tfsum's secure, ephemeral working directory for
// the Terraform plan file and its JSON conversion, guaranteeing cleanup on
// success, on error, and on interruption (SIGINT/SIGTERM).
package tempdir

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Manager owns a single `.tfsum-<random>` directory created with 0700
// permissions.
type Manager struct {
	dir string

	mu      sync.Mutex
	removed bool
}

// New creates a `.tfsum-<random>` directory with 0700 permissions inside
// baseDir (use "." for the current working directory).
func New(baseDir string) (*Manager, error) {
	dir, err := os.MkdirTemp(baseDir, ".tfsum-")
	if err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}
	// os.MkdirTemp already creates with 0700, but set it explicitly so
	// the permission is not dependent on platform defaults.
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("secure temp directory: %w", err)
	}
	return &Manager{dir: dir}, nil
}

// Dir returns the path of the managed temp directory.
func (m *Manager) Dir() string { return m.dir }

// PlanPath returns the path to use for the saved Terraform plan file.
func (m *Manager) PlanPath() string {
	return filepath.Join(m.dir, "plan.tfplan")
}

// WriteFile writes data to name inside the temp directory with 0600
// permissions, since plan/state data may contain secrets.
func (m *Manager) WriteFile(name string, data []byte) (string, error) {
	path := filepath.Join(m.dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// Keep moves the saved plan file out of the temp directory into destDir
// (e.g. the current working directory) so it survives Cleanup, returning
// its new path. Used to implement --keep-plan.
func (m *Manager) Keep(destDir string) (string, error) {
	src := m.PlanPath()
	if _, err := os.Stat(src); err != nil {
		return "", fmt.Errorf("plan file not found: %w", err)
	}
	dest := filepath.Join(destDir, "tfsum.tfplan")
	if err := os.Rename(src, dest); err != nil {
		return "", fmt.Errorf("retain plan file: %w", err)
	}
	if err := os.Chmod(dest, 0o600); err != nil {
		return dest, fmt.Errorf("secure retained plan file: %w", err)
	}
	return dest, nil
}

// Cleanup removes the temp directory and everything in it. Safe to call
// more than once (e.g. once via defer and once from a signal handler).
func (m *Manager) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.removed {
		return nil
	}
	m.removed = true
	return os.RemoveAll(m.dir)
}
