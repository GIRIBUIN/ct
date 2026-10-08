//go:build !windows

package editor

import (
	"context"
	"os/exec"
)

func extensionCommand(ctx context.Context, editor, profile string) *exec.Cmd {
	return extensionOperation(ctx, editor, profile, "--list-extensions")
}

func extensionOperation(ctx context.Context, editor, profile string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, editor, append(profileArgs(profile), args...)...)
}

func findEditor(editor string) (string, error) {
	return exec.LookPath(editor)
}

func launchCommand(editor, root, target, profile string) *exec.Cmd {
	return exec.Command(editor, append(profileArgs(profile), "--reuse-window", root, target)...)
}
