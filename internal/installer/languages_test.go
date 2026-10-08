package installer

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/GIRIBUIN/ct/internal/config"
	"github.com/GIRIBUIN/ct/internal/environment"
	"github.com/GIRIBUIN/ct/internal/language"
)

func selectedChecks(selected string, failures ...string) []environment.Check {
	checks := checksWithFailures()
	checks = append(checks, environment.Check{Section: "Language", Name: "selected", Status: environment.OK, Detail: selected})
	for _, ext := range language.Extensions(selected) {
		checks = append(checks, environment.Check{Name: ext.Name, Status: environment.OK})
	}
	for _, name := range []string{"java", "javac", "Java 17+ compile", "rustc", "Rust 2021 compile", "Python interpreter"} {
		checks = append(checks, environment.Check{Name: name, Status: environment.OK})
	}
	for i := range checks {
		for _, name := range failures {
			if checks[i].Name == name {
				checks[i].Status = environment.Fail
			}
		}
	}
	return checks
}

func TestOnlySelectedLanguageExtensionIsInstalled(t *testing.T) {
	for _, selected := range []string{"java", "rust", "python"} {
		for _, profile := range []string{"", "my language profile"} {
			h := fakeHost(t)
			h.loadConfig = func() (config.Config, error) {
				cfg := config.Defaults("test-root")
				cfg.EditorProfile = profile
				return cfg, nil // Stored cpp must not override selected doctor language.
			}
			extension := language.Extensions(selected)[0]
			plan, err := h.plan(selectedChecks(selected, extension.Name, "g++", "GDB", "C/C++"))
			if err != nil || len(plan.Actions) != 1 || plan.Actions[0].ID != "extension:"+extension.ID {
				t.Fatalf("irrelevant actions: %+v %v", plan, err)
			}
			called := false
			h.installExtension = func(path, actualProfile, id string) error {
				called = true
				if path != "code CLI" || actualProfile != profile || id != extension.ID {
					t.Fatal(path, actualProfile, id)
				}
				return nil
			}
			plan, _ = h.plan(selectedChecks(selected, extension.Name))
			if err := plan.Actions[0].Execute(); err != nil || !called {
				t.Fatal(err)
			}
		}
	}
}

func TestJavaInstallationPlans(t *testing.T) {
	for _, platform := range []string{"windows", "linux"} {
		h := fakeHost(t)
		h.goos = platform
		var calls []invocation
		h.run = func(c invocation) error { calls = append(calls, c); return nil }
		h.getenv = func(string) string { return "old-process-path" }
		h.query = func(c invocation) ([]byte, error) {
			if len(calls) != 1 || !strings.Contains(strings.Join(c.args, " "), "GetEnvironmentVariable") {
				t.Fatal("refresh before install or unexpected mutation")
			}
			return []byte("new-jdk-bin"), nil
		}
		refreshed := false
		h.setenv = func(name, value string) error {
			if name != "PATH" || value != "new-jdk-bin;old-process-path" {
				t.Fatal(name, value)
			}
			refreshed = true
			return nil
		}
		plan, err := h.plan(selectedChecks("java", "javac"))
		if err != nil || len(plan.Actions) != 2 || len(calls) != 0 || refreshed {
			t.Fatalf("bad plan: %+v %v", plan, err)
		}
		for _, action := range plan.Actions {
			if err := action.Execute(); err != nil {
				t.Fatal(err)
			}
		}
		if platform == "windows" {
			if len(calls) != 1 || !strings.Contains(strings.Join(calls[0].args, " "), "Microsoft.OpenJDK.21") || !refreshed || !reflect.DeepEqual(plan.Actions[1].DependsOn, []string{"java-jdk"}) {
				t.Fatal(calls, plan)
			}
		} else if len(calls) != 2 || calls[1].path != "sudo" || !reflect.DeepEqual(calls[1].args, []string{"apt-get", "install", "--yes", "--no-upgrade", "openjdk-21-jdk"}) {
			t.Fatal(calls)
		}
		ready, err := h.plan(selectedChecks("java"))
		if err != nil || len(ready.Actions) != 0 {
			t.Fatal(ready, err)
		}
	}
}

func TestManualRuntimePoliciesAndUnsupportedJDK(t *testing.T) {
	for _, tt := range []struct{ language, failed, marker string }{
		{"rust", "rustc", "https://rustup.rs/"}, {"python", "Python interpreter", "Python runtime installation is not automated"},
	} {
		h := fakeHost(t)
		h.lookup = func(string) (string, error) { t.Fatal("automatic runtime installation attempted"); return "", nil }
		plan, err := h.plan(selectedChecks(tt.language, tt.failed))
		if err != nil || len(plan.Actions) != 0 || !strings.Contains(strings.Join(plan.Notes, " "), tt.marker) {
			t.Fatal(plan, err)
		}
	}
	h := fakeHost(t)
	h.readFile = func(string) ([]byte, error) { return []byte("ID=fedora"), nil }
	plan, err := h.plan(selectedChecks("java", "javac"))
	if err != nil || len(plan.Actions) != 0 || !strings.Contains(strings.Join(plan.Notes, " "), "Debian/Ubuntu only") {
		t.Fatal(plan, err)
	}
	h.goos = "windows"
	h.lookup = func(string) (string, error) { return "", errors.New("missing winget") }
	plan, err = h.plan(selectedChecks("java", "javac"))
	if err != nil || len(plan.Actions) != 0 || !strings.Contains(strings.Join(plan.Notes, " "), "winget is unavailable") {
		t.Fatal(plan, err)
	}
}
