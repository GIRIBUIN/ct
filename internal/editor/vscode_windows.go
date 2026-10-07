package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

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

func launchCommand(editor, root, target string) *exec.Cmd {
	ext := filepath.Ext(editor)
	if !strings.EqualFold(ext, ".cmd") && !strings.EqualFold(ext, ".bat") {
		return exec.Command(editor, "--reuse-window", root, "--goto", target)
	}
	// VS Code's Windows CLI is code.cmd. Quoted environment variables keep
	// paths literal; disabled delayed expansion preserves exclamation marks.
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: `cmd.exe /d /v:off /s /c ""%CT_EDITOR_PATH%" --reuse-window "%CT_EDITOR_ROOT%" --goto "%CT_EDITOR_TARGET%""`,
	}
	cmd.Env = append(os.Environ(), "CT_EDITOR_PATH="+editor, "CT_EDITOR_ROOT="+root, "CT_EDITOR_TARGET="+target)
	return cmd
}
