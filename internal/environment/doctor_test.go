package environment

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GIRIBUIN/ct/internal/config"
)

func TestDoctorUsesConfiguredProfile(t *testing.T) {
	for _, profile := range []string{"", "my coding profile"} {
		t.Run("profile="+profile, func(t *testing.T) {
			c, path, root := healthyChecker(t)
			cfg := config.Defaults(root)
			cfg.EditorProfile = profile
			data, err := json.Marshal(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			calls := 0
			c.extensions = func(code, selectedProfile string) ([]byte, error) {
				calls++
				if code != "code" || selectedProfile != profile {
					t.Fatalf("wrong extension query: %q %q", code, selectedProfile)
				}
				if selectedProfile == "" {
					return nil, nil
				}
				return []byte("ms-vscode.cpptools\ndivyanshuagrawal.competitive-programming-helper\n"), nil
			}
			results := c.check()
			wantStatus, label := OK, profile
			if profile == "" {
				wantStatus, label = Fail, "default"
			}
			if calls != 1 || status(t, results, "C/C++") != wantStatus || status(t, results, "CPH") != wantStatus {
				t.Fatalf("profile extensions not respected: %+v", results)
			}
			found := false
			for _, result := range results {
				if result.Name == "VS Code profile" {
					found = true
					if result.Status != OK || result.Detail != label {
						t.Fatalf("incorrect profile report: %+v", result)
					}
				}
			}
			if !found {
				t.Fatal("profile missing from report")
			}
		})
	}
}

func healthyChecker(t *testing.T) (checker, string, string) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "solutions")
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, "config.json")
	if err := config.Save(path, config.Defaults(root)); err != nil {
		t.Fatal(err)
	}
	c := checker{
		lookup:     func(name string) (string, error) { return name, nil },
		findEditor: func(name string) (string, error) { return name, nil },
		run: func(path string, args ...string) ([]byte, error) {
			if len(args) == 1 && args[0] == "--version" {
				return []byte(path + " 1.2.3\nlicense text"), nil
			}
			return nil, nil
		},
		extensions: func(string, string) ([]byte, error) {
			return []byte("MS-VSCODE.CPPTOOLS\r\nDivyanshuAgrawal.Competitive-Programming-Helper\n"), nil
		},
		configPath: func() (string, error) { return path, nil },
	}
	return c, path, root
}

func status(t *testing.T, results []Check, name string) string {
	t.Helper()
	for _, result := range results {
		if result.Name == name {
			return result.Status
		}
	}
	t.Fatalf("missing check %q in %+v", name, results)
	return ""
}

