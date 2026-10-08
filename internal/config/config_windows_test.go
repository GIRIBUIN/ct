package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateLockedFilePreservesConfig(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "config.json")
	cfg := Defaults(filepath.Join(base, "solutions"))
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// An open file on Windows prevents replacement without delete sharing.
	locked, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer locked.Close()
	cfg.EditorProfile = "changed"
	if err := Update(path, cfg); err == nil {
		t.Fatal("locked configuration was replaced")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("failed replacement corrupted config: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(base, ".ct-config-*.tmp"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary configuration remains after failure: %v, %v", files, err)
	}
}
