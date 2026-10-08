package installer

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/GIRIBUIN/ct/internal/config"
	"github.com/GIRIBUIN/ct/internal/editor"
	"github.com/GIRIBUIN/ct/internal/environment"
)

type Action struct {
	ID          string
	Description string
	Details     []string
	DependsOn   []string
	Execute     func() error
}

type Plan struct {
	Actions []Action
	Notes   []string
}

type invocation struct {
	path string
	args []string
	env  []string
}

func (c invocation) String() string {
	parts := []string{strconv.Quote(c.path)}
	for _, arg := range c.args {
		parts = append(parts, strconv.Quote(arg))
	}
	if len(c.env) > 0 {
		return strings.Join(c.env, " ") + " " + strings.Join(parts, " ")
	}
	return strings.Join(parts, " ")
}

// host contains only the small OS boundary needed for planning and execution.
// Tests replace it; production never invokes a mutation while building a plan.
type host struct {
	goos, arch       string
	lookup           func(string) (string, error)
	getenv           func(string) string
	setenv           func(string, string) error
	readFile         func(string) ([]byte, error)
	exists           func(string) bool
	query            func(invocation) ([]byte, error)
	run              func(invocation) error
	installExtension func(string, string, string) error
	loadConfig       func() (config.Config, error)
	uid              func() int
}

func BuildPlan(checks []environment.Check, input io.Reader, output io.Writer) (Plan, error) {
	h := host{
		goos: runtime.GOOS, arch: runtime.GOARCH, lookup: exec.LookPath,
		getenv: os.Getenv, setenv: os.Setenv, readFile: os.ReadFile, uid: os.Geteuid,
		exists: func(path string) bool { _, err := os.Stat(path); return err == nil },
		loadConfig: func() (config.Config, error) {
			path, err := config.Path()
			if err != nil {
				return config.Config{}, err
			}
			return config.Load(path)
		},
		query: func(c invocation) ([]byte, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, c.path, c.args...)
			cmd.Env = append(os.Environ(), c.env...)
			cmd.WaitDelay = time.Second
			return cmd.CombinedOutput()
		},
		run: func(c invocation) error {
			cmd := exec.Command(c.path, c.args...)
			cmd.Env = append(os.Environ(), c.env...)
			cmd.Stdin, cmd.Stdout, cmd.Stderr = input, output, output
			return cmd.Run()
		},
		installExtension: func(path, profile, id string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			cmd, err := editor.InstallExtensionCommand(ctx, path, profile, id)
			if err != nil {
				return err
			}
			cmd.Stdin, cmd.Stdout, cmd.Stderr = input, output, output
			return cmd.Run()
		},
	}
	return h.plan(checks)
}

func lookupCheck(checks []environment.Check, name string) environment.Check {
	for _, check := range checks {
		if check.Name == name {
			return check
		}
	}
	return environment.Check{Name: name, Status: environment.Skip}
}

func (h host) commandAction(id, description string, c invocation, dependencies ...string) Action {
	return Action{ID: id, Description: description, Details: []string{c.String()}, DependsOn: dependencies, Execute: func() error { return h.run(c) }}
}

func (h host) plan(checks []environment.Check) (Plan, error) {
	var plan Plan
	missingCompiler := lookupCheck(checks, "g++").Status == environment.Fail
	missingDebugger := lookupCheck(checks, "GDB").Status == environment.Fail
	if missingCompiler || missingDebugger {
		switch h.goos {
		case "windows":
			h.planWindows(&plan, missingCompiler, missingDebugger)
		case "linux":
			h.planLinux(&plan, missingCompiler, missingDebugger)
		default:
			plan.Notes = append(plan.Notes, "Automatic toolchain installation is unsupported on "+h.goos+". Install g++ and GDB manually, then run ct doctor.")
		}
	}
	for _, name := range []string{"C++20 compile", "bits/stdc++.h"} {
		if lookupCheck(checks, name).Status == environment.Fail {
			plan.Notes = append(plan.Notes, name+" failed with an existing compiler. Repair/select a compatible GCC toolchain manually; setup will not replace it.")
		}
	}
	if lookupCheck(checks, "code CLI").Status != environment.OK {
		plan.Notes = append(plan.Notes, "VS Code installation is not automated. Install it from https://code.visualstudio.com/ and enable its CLI, then rerun ct setup.")
	} else if lookupCheck(checks, "extension query").Status == environment.Fail {
		plan.Notes = append(plan.Notes, "Extension query failed; installation state is unknown. Fix the VS Code CLI/profile and rerun ct doctor.")
	} else {
		for _, extension := range environment.RequiredExtensions {
			if lookupCheck(checks, extension.Name).Status != environment.Fail {
				continue
			}
			cfg, err := h.loadConfig()
			if err != nil {
				plan.Notes = append(plan.Notes, "Cannot determine the configured profile; run ct config first: "+err.Error())
				break
			}
			path := lookupCheck(checks, "code CLI").Detail
			id, profile := extension.ID, cfg.EditorProfile
			args := []string{}
			label := "Default"
			if profile != "" {
				args = append(args, "--profile", profile)
				label = strconv.Quote(profile)
			}
			args = append(args, "--install-extension", id)
			plan.Actions = append(plan.Actions, Action{ID: "extension:" + id, Description: fmt.Sprintf("Install %s into VS Code profile %s", id, label), Details: []string{(invocation{path: path, args: args}).String()}, Execute: func() error { return h.installExtension(path, profile, id) }})
		}
	}
	for _, name := range []string{"config", "root", "platform", "language"} {
		if lookupCheck(checks, name).Status == environment.Fail {
			plan.Notes = append(plan.Notes, "Configuration check failed ("+name+"). Run ct config or repair the configured root; setup does not change configuration or solution directories.")
		}
	}
	return plan, nil
}