func TestHealthyDoctorAndReadOnlyConfiguration(t *testing.T) {
	c, path, root := healthyChecker(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	results := c.check()
	for _, result := range results {
		if result.Status != OK {
			t.Errorf("unexpected result: %+v", result)
		}
	}
	for _, name := range []string{"OS", "code CLI", "g++", "GDB", "C++20 compile", "bits/stdc++.h", "C/C++", "CPH", "config", "root", "platform", "language"} {
		if status(t, results, name) != OK {
			t.Errorf("%s did not pass", name)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("configuration changed: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("doctor wrote to solution root: %v, %v", entries, err)
	}
}

func TestMissingCompilerContinues(t *testing.T) {
	c, _, _ := healthyChecker(t)
	c.lookup = func(name string) (string, error) {
		if name == "g++" {
			return "", errors.New("not found")
		}
		return name, nil
	}
	c.run = func(path string, args ...string) ([]byte, error) {
		if path != "gdb" || !reflect.DeepEqual(args, []string{"--version"}) {
			t.Fatalf("unexpected process after missing compiler: %s %v", path, args)
		}
		return []byte("GDB 1.0"), nil
	}
	results := c.check()
	for name, want := range map[string]string{"g++": Fail, "C++20 compile": Skip, "bits/stdc++.h": Skip, "GDB": OK, "CPH": OK, "root": OK} {
		if got := status(t, results, name); got != want {
			t.Errorf("%s = %s; want %s", name, got, want)
		}
	}
}

func TestCompilerProbesAndCleanup(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "success"
		if fail {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			var dirs []string
			var sources []string
			c := checker{run: func(path string, args ...string) ([]byte, error) {
				if path != "test-g++" || len(args) != 4 || args[0] != "-std=c++20" || args[2] != "-o" {
					t.Fatalf("unexpected compiler command: %q %q", path, args)
				}
				dir := filepath.Dir(args[1])
				if filepath.Clean(filepath.Dir(dir)) != filepath.Clean(os.TempDir()) {
					t.Fatalf("probe is outside OS temporary directory: %s", dir)
				}
				dirs = append(dirs, dir)
				data, err := os.ReadFile(args[1])
				if err != nil {
					t.Fatal(err)
				}
				sources = append(sources, string(data))
				if err := os.WriteFile(args[3], []byte("simulated compiler output"), 0600); err != nil {
					t.Fatal(err)
				}
				if fail {
					return []byte("compiler diagnostic"), errors.New("exit status 1")
				}
				return nil, nil
			}}
			results := c.compile("test-g++")
			if len(sources) != 2 || !strings.Contains(sources[0], "std::integral<int>") || !strings.Contains(sources[1], "#include <bits/stdc++.h>") {
				t.Fatalf("incorrect probes: %q", sources)
			}
			want := OK
			if fail {
				want = Fail
			}
			for _, result := range results {
				if result.Status != want || (fail && !strings.Contains(result.Detail, "compiler diagnostic")) {
					t.Errorf("unexpected compilation result: %+v", result)
				}
			}
			for _, dir := range dirs {
				if _, err := os.Stat(dir); !os.IsNotExist(err) {
					t.Errorf("temporary compiler files remain: %s, %v", dir, err)
				}
			}
		})
	}
}

func TestExtensionFailuresAndContinuation(t *testing.T) {
	for _, mode := range []string{"missing-cli", "missing-extension", "query-failure"} {
		t.Run(mode, func(t *testing.T) {
			c, _, _ := healthyChecker(t)
			switch mode {
			case "missing-cli":
				c.findEditor = func(string) (string, error) { return "", errors.New("missing code") }
				c.extensions = func(string, string) ([]byte, error) { t.Fatal("queried missing CLI"); return nil, nil }
			case "missing-extension":
				c.extensions = func(string, string) ([]byte, error) {
					return []byte("ms-vscode.cpptools\nother.competitive-programming-helper\n"), nil
				}
			case "query-failure":
				c.extensions = func(string, string) ([]byte, error) { return []byte("query failed"), errors.New("exit status 1") }
			}
			results := c.check()
			want := Skip
			if mode == "missing-extension" {
				want = Fail
				if status(t, results, "C/C++") != OK {
					t.Fatal("installed extension not detected")
				}
			}
			if status(t, results, "CPH") != want || status(t, results, "root") != OK || status(t, results, "C++20 compile") != OK {
				t.Fatalf("doctor did not continue: %+v", results)
			}
			if mode == "query-failure" && status(t, results, "extension query") != Fail {
				t.Fatal("query failure hidden")
			}
			if mode == "missing-cli" && status(t, results, "code CLI") != Fail {
				t.Fatal("missing CLI hidden")
			}
		})
	}
}

func TestConfigurationMissingRootAndConfig(t *testing.T) {
	for _, mode := range []string{"missing-root", "file-root", "missing-config", "malformed-config"} {
		t.Run(mode, func(t *testing.T) {
			c, path, root := healthyChecker(t)
			switch mode {
			case "missing-root", "file-root":
				if err := os.Remove(root); err != nil {
					t.Fatal(err)
				}
				if mode == "file-root" {
					if err := os.WriteFile(root, []byte("not a directory"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			case "missing-config":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "malformed-config":
				if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("CT_CONFIG_DIR", filepath.Dir(path))
			c.configPath = config.Path
			results := c.check()
			name := "config"
			if mode == "missing-root" {
				name = "root"
			}
			if status(t, results, name) != Fail || status(t, results, "CPH") != OK {
				t.Fatalf("unexpected checks: %+v", results)
			}
			if mode == "missing-root" {
				if _, err := os.Stat(root); !os.IsNotExist(err) {
					t.Fatal("missing root was created")
				}
			}
			if mode == "missing-config" {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatal("missing config was created")
				}
			}
		})
	}
}

func TestVersionIsInformationalAndMissingDebuggerFails(t *testing.T) {
	c, _, _ := healthyChecker(t)
	c.run = func(string, ...string) ([]byte, error) { return nil, errors.New("version failed") }
	if result := c.tool("GDB", "gdb", nil); result.Status != OK || !strings.Contains(result.Detail, "version unavailable") {
		t.Fatalf("version affected readiness: %+v", result)
	}
	if result := c.tool("GDB", "", errors.New("missing")); result.Status != Fail {
		t.Fatalf("missing debugger passed: %+v", result)
	}
}
