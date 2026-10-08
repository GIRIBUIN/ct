package installer

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GIRIBUIN/ct/internal/config"
	"github.com/GIRIBUIN/ct/internal/environment"
	"github.com/GIRIBUIN/ct/internal/language"
)

func checksWithFailures(names ...string) []environment.Check {
	var checks []environment.Check
	for _, name := range []string{"g++", "GDB", "code CLI", "C++20 compile", "bits/stdc++.h", "C/C++", "CPH", "config", "root", "platform", "language"} {
		check := environment.Check{Name: name, Status: environment.OK, Detail: name}
		for _, failed := range names {
			if name == failed {
				check.Status = environment.Fail
			}
		}
		checks = append(checks, check)
	}
	return checks
}

func fakeHost(t *testing.T) host {
	t.Helper()
	return host{
		goos: "linux", arch: "amd64",
		lookup:           func(name string) (string, error) { return name, nil },
		getenv:           func(string) string { return "" },
		setenv:           func(string, string) error { t.Fatal("unexpected environment mutation"); return nil },
		readFile:         func(string) ([]byte, error) { return []byte("ID=ubuntu\nID_LIKE=debian\n"), nil },
		exists:           func(string) bool { return false },
		query:            func(invocation) ([]byte, error) { return []byte("[]"), nil },
		run:              func(invocation) error { t.Fatal("mutation during planning"); return nil },
		installExtension: func(string, string, string) error { t.Fatal("extension mutation during planning"); return nil },
		loadConfig:       func() (config.Config, error) { return config.Defaults("test-root"), nil },
		uid:              func() int { return 1000 },
	}
}

func TestReadyPlansNothing(t *testing.T) {
	h := fakeHost(t)
	h.lookup = func(string) (string, error) { t.Fatal("unnecessary tool discovery"); return "", nil }
	plan, err := h.plan(checksWithFailures())
	if err != nil || len(plan.Actions) != 0 || len(plan.Notes) != 0 {
		t.Fatalf("unexpected plan: %+v, %v", plan, err)
	}
}

func TestOnlyMissingExtensionPlannedAndProfileUsed(t *testing.T) {
	for _, extension := range language.Extensions("cpp") {
		for _, profile := range []string{"", "my coding profile"} {
			t.Run(extension.Name+profile, func(t *testing.T) {
				h := fakeHost(t)
				h.loadConfig = func() (config.Config, error) {
					cfg := config.Defaults("test-root")
					cfg.EditorProfile = profile
					return cfg, nil
				}
				called := false
				h.installExtension = func(path, selected, id string) error {
					called = true
					if path != "code CLI" || selected != profile || id != extension.ID {
						t.Fatalf("wrong extension target: %q %q %q", path, selected, id)
					}
					return nil
				}
				plan, err := h.plan(checksWithFailures(extension.Name))
				if err != nil || len(plan.Actions) != 1 || called {
					t.Fatalf("plan mutated or incorrect: %+v, %v", plan, err)
				}
				if strings.Contains(strings.Join(plan.Actions[0].Details, " "), "--profile") != (profile != "") {
					t.Fatal("plan profile mismatch")
				}
				if err := plan.Actions[0].Execute(); err != nil || !called {
					t.Fatalf("execution failed: %v", err)
				}
			})
		}
	}
}

func TestUnknownExtensionsNotInstalled(t *testing.T) {
	h := fakeHost(t)
	checks := checksWithFailures("C/C++", "CPH")
	checks = append(checks, environment.Check{Name: "extension query", Status: environment.Fail})
	plan, err := h.plan(checks)
	if err != nil || len(plan.Actions) != 0 || !strings.Contains(strings.Join(plan.Notes, " "), "unknown") {
		t.Fatalf("unknown state treated as missing: %+v, %v", plan, err)
	}
}

