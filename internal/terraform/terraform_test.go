package terraform

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFilterDisallowedArgs(t *testing.T) {
	in := []string{"-var-file=prod.tfvars", "-detailed-exitcode", "--detailed-exitcode", "-target=module.x"}
	got := FilterDisallowedArgs(in)
	want := []string{"-var-file=prod.tfvars", "-target=module.x"}
	if len(got) != len(want) {
		t.Fatalf("FilterDisallowedArgs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("FilterDisallowedArgs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLooksUninitializedMissingTerraformDir(t *testing.T) {
	dir := t.TempDir()
	if !LooksUninitialized(dir) {
		t.Error("expected true when .terraform/ is missing")
	}
}

func TestLooksUninitializedFalseWhenDirPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".terraform"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if LooksUninitialized(dir) {
		t.Error("expected false once .terraform/ exists, even without a lock file")
	}
}

func TestResolveBinaryExplicitPathNotFound(t *testing.T) {
	if _, err := ResolveBinary("/nonexistent/path/to/terraform"); err == nil {
		t.Error("expected error for a nonexistent --terraform-bin path")
	}
}

func TestResolveBinaryEnvOverride(t *testing.T) {
	// Point TF_CLI_BINARY at something guaranteed to resolve via LookPath.
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available on this system")
	}
	t.Setenv("TF_CLI_BINARY", shPath)
	got, err := ResolveBinary("")
	if err != nil {
		t.Fatalf("ResolveBinary: %v", err)
	}
	if got != shPath {
		t.Errorf("ResolveBinary() = %q, want %q", got, shPath)
	}
}
