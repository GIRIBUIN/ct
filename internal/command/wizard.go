package command

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/GIRIBUIN/ct/internal/config"
	"github.com/GIRIBUIN/ct/internal/registry"
)

func saveConfiguration(path string, cfg config.Config, existing bool) error {
	if existing {
		return config.Update(path, cfg)
	}
	return config.Save(path, cfg)
}

func configure(path string, input io.Reader, output io.Writer, ensureProfile func(string, string) error, save func(string, config.Config, bool) error) (config.Config, error) {
	r, err := registry.Load(filepath.Dir(path))
	if err != nil {
		return config.Config{}, err
	}
	cfg, err := config.Load(path)
	existing := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return config.Config{}, err
	}
	if !existing {
		cfg = config.Defaults("")
		fmt.Fprintln(output, "ct initial configuration")
	} else {
		fmt.Fprintln(output, "ct configuration")
	}
	for kind, value := range map[string]string{"platform": cfg.Platform, "language": cfg.Language} {
		if _, err := r.Normalize(kind, value); err != nil {
			fmt.Fprintln(output, "Current default:", err)
		}
	}
	reader := bufio.NewReader(input)
	prompt := func(label, current string) (string, error) {
		if current == "" {
			fmt.Fprintf(output, "%s: ", label)
		} else {
			fmt.Fprintf(output, "%s [%s]: ", label, current)
		}
		line, err := reader.ReadString('\n')
		if err != nil && !(errors.Is(err, io.EOF) && len(line) > 0) {
			return "", fmt.Errorf("read %s: %w", label, err)
		}
		return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
	}
	for {
		value, err := prompt("Coding-test root", cfg.Root)
		if err != nil {
			return config.Config{}, err
		}
		root := strings.TrimSpace(value)
		if root == "" {
			root = cfg.Root
		}
		if len(root) >= 2 && root[0] == '"' && root[len(root)-1] == '"' {
			root = root[1 : len(root)-1]
		}
		if root == "" {
			fmt.Fprintln(output, "Coding-test root is required.")
			continue
		}
		root, err = filepath.Abs(root)
		if err == nil {
			err = config.ValidateRoot(path, root)
		}
		if err != nil {
			fmt.Fprintln(output, "Invalid root:", err)
			continue
		}
		cfg.Root = root
		break
	}
	for _, field := range []struct {
		label     string
		value     *string
		normalize func(string) (string, error)
	}{
		{"Default platform", &cfg.Platform, func(v string) (string, error) { return r.Normalize("platform", v) }},
		{"Default language", &cfg.Language, func(v string) (string, error) { return r.Normalize("language", v) }},
	} {
		for {
			value, err := prompt(field.label, *field.value)
			if err != nil {
				return config.Config{}, err
			}
			if strings.TrimSpace(value) == "" {
				value = *field.value
			}
			normalized, err := field.normalize(value)
			if err != nil {
				fmt.Fprintln(output, err)
				continue
			}
			*field.value = normalized
			break
		}
	}
	for {
		label := cfg.EditorProfile
		if label == "" {
			label = "Default"
		}
		fmt.Fprintln(output, "Enter keeps the current profile; - selects Default.")
		value, err := prompt("VS Code profile", label)
		if err != nil {
			return config.Config{}, err
		}
		profile := value
		if profile == "" {
			profile = cfg.EditorProfile
		}
		if profile == "-" {
			profile = ""
		}
		if profile != "" {
			fmt.Fprintf(output, "VS Code profile %q will be used.\nIf it does not already exist, VS Code may create a new empty profile.\n", profile)
			accepted := false
			for {
				answer, err := prompt("Continue?", "Y/n")
				if err != nil {
					return config.Config{}, err
				}
				switch strings.ToLower(strings.TrimSpace(answer)) {
				case "", "y", "yes":
					accepted = true
				case "n", "no":
				default:
					fmt.Fprintln(output, "Please answer y or n.")
					continue
				}
				break
			}
			if !accepted {
				continue
			}
			if err := ensureProfile(cfg.Editor, profile); err != nil {
				fmt.Fprintln(output, "Profile unavailable:", err)
				fmt.Fprintln(output, "Choose - for Default or enter a profile name to retry.")
				continue
			}
		}
		cfg.EditorProfile = profile
		break
	}
	if err := save(path, cfg, existing); err != nil {
		return config.Config{}, err
	}
	profile := cfg.EditorProfile
	if profile == "" {
		profile = "Default"
	}
	fmt.Fprintf(output, "Configuration saved: %s\n\nroot             %s\nplatform         %s\nlanguage         %s\nVS Code profile  %s\n", path, cfg.Root, cfg.Platform, cfg.Language, profile)
	return cfg, nil
}
