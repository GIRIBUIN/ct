package editor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWindowsExtensionQuery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fake code & cli.cmd")
	data := "@echo off\r\nif defined CT_CONFIG_DIR exit /b 9\r\nif not \"%1\"==\"--list-extensions\" exit /b 8\r\necho ms-vscode.cpptools\r\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CT_CONFIG_DIR", "test-config")
	output, err := ListExtensions(context.Background(), path, "")
	if err != nil || strings.TrimSpace(string(output)) != "ms-vscode.cpptools" {
		t.Fatalf("extension query = %q, %v", output, err)
	}
}

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
	output, err := launchCommand(editor, root, target, "").CombinedOutput()
	if err != nil {
		t.Fatalf("batch launch: %v, %s", err, output)
	}
	// cmd.exe uses the active Windows code page, so test ASCII path portions.
	got := strings.ReplaceAll(string(output), "\r\n", "\n")
	for _, marker := range []string{"--reuse-window\n", "& %PATH% !literal! (x)", `main.cpp"`} {
		if !strings.Contains(got, marker) {
			t.Fatalf("path/argument was altered; missing %q in %q", marker, got)
		}
	}
}

func TestWindowsProfileArguments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fake code.cmd")
	data := "@echo off\r\nif defined CT_CONFIG_DIR exit /b 9\r\necho %1\r\necho %2\r\necho %3\r\necho %4\r\necho %5\r\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	profile := "coding & %PATH% !profile! (x)"
	cmd := launchCommand(path, "test root", "test root/main.cpp", profile)
	cmd.Env = editorEnvironment(cmd.Environ())
	output, err := cmd.CombinedOutput()
	want := "--profile\n\"" + profile + "\"\n--reuse-window\n\"test root\"\n\"test root/main.cpp\"\n"
	if err != nil || strings.ReplaceAll(string(output), "\r\n", "\n") != want {
		t.Fatalf("profile launch = %q, %v; want %q", output, err, want)
	}
	t.Setenv("CT_CONFIG_DIR", "test-config")
	output, err = ListExtensions(context.Background(), path, profile)
	if err != nil || !strings.HasPrefix(strings.ReplaceAll(string(output), "\r\n", "\n"), "--profile\n\""+profile+"\"\n--list-extensions\n") {
		t.Fatalf("profile extension query = %q, %v", output, err)
	}
}

func TestWindowsUnsafeProfileRejected(t *testing.T) {
	for _, profile := range []string{"bad\"profile", "bad\nprofile", "bad\x00profile"} {
		if _, err := ListExtensions(context.Background(), "unused.cmd", profile); err == nil {
			t.Fatalf("unsafe profile accepted: %q", profile)
		}
		if err := Open("unused.cmd", "root", "file", profile); err == nil {
			t.Fatalf("unsafe launch profile accepted: %q", profile)
		}
	}
}

func TestWindowsExtensionInstallationArguments(t *testing.T) {
	cli := filepath.Join(t.TempDir(), "fake code & cli.cmd")
	data := "@echo off\r\nif defined CT_CONFIG_DIR exit /b 9\r\n:args\r\nif \"%~1\"==\"\" exit /b 0\r\necho %1\r\nshift\r\ngoto args\r\n"
	if err := os.WriteFile(cli, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CT_CONFIG_DIR", "test-config")
	for _, profile := range []string{"", "coding & %PATH% !profile! (x)"} {
		t.Run(profile, func(t *testing.T) {
			cmd, err := InstallExtensionCommand(context.Background(), cli, profile, "ms-vscode.cpptools")
			if err != nil {
				t.Fatal(err)
			}
			output, err := cmd.CombinedOutput()
			want := "--install-extension\nms-vscode.cpptools\n"
			if profile != "" {
				want = "--profile\n\"" + profile + "\"\n" + want
			}
			if err != nil || strings.ReplaceAll(string(output), "\r\n", "\n") != want {
				t.Fatalf("extension install = %q, %v; want %q", output, err, want)
			}
		})
	}
}
