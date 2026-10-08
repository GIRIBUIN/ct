// Package configdir resolves machine-local ct storage, shared by config and registry.
package configdir

import (
	"fmt"
	"os"
	"path/filepath"
)

func Dir() (string, error) {
	if dir := os.Getenv("CT_CONFIG_DIR"); dir != "" {
		return filepath.Abs(dir)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user configuration directory: %w", err)
	}
	return filepath.Join(dir, "ct"), nil
}
