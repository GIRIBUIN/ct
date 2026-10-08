package command

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/GIRIBUIN/ct/internal/config"
	"github.com/GIRIBUIN/ct/internal/editor"
	"github.com/GIRIBUIN/ct/internal/problem"
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
	opts, err := parse(args)
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
	target, err := problem.Target(cfg.Root, platform, language, opts.id)
	if err != nil {
		return err
	}
	created, err := create(target, platform, language)
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
