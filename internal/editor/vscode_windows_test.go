package editor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWindowsEditorDiscoveryOrder(t *testing.T) {
	base := t.TempDir()
	environment := map[string]string{
		"LOCALAPPDATA":      filepath.Join(base, "local app data"),
		"ProgramFiles":      filepath.Join(base, "program files"),
		"ProgramFiles(x86)": filepath.Join(base, "program files x86"),
	}
	getenv := func(name string) string { return environment[name] }
	candidates := []string{
		"code",
		filepath.Join(environment["LOCALAPPDATA"], "Programs", "Microsoft VS Code", "bin", "code.cmd"),
		filepath.Join(environment["ProgramFiles"], "Microsoft VS Code", "bin", "code.cmd"),
		filepath.Join(environment["ProgramFiles(x86)"], "Microsoft VS Code", "bin", "code.cmd"),
	}
	for foundAt := range candidates {
		t.Run(candidates[foundAt], func(t *testing.T) {
			var calls []string
			lookup := func(name string) (string, error) {
				calls = append(calls, name)
				if name == candidates[foundAt] {
					return name, nil
				}
				return "", exec.ErrNotFound
			}
			got, err := discoverWindowsEditor("code", lookup, getenv)
			if err != nil || got != candidates[foundAt] {
				t.Fatalf("discovery = %q, %v; want %q", got, err, candidates[foundAt])
			}
			if !reflect.DeepEqual(calls, candidates[:foundAt+1]) {
				t.Fatalf("lookup order = %q; want %q", calls, candidates[:foundAt+1])
			}
		})
	}
}

func TestWindowsEditorDiscoveryTemporaryCLI(t *testing.T) {
	base := t.TempDir()
	want := filepath.Join(base, "Programs", "Microsoft VS Code", "bin", "code.cmd")
	if err := os.MkdirAll(filepath.Dir(want), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("@echo off\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	lookup := func(name string) (string, error) {
		if name == "code" {
			return "", exec.ErrNotFound
		}
		return exec.LookPath(name)
	}
	getenv := func(name string) string {
		if name == "LOCALAPPDATA" {
			return base
		}
		return ""
	}
	got, err := discoverWindowsEditor("code", lookup, getenv)
	if err != nil || got != want {
		t.Fatalf("temporary CLI discovery = %q, %v; want %q", got, err, want)
	}
}

func TestWindowsEditorDiscoveryMissingOrCustom(t *testing.T) {
	for _, editor := range []string{"code", "custom-code"} {
		t.Run(editor, func(t *testing.T) {
			var calls []string
			lookup := func(name string) (string, error) {
				calls = append(calls, name)
				return "", exec.ErrNotFound
			}
			getenv := func(string) string {
				if editor != "code" {
					t.Fatal("custom editor triggered VS Code fallback")
				}
				return ""
			}
			_, err := discoverWindowsEditor(editor, lookup, getenv)
			if !errors.Is(err, exec.ErrNotFound) {
				t.Fatalf("missing lookup error: %v", err)
			}
			if !reflect.DeepEqual(calls, []string{editor}) {
				t.Fatalf("unset environment generated relative fallback paths: %q", calls)
			}
			if editor == "code" && !strings.Contains(err.Error(), "PATH or common installations") {
				t.Fatalf("missing discovery details: %v", err)
			}
		})
	}
}

func TestWindowsBatchPaths(t *testing.T) {
	base := t.TempDir()
	editor := filepath.Join(base, "fake code.cmd")
	// Simulate code.cmd handing quoted arguments to the VS Code executable.
	if err := os.WriteFile(editor, []byte("@echo off\r\nsetlocal\r\necho %1\r\necho %2\r\necho %3\r\necho %4\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "coding test 한글 & %PATH% !literal! (x)")
	target := filepath.Join(root, "main.cpp")
	output, err := launchCommand(editor, root, target).CombinedOutput()
	if err != nil {
		t.Fatalf("batch launch: %v, %s", err, output)
	}
	// cmd.exe uses the active Windows code page, so test ASCII path portions.
	got := strings.ReplaceAll(string(output), "\r\n", "\n")
	for _, marker := range []string{"--reuse-window\n", "--goto\n", "& %PATH% !literal! (x)", `main.cpp"`} {
		if !strings.Contains(got, marker) {
			t.Fatalf("path/argument was altered; missing %q in %q", marker, got)
		}
	}
}
