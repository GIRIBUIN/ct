package problem

import (
	"fmt"
	"github.com/GIRIBUIN/ct/internal/platform"
	"github.com/GIRIBUIN/ct/internal/registry"
	"path/filepath"
)

func NormalizePlatform(value string) (string, error) {
	return registry.Builtins().Normalize("platform", value)
}

func NormalizeLanguage(value string) (string, error) {
	return registry.Builtins().Normalize("language", value)
}

// Target validates the ID before using it as a directory name.
func Target(root, platform, language, id string) (string, error) {
	return TargetWithRegistry(registry.Builtins(), root, platform, language, id)
}

func TargetWithRegistry(r *registry.Registry, root, p, l, id string) (string, error) {
	b, err := r.Binding(p, l)
	if err != nil {
		return "", err
	}
	if err := platform.ValidateID(b.Platform, id); err != nil {
		return "", err
	}
	if err := registry.Filename(id); err != nil {
		return "", fmt.Errorf("invalid problem directory: %w", err)
	}
	return filepath.Join(root, b.Platform, id, b.Filename), nil
}
