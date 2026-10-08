package command

import (
	"fmt"
	"io"

	"github.com/GIRIBUIN/ct/internal/config"
	"github.com/GIRIBUIN/ct/internal/editor"
	"github.com/GIRIBUIN/ct/internal/problem"
)

const usage = `Usage: ct <problem> [-p <platform>] [-l <language>]
       ct doctor

Commands:
  doctor  Diagnose tools, C++ capabilities, extensions and configuration (read-only)

Platforms: codeforces (cf), programmers (pg)
Languages: cpp (c++), python (py)
Defaults: codeforces, cpp; editor: code

Examples:
  ct 71A
  ct 71A -l py
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
	if opts.doctor {
		return doctor(output)
	}
	path, err := config.Path()
	if err != nil {
		return err
	}
	return run(opts, input, output, path, editor.Open)
}

func run(opts options, input io.Reader, output io.Writer, configPath string, open func(string, string, string, string) error) error {
	cfg, err := config.LoadOrCreate(configPath, input, output)
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
