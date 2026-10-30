package tempdir

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewCreatesSecureDir(t *testing.T) {
	base := t.TempDir()
	mgr, err := New(base)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer mgr.Cleanup()

	info, err := os.Stat(mgr.Dir())
	if err != nil {
		t.Fatalf("stat temp dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected a directory")
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o700 {
			t.Errorf("temp dir perm = %o, want 0700", perm)
		}
	}
	if filepath.Dir(mgr.Dir()) != base {
		t.Errorf("temp dir created in %s, want under %s", mgr.Dir(), base)
	}
}

func TestWriteFileSecurePermissions(t *testing.T) {
	mgr, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer mgr.Cleanup()

	path, err := mgr.WriteFile("plan.json", []byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat written file: %v", err)
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("file perm = %o, want 0600", perm)
		}
	}
}

func TestCleanupRemovesDirAndIsIdempotent(t *testing.T) {
	mgr, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dir := mgr.Dir()

	if err := mgr.Cleanup(); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected temp dir to be removed, stat error = %v", err)
	}

	// Calling Cleanup again must not error (idempotent, e.g. called once
	// via defer and once from error-handling code).
	if err := mgr.Cleanup(); err != nil {
		t.Errorf("second Cleanup call returned error: %v", err)
	}
}

func TestCleanupOnSimulatedFailure(t *testing.T) {
	mgr, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dir := mgr.Dir()

	func() {
		defer mgr.Cleanup()
		// Simulate an error path that still reaches the deferred cleanup.
		_ = os.ErrNotExist
	}()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("expected temp dir to be cleaned up after deferred Cleanup on an error path")
	}
}

func TestKeepMovesPlanFile(t *testing.T) {
	mgr, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer mgr.Cleanup()

	if err := os.WriteFile(mgr.PlanPath(), []byte("binary-plan-data"), 0o600); err != nil {
		t.Fatalf("seed plan file: %v", err)
	}

	destDir := t.TempDir()
	kept, err := mgr.Keep(destDir)
	if err != nil {
		t.Fatalf("Keep: %v", err)
	}
	if filepath.Dir(kept) != destDir {
		t.Errorf("kept plan at %s, want under %s", kept, destDir)
	}
	if _, err := os.Stat(kept); err != nil {
		t.Errorf("kept plan file missing: %v", err)
	}
}
