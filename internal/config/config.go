package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/GIRIBUIN/ct/internal/problem"
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
	if dir := os.Getenv("CT_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "config.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user configuration directory: %w", err)
	}
	return filepath.Join(dir, "ct", "config.json"), nil
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
	if err := cfg.validate(path); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save creates the first-run configuration without replacing an existing one.
func Save(path string, cfg Config) error {
	if err := cfg.validate(path); err != nil {
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

func LoadOrCreate(path string, input io.Reader, output io.Writer) (Config, error) {
	cfg, err := Load(path)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return cfg, err
	}
	fmt.Fprint(output, "Coding-test root directory: ")
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("read root directory: %w", err)
	}
	root := strings.TrimSpace(line)
	// Accept paths copied with quotes from a terminal or file manager.
	if len(root) >= 2 && root[0] == '"' && root[len(root)-1] == '"' {
		root = root[1 : len(root)-1]
	}
	if root == "" {
		return Config{}, errors.New("coding-test root directory is required")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return Config{}, fmt.Errorf("resolve root directory: %w", err)
	}
	cfg = Defaults(root)
	if err := Save(path, cfg); err != nil {
		return Config{}, err
	}
	fmt.Fprintln(output, "Configuration saved:", path)
	return cfg, nil
}

func (cfg *Config) validate(path string) error {
	if cfg.Root == "" || !filepath.IsAbs(cfg.Root) {
		return errors.New("configuration root must be an absolute directory path")
	}
	if info, err := os.Stat(cfg.Root); err == nil && !info.IsDir() {
		return errors.New("configuration root is not a directory")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check root directory: %w", err)
	}
	configPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(cfg.Root, configPath)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("configuration location must be outside the coding-test root; choose a more specific root directory")
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
	cfg.Platform, err = problem.NormalizePlatform(cfg.Platform)
	if err != nil {
		return err
	}
	cfg.Language, err = problem.NormalizeLanguage(cfg.Language)
	return err
}