func TestLinuxInstallPlan(t *testing.T) {
	for _, missing := range []string{"g++", "GDB"} {
		h := fakeHost(t)
		var calls []invocation
		h.run = func(c invocation) error { calls = append(calls, c); return nil }
		plan, err := h.plan(checksWithFailures(missing))
		if err != nil || len(plan.Actions) != 2 || len(calls) != 0 {
			t.Fatalf("plan = %+v, %v", plan, err)
		}
		if !reflect.DeepEqual(plan.Actions[1].DependsOn, []string{"apt-update"}) {
			t.Fatal("apt dependency missing")
		}
		for _, action := range plan.Actions {
			if err := action.Execute(); err != nil {
				t.Fatal(err)
			}
		}
		pkg := "gdb"
		if missing == "g++" {
			pkg = "build-essential"
		}
		if calls[0].path != "sudo" || !reflect.DeepEqual(calls[0].args, []string{"apt-get", "update"}) || !reflect.DeepEqual(calls[1].args, []string{"apt-get", "install", "--yes", "--no-upgrade", pkg}) {
			t.Fatalf("unexpected Linux commands: %+v", calls)
		}
	}
}

func TestLinuxUnsupported(t *testing.T) {
	h := fakeHost(t)
	h.readFile = func(string) ([]byte, error) { return []byte("ID=fedora\n"), nil }
	h.lookup = func(string) (string, error) { t.Fatal("guessed package manager"); return "", nil }
	plan, err := h.plan(checksWithFailures("g++", "GDB"))
	if err != nil || len(plan.Actions) != 0 || !strings.Contains(strings.Join(plan.Notes, " "), "Debian/Ubuntu only") {
		t.Fatalf("unsupported distro plan: %+v, %v", plan, err)
	}
}

