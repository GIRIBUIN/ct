package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GIRIBUIN/ct/internal/template"
)

func create(target, platform, language string) (bool, error) {
	data, err := template.Load(platform, language)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return false, fmt.Errorf("create problem directory: %w", err)
	}
	// O_EXCL makes preservation atomic, including simultaneous ct invocations.
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if errors.Is(err, os.ErrExist) {
		info, statErr := os.Stat(target)
		if statErr != nil {
			return false, fmt.Errorf("inspect existing solution: %w", statErr)
		}
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("target is not a regular solution file: %s", target)
		}
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("create solution: %w", err)
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return false, fmt.Errorf("write solution: %w", writeErr)
	}
	if closeErr != nil {
		return false, fmt.Errorf("close solution: %w", closeErr)
	}
	return true, nil
}
