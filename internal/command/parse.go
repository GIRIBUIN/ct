package command

import (
	"fmt"
	"strings"

	"github.com/GIRIBUIN/ct/internal/problem"
)

type options struct {
	id       string
	platform string
	language string
	help     bool
	doctor   bool
}

// parse accepts options on either side of the problem ID.
func parse(args []string) (options, error) {
	if len(args) > 0 && args[0] == "doctor" {
		opts := options{doctor: true}
		for _, arg := range args[1:] {
			if arg != "-h" && arg != "--help" {
				return options{}, fmt.Errorf("ct doctor accepts only --help")
			}
			opts.help = true
		}
		return opts, nil
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
			opts.platform, err = problem.NormalizePlatform(value)
		} else {
			opts.language, err = problem.NormalizeLanguage(value)
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
