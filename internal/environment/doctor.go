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
	"github.com/GIRIBUIN/ct/internal/language"
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

func Diagnose(selected ...string) []Check {
	c := checker{
		lookup: exec.LookPath, findEditor: editor.Find, configPath: config.Path,
		run: func(path string, args ...string) ([]byte, error) {
			ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
			defer cancel()
			cmd := exec.CommandContext(ctx, path, args...)
			// rustup proxies must not auto-install toolchains during read-only diagnosis.
			cmd.Env = append(os.Environ(), "RUSTUP_AUTO_INSTALL=0")
			cmd.WaitDelay = time.Second
			return cmd.CombinedOutput()
		},
		extensions: func(path, profile string) ([]byte, error) {
			ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
			defer cancel()
			return editor.ListExtensions(ctx, path, profile)
		},
	}
	return c.check(selected...)
}

func (c checker) check(override ...string) []Check {
	configuration, editorName, profile := c.configuration()
	results := []Check{{"System", "OS", OK, runtime.GOOS + "/" + runtime.GOARCH}}
	selected := "cpp"
	for _, check := range configuration {
		if check.Name == "language" && check.Status == OK {
			selected = check.Detail
		}
	}
	if len(override) > 0 && override[0] != "" {
		selected = override[0]
	}
	definition, languageErr := language.Lookup(selected)
	if languageErr != nil {
		results = append(results, Check{"Language", "selected", Fail, languageErr.Error()})
	} else {
		selected = definition.Name
		results = append(results, Check{"Language", "selected", OK, selected})
	}
	code, codeErr := c.findEditor(editorName)
	if codeErr != nil {
		results = append(results, Check{"Tools", "code CLI", Fail, "not found: " + codeErr.Error()})
	} else {
		results = append(results, Check{"Tools", "code CLI", OK, code})
	}
	if languageErr == nil {
		results = append(results, c.languageChecks(selected)...)
	}
	results = append(results, c.checkExtensions(code, profile, codeErr, selected)...)
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
