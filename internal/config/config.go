package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GIRIBUIN/ct/internal/configdir"
	"github.com/GIRIBUIN/ct/internal/registry"
)

type Config struct {
	Root          string `json:"root"`
	Platform      string `json:"platform"`
	Language      string `json:"language"`
	Editor        string `json:"editor"`
	EditorProfile string `json:"editor_profile,omitempty"`
}

func Defaults(root string) Config {
	return Config{Root: root, Platform: "codeforces", Language: "cpp", Editor: "code"}
}

func Path() (string, error) {
	dir, err := configdir.Dir()
	if err != nil {
		return "", fmt.Errorf("locate user configuration directory: %w", err)
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read configuration: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode configuration %s: %w", path, err)
	}
	if err := cfg.validate(path, false); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save creates the first-run configuration without replacing an existing one.
func Save(path string, cfg Config) error {
	if err := cfg.validate(path, true); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create configuration: %w", err)
	}
	_, writeErr := file.Write(append(data, '\n'))
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("write configuration: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close configuration: %w", closeErr)
	}
	return nil
}

// Update replaces configuration only after a complete temporary file is flushed
// and closed. The temporary file is on the same filesystem as the destination.
func Update(path string, cfg Config) error {
	if err := cfg.validate(path, true); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".ct-config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary configuration: %w", err)
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(append(data, '\n')); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return fmt.Errorf("write configuration: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close configuration: %w", closeErr)
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return fmt.Errorf("replace configuration: %w", err)
	}
	return nil
}

// Validate checks a proposed configuration without writing or creating paths.
func Validate(path string, cfg Config) error {
	return cfg.validate(path, true)
}

func ValidateRoot(path, root string) error {
	if root == "" || !filepath.IsAbs(root) {
		return errors.New("configuration root must be an absolute directory path")
	}
	if info, err := os.Stat(root); err == nil && !info.IsDir() {
		return errors.New("configuration root is not a directory")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check root directory: %w", err)
	}
	configPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, configPath)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("configuration location must be outside the coding-test root; choose a more specific root directory")
	}
	return nil
}

func (cfg *Config) validate(path string, requireEnabled bool) error {
	if err := ValidateRoot(path, cfg.Root); err != nil {
		return err
	}
	if cfg.Platform == "" {
		cfg.Platform = "codeforces"
	}
	if cfg.Language == "" {
		cfg.Language = "cpp"
	}
	if cfg.Editor == "" {
		cfg.Editor = "code"
	}
	r, err := registry.Load(filepath.Dir(path))
	if err != nil {
		return err
	}
	for _, field := range []struct {
		kind  string
		value *string
	}{{"platform", &cfg.Platform}, {"language", &cfg.Language}} {
		d, err := r.Lookup(field.kind, *field.value, requireEnabled)
		if err != nil {
			if requireEnabled {
				return err
			}
			// A removed/disabled default must remain repairable through ct config,
			// and explicit generation overrides must still be usable.
			*field.value = strings.ToLower(strings.TrimSpace(*field.value))
		} else {
			*field.value = d.Name
		}
	}
	return nil
}
