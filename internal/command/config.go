package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/GIRIBUIN/ct/internal/config"
	"github.com/GIRIBUIN/ct/internal/editor"
	"github.com/GIRIBUIN/ct/internal/problem"
	"github.com/GIRIBUIN/ct/internal/registry"
	"github.com/GIRIBUIN/ct/internal/template"
	"github.com/GIRIBUIN/ct/internal/version"
)

const usage = `Usage: ct <problem> [-p <platform>] [-l <language>]
       ct doctor [-l <language>]
       ct config
       ct setup [-l <language>] [--yes] [--dry-run]
       ct --version

Commands:
  doctor  Diagnose selected language tools, capabilities, extensions and configuration (read-only)
  config  Configure coding-test root, defaults and VS Code profile interactively
  setup   Plan and install missing components after approval (--dry-run previews only)

Registry:
  ct language list|show|add|remove|enable|disable
  ct platform list|show|add|remove|enable|disable

Platforms: codeforces (cf), programmers (pg)
Languages: cpp (c++), python (py), java, rust (rs)
Defaults: codeforces, cpp; editor: code

Examples:
  ct 71A
  ct 71A -l py
  ct 71A -l java
  ct 71A -l rust
  ct 181188 --platform programmers --language python
`

func Run(args []string, input io.Reader, output io.Writer) error {
	if len(args) > 0 && (args[0] == "language" || args[0] == "platform") {
		return registryCommand(args, input, output)
	}
	r := registry.Builtins()
	// Help/version remain available even when local registry data needs repair.
	if !(len(args) == 1 && (args[0] == "--version" || args[0] == "--help" || args[0] == "-h")) {
		var err error
		r, err = registry.Current()
		if err != nil {
			return err
		}
	}
	opts, err := parseWithRegistry(args, r)
	if err != nil {
		return err
	}
	if opts.help {
		_, err := fmt.Fprint(output, usage)
		return err
	}
	if opts.version {
		_, err := fmt.Fprintln(output, "ct", version.Version)
		return err
	}
	if opts.doctor {
		return doctor(output, opts.language)
	}
	if opts.setup {
		return setup(opts, input, output)
	}
	path, err := config.Path()
	if err != nil {
		return err
	}
	if opts.config {
		_, err := configure(path, input, output, editor.EnsureProfile, saveConfiguration)
		return err
	}
	return run(opts, input, output, path, editor.Open)
}

func run(opts options, input io.Reader, output io.Writer, configPath string, open func(string, string, string, string) error) error {
	r, err := registry.Load(filepath.Dir(configPath))
	if err != nil {
		return err
	}
	cfg, err := config.Load(configPath)
	if errors.Is(err, os.ErrNotExist) {
		cfg, err = configure(configPath, input, output, editor.EnsureProfile, saveConfiguration)
	}
	if err != nil {
		return err
	}
	platform, language := cfg.Platform, cfg.Language
	if opts.platform != "" {
		platform = opts.platform
	}
	if opts.language != "" {
		language = opts.language
	}
	target, err := problem.TargetWithRegistry(r, cfg.Root, platform, language, opts.id)
	if err != nil {
		return err
	}
	// Existing solutions remain usable even if an editable template is missing.
	var data []byte
	if _, statErr := os.Lstat(target); errors.Is(statErr, os.ErrNotExist) {
		data, err = template.LoadWithRegistry(r, platform, language)
		if err != nil {
			return err
		}
	}
	created, err := createData(target, data)
	if err != nil {
		return err
	}
	if created {
		fmt.Fprintln(output, "Created:", target)
	} else {
		fmt.Fprintln(output, "Already exists:", target)
	}
	return open(cfg.Editor, cfg.Root, target, cfg.EditorProfile)
}
