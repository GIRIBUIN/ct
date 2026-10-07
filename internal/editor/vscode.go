package editor

import (
	"fmt"
	"os"
	"strings"
)

func Open(editor, root, target string) error {
	path, err := findEditor(editor)
	if err != nil {
		return fmt.Errorf("find editor %q: %w; install VS Code and make its CLI available on PATH", editor, err)
	}
	cmd := launchCommand(path, root, target)
	cmd.Env = editorEnvironment(cmd.Environ())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open solution in editor %q: %w", editor, err)
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
