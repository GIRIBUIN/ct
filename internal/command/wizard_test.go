package command

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GIRIBUIN/ct/internal/config"
)

func TestFirstRunWizardAndAliases(t *testing.T) {
	for _, tt := range []struct{ input, platform, language string }{
		{"\n\n\n", "codeforces", "cpp"},
		{"pg\npy\n\n", "programmers", "python"},
		{"pg\njava\n\n", "programmers", "java"},
		{"cf\nrs\n\n", "codeforces", "rust"},
		{"cf\nc++\n\n", "codeforces", "cpp"},
		{"invalid-platform\npg\ninvalid-language\npy\n\n", "programmers", "python"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			base := t.TempDir()
			path := filepath.Join(base, "config", "config.json")
			root := filepath.Join(base, "coding test 한글")
			var output bytes.Buffer
			cfg, err := configure(path, strings.NewReader("\n\""+root+"\"\r\n"+tt.input), &output, func(string, string) error { t.Fatal("default profile triggered CLI"); return nil }, saveConfiguration)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Root != root || cfg.Platform != tt.platform || cfg.Language != tt.language || cfg.EditorProfile != "" {
				t.Fatalf("unexpected config: %+v", cfg)
			}
			loaded, err := config.Load(path)
			if err != nil || loaded != cfg {
				t.Fatalf("saved config mismatch: %+v, %v", loaded, err)
			}
			for _, marker := range []string{"ct initial configuration", "Coding-test root is required.", "VS Code profile [Default]", "VS Code profile  Default", "Configuration saved:"} {
				if !strings.Contains(output.String(), marker) {
					t.Errorf("missing %q: %s", marker, output.String())
				}
			}
			if strings.Contains(tt.input, "invalid-") && (!strings.Contains(output.String(), "unsupported platform") || !strings.Contains(output.String(), "unsupported language")) {
				t.Fatal("validation errors were not reported")
			}
		})
	}
}

func TestWizardExistingEnterPreservesValues(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "config.json")
	want := config.Defaults(filepath.Join(base, "solutions"))
	want.Platform, want.Language, want.Editor, want.EditorProfile = "programmers", "python", "custom-code", " my exact profile "
	if err := config.Save(path, want); err != nil {
		t.Fatal(err)
	}
	called := false
	var output bytes.Buffer
	got, err := configure(path, strings.NewReader("\n\n\n\n\n"), &output, func(editor, profile string) error {
		called = true
		if editor != want.Editor || profile != want.EditorProfile {
			t.Fatalf("wrong profile validation: %q %q", editor, profile)
		}
		return nil
	}, saveConfiguration)
	if err != nil || got != want || !called {
		t.Fatalf("current values changed: %+v, %v", got, err)
	}
	loaded, err := config.Load(path)
	if err != nil || loaded != want {
		t.Fatalf("current values not saved: %+v, %v", loaded, err)
	}
	if !strings.Contains(output.String(), "Default platform [programmers]") || !strings.Contains(output.String(), "Default language [python]") {
		t.Fatalf("missing current defaults: %s", output.String())
	}
}

func TestWizardProfileSelection(t *testing.T) {
	for _, tt := range []struct {
		name, input, want string
		failFirst         bool
		calls             int
	}{
		{"accepted", " my exact profile \ny\n", " my exact profile ", false, 1},
		{"declined", "rejected\nn\naccepted\n\n", "accepted", false, 1},
		{"declined-default", "rejected\nn\n\n", "", false, 0},
		{"retry-unavailable", "retry profile\ny\nretry profile\ny\n", "retry profile", true, 2},
		{"unavailable-default", "unavailable\ny\n\n", "", true, 1},
		{"confirmation-retry", "accepted\ninvalid\nyes\n", "accepted", false, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			path := filepath.Join(base, "config.json")
			root := filepath.Join(base, "solutions")
			calls := 0
			var output bytes.Buffer
			got, err := configure(path, strings.NewReader(root+"\n\n\n"+tt.input), &output, func(editor, profile string) error {
				calls++
				if profile == "rejected" {
					t.Fatal("declined profile reached CLI")
				}
				if tt.failFirst && calls == 1 {
					return errors.New("VS Code CLI unavailable")
				}
				return nil
			}, saveConfiguration)
			if err != nil || got.EditorProfile != tt.want || calls != tt.calls {
				t.Fatalf("profile=%q, calls=%d, err=%v; output=%s", got.EditorProfile, calls, err, output.String())
			}
			if !strings.Contains(output.String(), "may create a new empty profile") {
				t.Fatal("profile creation warning missing")
			}
			if tt.failFirst && !strings.Contains(output.String(), "Profile unavailable:") {
				t.Fatal("CLI failure hidden")
			}
		})
	}
}

func TestWizardClearExistingProfile(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "config.json")
	cfg := config.Defaults(filepath.Join(base, "solutions"))
	cfg.EditorProfile = "previous"
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := configure(path, strings.NewReader("\n\n\n-\n"), &bytes.Buffer{}, func(string, string) error { t.Fatal("Default triggered CLI"); return nil }, saveConfiguration)
	if err != nil || got.EditorProfile != "" {
		t.Fatalf("profile not cleared: %+v, %v", got, err)
	}
}

func TestWizardSaveFailureAndEOFPreserveConfig(t *testing.T) {
	for _, mode := range []string{"save-failure", "eof", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			base := t.TempDir()
			path := filepath.Join(base, "config.json")
			cfg := config.Defaults(filepath.Join(base, "solutions"))
			if err := config.Save(path, cfg); err != nil {
				t.Fatal(err)
			}
			if mode == "malformed" {
				if err := os.WriteFile(path, []byte("invalid json"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			input := "\npg\npy\n\n"
			if mode == "eof" {
				input = ""
			}
			_, err = configure(path, strings.NewReader(input), &bytes.Buffer{}, func(string, string) error { return nil }, func(string, config.Config, bool) error { return errors.New("disk write failure") })
			if err == nil {
				t.Fatal("failure ignored")
			}
			after, readErr := os.ReadFile(path)
			if readErr != nil || !bytes.Equal(before, after) {
				t.Fatalf("existing config corrupted: %v", readErr)
			}
		})
	}
}

func TestConfigParsingAndCommand(t *testing.T) {
	for _, args := range [][]string{{"config"}, {"config", "--help"}} {
		opts, err := parse(args)
		if err != nil || !opts.config || opts.id != "" || opts.doctor {
			t.Fatalf("config parsed as problem: %+v, %v", opts, err)
		}
	}
	for _, args := range [][]string{{"config", "71A"}, {"config", "-p", "pg"}} {
		if _, err := parse(args); err == nil {
			t.Fatalf("invalid config arguments accepted: %v", args)
		}
	}
	base := t.TempDir()
	t.Setenv("CT_CONFIG_DIR", filepath.Join(base, "config"))
	root := filepath.Join(base, "solutions")
	var output bytes.Buffer
	if err := Run([]string{"config"}, strings.NewReader(root+"\n\n\n\n"), &output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("ct config created solution root: %v", err)
	}
	path, err := config.Path()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil || cfg.Root != root {
		t.Fatalf("ct config failed to save: %+v, %v", cfg, err)
	}
	output.Reset()
	if err := Run([]string{"--help"}, strings.NewReader(""), &output); err != nil || !strings.Contains(output.String(), "ct config") || !strings.Contains(output.String(), "ct doctor") {
		t.Fatalf("help incomplete: %s, %v", output.String(), err)
	}
}
