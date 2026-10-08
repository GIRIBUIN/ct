package editor

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// EnsureProfile lets VS Code select (or create) a named profile through its CLI.
func EnsureProfile(name, profile string) error {
	path, err := Find(name)
	if err != nil {
		return fmt.Errorf("find VS Code CLI: %w; choose Default or make the CLI available", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	data, err := ListExtensions(ctx, path, profile)
	if err != nil {
		return fmt.Errorf("use VS Code profile %q: %w: %s", profile, err, strings.TrimSpace(string(data)))
	}
	return nil
}

// Find reuses the platform-specific discovery used when opening solutions.
func Find(name string) (string, error) {
	return findEditor(name)
}

// ListExtensions queries the existing user environment without opening a file.
func ListExtensions(ctx context.Context, path, profile string) ([]byte, error) {
	if err := validateProfile(profile); err != nil {
		return nil, err
	}
	cmd := extensionCommand(ctx, path, profile)
	cmd.Env = editorEnvironment(cmd.Environ())
	return cmd.CombinedOutput()
}

func Open(editor, root, target, profile string) error {
	if err := validateProfile(profile); err != nil {
		return err
	}
	path, err := findEditor(editor)
	if err != nil {
		return fmt.Errorf("find editor %q: %w; install VS Code and make its CLI available on PATH", editor, err)
	}
	cmd := launchCommand(path, root, target, profile)
	cmd.Env = editorEnvironment(cmd.Environ())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open solution in editor %q: %w", editor, err)
	}
	return nil
}

func profileArgs(profile string) []string {
	if profile == "" {
		return nil
	}
	return []string{"--profile", profile}
}

func validateProfile(profile string) error {
	// These characters cannot be passed literally through Windows batch quoting.
	if runtime.GOOS == "windows" && strings.ContainsAny(profile, "\"\r\n\x00") {
		return fmt.Errorf("VS Code profile cannot contain double quotes, newlines or NUL on Windows")
	}
	return nil
}

func editorEnvironment(environment []string) []string {
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if !strings.EqualFold(name, "CT_CONFIG_DIR") {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