func TestWindowsPathAppend(t *testing.T) {
	for _, tt := range []struct {
		current, entry, want string
		changed              bool
	}{
		{`C:\one;C:\two`, `C:\tool chain\bin`, `C:\one;C:\two;C:\tool chain\bin`, true},
		{`C:\one;C:\two;`, `C:\tool\bin`, `C:\one;C:\two;C:\tool\bin`, true},
		{`C:\one;C:\TOOL\bin\`, `c:/tool/bin`, `C:\one;C:\TOOL\bin\`, false},
		{`"C:\tool chain\bin";C:\other`, `c:\tool chain\bin`, `"C:\tool chain\bin";C:\other`, false},
		{`C:\one`, `"C:\tool chain\bin"`, `C:\one;C:\tool chain\bin`, true},
		{"", `C:\tool\bin`, `C:\tool\bin`, true},
	} {
		got, changed, err := AppendWindowsPath(tt.current, tt.entry)
		if err != nil || got != tt.want || changed != tt.changed {
			t.Errorf("append(%q,%q) = %q,%v,%v; want %q,%v", tt.current, tt.entry, got, changed, err, tt.want, tt.changed)
		}
	}
	for _, entry := range []string{"", "a;b", "a\"b", "a\nb"} {
		if _, _, err := AppendWindowsPath("original", entry); err == nil {
			t.Errorf("unsafe PATH entry accepted: %q", entry)
		}
	}
}

func TestWindowsExistingMSYSAndPathMutation(t *testing.T) {
	h := fakeHost(t)
	h.goos = "windows"
	root := filepath.Join(t.TempDir(), "custom-msys")
	bin := filepath.Join(root, "ucrt64", "bin")
	processPath := "existing-process-path"
	h.getenv = func(name string) string {
		if name == "MSYS2_ROOT" {
			return root
		}
		if name == "PATH" {
			return processPath
		}
		return ""
	}
	h.exists = func(p string) bool {
		return p == filepath.Join(root, "usr", "bin", "pacman.exe") || p == filepath.Join(root, "usr", "bin", "bash.exe")
	}
	queries := 0
	h.query = func(c invocation) ([]byte, error) {
		if !strings.Contains(strings.Join(c.args, " "), readUserPath) {
			t.Fatal("registry/winget used despite existing MSYS2")
		}
		queries++
		if queries == 1 {
			return []byte("old-user-path"), nil
		}
		return []byte("old-user-path;concurrent-addition"), nil
	}
	var calls []invocation
	h.run = func(c invocation) error { calls = append(calls, c); return nil }
	h.setenv = func(name, value string) error {
		if name != "PATH" {
			t.Fatal(name)
		}
		processPath = value
		return nil
	}
	plan, err := h.plan(checksWithFailures("g++", "GDB"))
	if err != nil || len(plan.Actions) != 2 || len(calls) != 0 {
		t.Fatalf("Windows plan: %+v, %v", plan, err)
	}
	for _, action := range plan.Actions {
		if err := action.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	if len(calls) != 2 || !strings.Contains(strings.Join(calls[0].args, " "), "mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-gdb") {
		t.Fatalf("unexpected MSYS commands: %+v", calls)
	}
	if !reflect.DeepEqual(calls[1].env, []string{"CT_SETUP_USER_PATH=old-user-path;concurrent-addition;" + bin}) {
		t.Fatalf("User PATH overwritten: %+v", calls[1])
	}
	if !strings.Contains(strings.Join(calls[1].args, " "), "'User'") || strings.Contains(strings.Join(calls[1].args, " "), "'Machine'") {
		t.Fatal("wrong PATH scope")
	}
	if processPath != "existing-process-path;"+bin {
		t.Fatalf("process PATH not refreshed: %s", processPath)
	}
}

func TestWindowsAlreadyInstalledToolsOnlyAddPath(t *testing.T) {
	h := fakeHost(t)
	h.goos = "windows"
	root := filepath.Join(t.TempDir(), "msys")
	h.getenv = func(name string) string {
		if name == "MSYS2_ROOT" {
			return root
		}
		return ""
	}
	h.exists = func(p string) bool { return strings.HasPrefix(p, root+string(filepath.Separator)) }
	h.query = func(invocation) ([]byte, error) { return nil, nil }
	plan, err := h.plan(checksWithFailures("g++", "GDB"))
	if err != nil || len(plan.Actions) != 1 || plan.Actions[0].ID != "ucrt-path" {
		t.Fatalf("reinstallation planned: %+v, %v", plan, err)
	}
}

func TestWindowsWingetAndRegisteredInstallation(t *testing.T) {
	for _, registered := range []bool{false, true} {
		h := fakeHost(t)
		h.goos = "windows"
		base := t.TempDir()
		h.getenv = func(name string) string {
			if name == "LOCALAPPDATA" {
				return base
			}
			return ""
		}
		h.query = func(c invocation) ([]byte, error) {
			if strings.Contains(strings.Join(c.args, " "), msysRegistry) {
				if registered {
					return []byte(`["broken-installation"]`), nil
				}
				return []byte(`[]`), nil
			}
			return nil, nil
		}
		plan, err := h.plan(checksWithFailures("g++"))
		if err != nil {
			t.Fatal(err)
		}
		if registered {
			if len(plan.Actions) != 0 || !strings.Contains(strings.Join(plan.Notes, " "), "second copy") {
				t.Fatalf("second install allowed: %+v", plan)
			}
		} else if len(plan.Actions) != 3 || plan.Actions[0].ID != "msys2" || !strings.Contains(strings.Join(plan.Actions[0].Details, " "), "--scope") || !reflect.DeepEqual(plan.Actions[1].DependsOn, []string{"msys2"}) {
			t.Fatalf("winget plan: %+v", plan)
		}
	}
}

func TestNoToolchainReplacementForCapabilityFailure(t *testing.T) {
	h := fakeHost(t)
	plan, err := h.plan(checksWithFailures("C++20 compile"))
	if err != nil || len(plan.Actions) != 0 || !strings.Contains(strings.Join(plan.Notes, " "), "compatible GCC") {
		t.Fatalf("unexpected replacement: %+v, %v", plan, err)
	}
}

func TestLinuxPackageFailurePropagates(t *testing.T) {
	h := fakeHost(t)
	want := errors.New("package manager failed")
	h.run = func(invocation) error { return want }
	plan, err := h.plan(checksWithFailures("GDB"))
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Actions[1].Execute(); !errors.Is(err, want) {
		t.Fatalf("lost failure: %v", err)
	}
}
