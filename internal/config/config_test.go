package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPathOverride(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ct config")
	t.Setenv("CT_CONFIG_DIR", dir)
	got, err := Path()
	want := filepath.Join(dir, "config.json")
	if err != nil || got != want {
		t.Fatalf("Path = %q, %v; want %q", got, err, want)
	}
}

func TestPathDefault(t *testing.T) {
	for _, absent := range []bool{true, false} {
		name := "empty"
		if absent {
			name = "absent"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("CT_CONFIG_DIR", "")
			if absent {
				if err := os.Unsetenv("CT_CONFIG_DIR"); err != nil {
					t.Fatal(err)
				}
			}
			dir, err := os.UserConfigDir()
			if err != nil {
				t.Fatal(err)
			}
			got, err := Path()
			want := filepath.Join(dir, "ct", "config.json")
			if err != nil || got != want {
				t.Fatalf("Path = %q, %v; want %q", got, err, want)
			}
		})
	}
}

func TestSaveLoad(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CT_CONFIG_DIR", filepath.Join(base, "config"))
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	want := Defaults(filepath.Join(base, "solutions"))
	want.Platform, want.Language, want.Editor = "programmers", "python", "custom-code"
	want.EditorProfile = "my coding profile"
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || got != want {
		t.Fatalf("Load = %+v, %v; want %+v", got, err, want)
	}
	if err := Save(path, Defaults(filepath.Join(base, "other"))); err == nil {
		t.Fatal("existing configuration replaced")
	}
	got, err = Load(path)
	if err != nil || got != want {
		t.Fatalf("configuration changed: %+v, %v", got, err)
	}
}

func TestLegacyConfigWithoutProfile(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "config.json")
	want := Defaults(filepath.Join(base, "solutions"))
	data, err := json.Marshal(map[string]string{
		"root": want.Root, "platform": want.Platform, "language": want.Language, "editor": want.Editor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || got != want || got.EditorProfile != "" {
		t.Fatalf("legacy config = %+v, %v; want %+v", got, err, want)
	}
	if saved, err := os.ReadFile(path); err != nil || !bytes.Equal(saved, data) {
		t.Fatalf("loading legacy config changed the file: %v", err)
	}
}

func TestInvalidConfiguration(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "config.json")
	if err := Save(path, Defaults(base)); err == nil {
		t.Fatal("configuration allowed inside solution root")
	}
	if err := Save(path, Defaults("relative")); err == nil {
		t.Fatal("relative configured root accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("invalid first run created config: %v", err)
	}
	if err := os.WriteFile(path, []byte("invalid json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("malformed config accepted")
	}
}

func TestUpdatePreservesValidConfigOnFailure(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "config.json")
	original := Defaults(filepath.Join(base, "solutions"))
	if err := Save(path, original); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Update(path, Defaults("relative")); err == nil {
		t.Fatal("invalid update succeeded")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("failed save corrupted config: %v", err)
	}
	updated := original
	updated.EditorProfile = "my profile"
	if err := Update(path, updated); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || got != updated {
		t.Fatalf("updated config = %+v, %v", got, err)
	}
	files, err := filepath.Glob(filepath.Join(base, ".ct-config-*.tmp"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary configuration remains: %v, %v", files, err)
	}
}
