package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func TestFirstRun(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "config", "config.json")
	root := filepath.Join(base, "coding test 한글")
	var output bytes.Buffer
	got, err := LoadOrCreate(path, strings.NewReader("\""+root+"\"\r\n"), &output)
	if err != nil || got != Defaults(root) {
		t.Fatalf("first run = %+v, %v", got, err)
	}
	output.Reset()
	loaded, err := LoadOrCreate(path, strings.NewReader(""), &output)
	if err != nil || loaded != got || output.Len() != 0 {
		t.Fatalf("second run = %+v, %v; output %q", loaded, err, output.String())
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
	if _, err := LoadOrCreate(path, strings.NewReader("\n"), &bytes.Buffer{}); err == nil {
		t.Fatal("empty root accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("invalid first run created config: %v", err)
	}
	if err := os.WriteFile(path, []byte("invalid json"), 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if _, err := LoadOrCreate(path, strings.NewReader(base), &output); err == nil || output.Len() != 0 {
		t.Fatalf("malformed config prompted or was accepted: %v, %q", err, output.String())
	}
}
