package editor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func extensionCommand(ctx context.Context, editor, profile string) *exec.Cmd {
	return extensionOperation(ctx, editor, profile, "--list-extensions")
}

// args are fixed extension operations and validated IDs, never shell input.
func extensionOperation(ctx context.Context, editor, profile string, args ...string) *exec.Cmd {
	ext := filepath.Ext(editor)
	if !strings.EqualFold(ext, ".cmd") && !strings.EqualFold(ext, ".bat") {
		return exec.CommandContext(ctx, editor, append(profileArgs(profile), args...)...)
	}
	profileArgument := ""
	if profile != "" {
		profileArgument = ` --profile "%CT_EDITOR_PROFILE%"`
	}
	cmd := exec.CommandContext(ctx, "cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: `cmd.exe /d /v:off /s /c ""%CT_EDITOR_PATH%"` + profileArgument + " " + strings.Join(args, " ") + `"`,
	}
	cmd.Env = append(os.Environ(), "CT_EDITOR_PATH="+editor, "CT_EDITOR_PROFILE="+profile)
	// Bound waiting for inherited pipes if a batch-file child outlives cmd.exe.
	cmd.WaitDelay = time.Second
	return cmd
}

func findEditor(editor string) (string, error) {
	return discoverWindowsEditor(editor, exec.LookPath, os.Getenv)
}

func discoverWindowsEditor(editor string, lookPath func(string) (string, error), getenv func(string) string) (string, error) {
	path, err := lookPath(editor)
	if err == nil {
		return path, nil
	}
	// Only the default VS Code command receives installation-path fallbacks.
	if !strings.EqualFold(editor, "code") {
		return "", err
	}
	for _, variable := range []string{"LOCALAPPDATA", "ProgramFiles", "ProgramFiles(x86)"} {
		base := getenv(variable)
		if base == "" {
			continue
		}
		parts := []string{base}
		if variable == "LOCALAPPDATA" {
			parts = append(parts, "Programs")
		}
		parts = append(parts, "Microsoft VS Code", "bin", "code.cmd")
		if path, lookupErr := lookPath(filepath.Join(parts...)); lookupErr == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("VS Code CLI not found in PATH or common installations under LOCALAPPDATA, ProgramFiles, or ProgramFiles(x86): %w", err)
}

func launchCommand(editor, root, target, profile string) *exec.Cmd {
	ext := filepath.Ext(editor)
	if !strings.EqualFold(ext, ".cmd") && !strings.EqualFold(ext, ".bat") {
		return exec.Command(editor, append(profileArgs(profile), "--reuse-window", root, target)...)
	}
	profileArgument := ""
	if profile != "" {
		profileArgument = ` --profile "%CT_EDITOR_PROFILE%"`
	}
	// VS Code's Windows CLI is code.cmd. Quoted environment variables keep
	// paths literal; disabled delayed expansion preserves exclamation marks.
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: `cmd.exe /d /v:off /s /c ""%CT_EDITOR_PATH%"` + profileArgument + ` --reuse-window "%CT_EDITOR_ROOT%" "%CT_EDITOR_TARGET%""`,
	}
	cmd.Env = append(os.Environ(), "CT_EDITOR_PATH="+editor, "CT_EDITOR_ROOT="+root, "CT_EDITOR_TARGET="+target, "CT_EDITOR_PROFILE="+profile)
	return cmd
}
