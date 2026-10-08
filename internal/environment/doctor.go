package environment

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/GIRIBUIN/ct/internal/config"
	"github.com/GIRIBUIN/ct/internal/editor"
)

type Check struct {
	Section string
	Name    string
	Status  string
	Detail  string
}

// Every FAIL is required; SKIP denotes a dependency that already failed.
const (
	OK   = "OK"
	Fail = "FAIL"
	Skip = "SKIP"
)

const checkTimeout = 20 * time.Second

type checker struct {
	lookup     func(string) (string, error)
	findEditor func(string) (string, error)
	run        func(string, ...string) ([]byte, error)
	extensions func(string, string) ([]byte, error)
	configPath func() (string, error)
}

func Diagnose() []Check {
	c := checker{
		lookup: exec.LookPath, findEditor: editor.Find, configPath: config.Path,
		run: func(path string, args ...string) ([]byte, error) {
			ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
			defer cancel()
			cmd := exec.CommandContext(ctx, path, args...)
			cmd.WaitDelay = time.Second
			return cmd.CombinedOutput()
		},
		extensions: func(path, profile string) ([]byte, error) {
			ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
			defer cancel()
			return editor.ListExtensions(ctx, path, profile)
		},
	}
	return c.check()
}

func (c checker) check() []Check {
	configuration, editorName, profile := c.configuration()
	results := []Check{{"System", "OS", OK, runtime.GOOS + "/" + runtime.GOARCH}}
	code, codeErr := c.findEditor(editorName)
	if codeErr != nil {
		results = append(results, Check{"Tools", "code CLI", Fail, "not found: " + codeErr.Error()})
	} else {
		results = append(results, Check{"Tools", "code CLI", OK, code})
	}
	compiler, compilerErr := c.lookup("g++")
	results = append(results, c.tool("g++", compiler, compilerErr))
	debugger, debuggerErr := c.lookup("gdb")
	results = append(results, c.tool("GDB", debugger, debuggerErr))
	if compilerErr != nil {
		results = append(results,
			Check{"Capabilities", "C++20 compile", Skip, "g++ unavailable"},
			Check{"Capabilities", "bits/stdc++.h", Skip, "g++ unavailable"})
	} else {
		results = append(results, c.compile(compiler)...)
	}
	results = append(results, c.checkExtensions(code, profile, codeErr)...)
	return append(results, configuration...)
}

func (c checker) configuration() ([]Check, string, string) {
	path, err := c.configPath()
	var cfg config.Config
	if err == nil {
		cfg, err = config.Load(path)
	}
	if err != nil {
		return []Check{
			{"Configuration", "config", Fail, fmt.Sprintf("%s: %v", path, err)},
			{"Configuration", "root", Skip, "configuration unavailable"},
			{"Configuration", "platform", Skip, "configuration unavailable"},
			{"Configuration", "language", Skip, "configuration unavailable"},
			{"Configuration", "VS Code profile", OK, "default"},
		}, "code", ""
	}
	root := Check{"Configuration", "root", OK, cfg.Root}
	info, err := os.Stat(cfg.Root)
	if err != nil {
		root.Status, root.Detail = Fail, fmt.Sprintf("%s: %v", cfg.Root, err)
	} else if !info.IsDir() {
		root.Status, root.Detail = Fail, cfg.Root+": not a directory"
	}
	profileLabel := cfg.EditorProfile
	if profileLabel == "" {
		profileLabel = "default"
	}
	return []Check{
		{"Configuration", "config", OK, path}, root,
		{"Configuration", "platform", OK, cfg.Platform},
		{"Configuration", "language", OK, cfg.Language},
		{"Configuration", "VS Code profile", OK, profileLabel},
	}, cfg.Editor, cfg.EditorProfile
}

func concise(data []byte) string {
	text := strings.Join(strings.Fields(string(data)), " ")
	if len(text) > 240 {
		text = text[:240] + "..."
	}
	return text
}

func failureDetail(data []byte, err error) string {
	if detail := concise(data); detail != "" {
		return fmt.Sprintf("%v: %s", err, detail)
	}
	return err.Error()
}
