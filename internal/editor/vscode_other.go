//go:build !windows

package editor

import "os/exec"

func findEditor(editor string) (string, error) {
	return exec.LookPath(editor)
}

func launchCommand(editor, root, target string) *exec.Cmd {
	return exec.Command(editor, "--reuse-window", root, "--goto", target)
}
