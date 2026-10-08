package environment

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GIRIBUIN/ct/internal/config"
	"github.com/GIRIBUIN/ct/internal/language"
)

func TestLanguageDiagnosisAndOverrideAreReadOnly(t *testing.T) {
	for _, selected := range []string{"java", "rust", "python"} {
		for _, override := range []bool{false, true} {
			c, path, root := healthyChecker(t)
			cfg := config.Defaults(root)
			cfg.EditorProfile = "language profile"
			if !override {
				cfg.Language = selected
			}
			if err := config.Update(path, cfg); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(path)
			c.lookup = func(name string) (string, error) {
				allowed := map[string][]string{"java": {"java", "javac"}, "rust": {"rustc"}, "python": {"python3"}}
				for _, candidate := range allowed[selected] {
					if name == candidate {
						return name, nil
					}
				}
				t.Fatalf("irrelevant language lookup: %s for %s", name, selected)
				return "", nil
			}
			var tempDirs []string
			c.run = func(exe string, args ...string) ([]byte, error) {
				switch exe {
				case "java":
					return []byte("openjdk version \"21.0.1\""), nil
				case "javac":
					if reflect.DeepEqual(args, []string{"-version"}) {
						return []byte("javac 21.0.1"), nil
					}
					if len(args) != 5 || !reflect.DeepEqual(args[:3], []string{"--release", "17", "-d"}) {
						t.Fatalf("javac args: %v", args)
					}
					tempDirs = append(tempDirs, args[3])
					if data, err := os.ReadFile(args[4]); err != nil || !bytes.Contains(data, []byte("class CTDoctor")) {
						t.Fatalf("Java source: %s %v", data, err)
					}
				case "rustc":
					if reflect.DeepEqual(args, []string{"--version"}) {
						return []byte("rustc 1.80.0"), nil
					}
					if len(args) != 4 || args[0] != "--edition=2021" || args[2] != "-o" {
						t.Fatalf("rustc args: %v", args)
					}
					tempDirs = append(tempDirs, filepath.Dir(args[1]))
					if data, err := os.ReadFile(args[1]); err != nil || !bytes.Contains(data, []byte("fn main()")) {
						t.Fatalf("Rust source: %s %v", data, err)
					}
				case "python3":
					return []byte("CT_PYTHON 3.12.0 /fake/python3"), nil
				default:
					t.Fatalf("unexpected process: %s %v", exe, args)
				}
				return nil, nil
			}
			c.extensions = func(_ string, profile string) ([]byte, error) {
				if profile != cfg.EditorProfile {
					t.Fatalf("profile changed: %q", profile)
				}
				var ids []string
				for _, ext := range language.Extensions(selected) {
					ids = append(ids, ext.ID)
				}
				return []byte(strings.Join(ids, "\n")), nil
			}
			arg := ""
			if override {
				arg = selected
			}
			results := c.check(arg)
			for _, check := range results {
				if check.Status != OK {
					t.Fatalf("%s: %+v", selected, results)
				}
				if check.Name == "selected" && check.Detail != selected {
					t.Fatalf("wrong selection: %+v", check)
				}
				if check.Section == "Configuration" && check.Name == "language" && check.Detail != cfg.Language {
					t.Fatalf("stored default changed by override: %+v", check)
				}
				if check.Name == "g++" || check.Name == "GDB" || check.Name == "C/C++" {
					t.Fatalf("irrelevant C++ requirement: %+v", check)
				}
			}
			if status(t, results, "CPH") != OK || status(t, results, "root") != OK {
				t.Fatal(results)
			}
			for _, dir := range tempDirs {
				if _, err := os.Stat(dir); !os.IsNotExist(err) {
					t.Fatalf("temporary probe remains: %s", dir)
				}
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(before, after) {
				t.Fatal("doctor rewrote config")
			}
		}
	}
}

func TestJavaOldMissingAndFailedCompiler(t *testing.T) {
	for _, mode := range []string{"old", "jre-only", "compile-failed", "version-failed"} {
		c, _, _ := healthyChecker(t)
		c.lookup = func(name string) (string, error) {
			if mode == "jre-only" && name == "javac" {
				return "", errors.New("missing")
			}
			return name, nil
		}
		c.run = func(exe string, args ...string) ([]byte, error) {
			if exe == "java" {
				return []byte("openjdk version \"21.0.1\""), nil
			}
			if len(args) == 1 {
				if mode == "old" {
					return []byte("javac 1.8.0_392"), nil
				}
				if mode == "version-failed" {
					return nil, errors.New("cannot execute javac")
				}
				return []byte("javac 17.0.2"), nil
			}
			if mode != "compile-failed" {
				t.Fatal("compiled with unusable javac")
			}
			return []byte("compile diagnostic"), errors.New("exit 1")
		}
		checks := c.javaChecks()
		if mode == "compile-failed" {
			if status(t, checks, "Java 17+ compile") != Fail {
				t.Fatal(checks)
			}
		} else if status(t, checks, "javac") != Fail || status(t, checks, "Java 17+ compile") != Skip {
			t.Fatal(checks)
		}
		if mode == "old" && !strings.Contains(checks[1].Detail, "17 or newer") {
			t.Fatal(checks)
		}
	}
}

func TestRustCompilationFailureAndMissingToolchain(t *testing.T) {
	c, _, _ := healthyChecker(t)
	c.run = func(_ string, args ...string) ([]byte, error) {
		if args[0] == "--version" {
			return []byte("rustc 1.80.0"), nil
		}
		return []byte("linker missing"), errors.New("exit 1")
	}
	if checks := c.rustChecks(); status(t, checks, "Rust 2021 compile") != Fail {
		t.Fatal(checks)
	}
	c.run = func(string, ...string) ([]byte, error) { return nil, errors.New("rustup has no default toolchain") }
	if checks := c.rustChecks(); status(t, checks, "rustc") != Fail || status(t, checks, "Rust 2021 compile") != Skip {
		t.Fatal(checks)
	}
}

func TestPythonSkipsUnusableCandidates(t *testing.T) {
	c, _, _ := healthyChecker(t)
	var calls []string
	c.run = func(exe string, args ...string) ([]byte, error) {
		calls = append(calls, exe)
		if exe == "py" && args[0] == "-3" {
			return []byte("CT_PYTHON 3.13.0 /python"), nil
		}
		return nil, errors.New("unusable alias")
	}
	if checks := c.pythonChecks(); status(t, checks, "Python interpreter") != OK || strings.Join(calls, ",") != "python3,python,py" {
		t.Fatal(checks, calls)
	}
	c.lookup = func(string) (string, error) { return "", errors.New("missing") }
	if checks := c.pythonChecks(); status(t, checks, "Python interpreter") != Fail {
		t.Fatal(checks)
	}
}
