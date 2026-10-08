package command

import (
	"fmt"
	"strings"

	"github.com/GIRIBUIN/ct/internal/registry"
)

type options struct {
	id       string
	platform string
	language string
	help     bool
	doctor   bool
	config   bool
	setup    bool
	yes      bool
	dryRun   bool
	version  bool
}

// parse accepts options on either side of the problem ID.
func parse(args []string) (options, error) {
	return parseWithRegistry(args, registry.Builtins())
}

func parseWithRegistry(args []string, r *registry.Registry) (options, error) {
	if len(args) == 1 && args[0] == "--version" {
		return options{version: true}, nil
	}
	if len(args) > 0 && (args[0] == "setup" || args[0] == "doctor" || args[0] == "config") {
		return parseCommand(args, r)
	}
	var opts options
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-h" || arg == "--help" {
			opts.help = true
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			if opts.id != "" || arg == "" {
				return options{}, fmt.Errorf("expected exactly one problem ID\n%s", usage)
			}
			opts.id = arg
			continue
		}
		name, value, hasValue := strings.Cut(arg, "=")
		switch name {
		case "-p", "--platform", "-l", "--language":
		default:
			return options{}, fmt.Errorf("unknown option %q\n%s", name, usage)
		}
		if !hasValue {
			i++
			if i == len(args) || strings.HasPrefix(args[i], "-") {
				return options{}, fmt.Errorf("option %s requires a value", name)
			}
			value = args[i]
		}
		var err error
		if name == "-p" || name == "--platform" {
			opts.platform, err = r.Normalize("platform", value)
		} else {
			opts.language, err = r.Normalize("language", value)
		}
		if err != nil {
			return options{}, err
		}
	}
	if opts.id == "" && !opts.help {
		return options{}, fmt.Errorf("a problem ID is required\n%s", usage)
	}
	return opts, nil
}

func parseCommand(args []string, r *registry.Registry) (options, error) {
	opts := options{setup: args[0] == "setup", doctor: args[0] == "doctor", config: args[0] == "config"}
	for i := 1; i < len(args); i++ {
		name, value, hasValue := strings.Cut(args[i], "=")
		switch {
		case !hasValue && (name == "--help" || name == "-h"):
			opts.help = true
		case opts.setup && !hasValue && name == "--yes":
			opts.yes = true
		case opts.setup && !hasValue && name == "--dry-run":
			opts.dryRun = true
		case !opts.config && (name == "-l" || name == "--language"):
			if !hasValue {
				i++
				if i == len(args) || strings.HasPrefix(args[i], "-") {
					return options{}, fmt.Errorf("option %s requires a language", name)
				}
				value = args[i]
			}
			var err error
			opts.language, err = r.Normalize("language", value)
			if err != nil {
				return options{}, err
			}
		default:
			return options{}, fmt.Errorf("unknown %s option %q", args[0], args[i])
		}
	}
	return opts, nil
}
