package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GIRIBUIN/ct/internal/config"
	"github.com/GIRIBUIN/ct/internal/environment"
	"github.com/GIRIBUIN/ct/internal/installer"
	"github.com/GIRIBUIN/ct/internal/registry"
)

func registryCLI(t *testing.T, input string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(args, strings.NewReader(input), &out)
	return out.String(), err
}

func customWorkspace(t *testing.T) (string, string) {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "config")
	root := filepath.Join(base, "solutions")
	t.Setenv("CT_CONFIG_DIR", dir)
	if err := config.Save(filepath.Join(dir, "config.json"), config.Defaults(root)); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(base, "source.kt")
	if err := os.WriteFile(source, []byte("fun main() { println(42) }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	input := "Kotlin\nKT\nkt\nMain.kt\n" + source + "\nSolution.kt\n\n"
	if out, err := registryCLI(t, input, "language", "add"); err != nil {
		t.Fatal(out, err)
	}
	// cpp, python, java, rust, kotlin: each filename followed by template source.
	input = "Baekjoon\nBOJ\n\n\n\n\n\n\n\n\nMain.kt\n" + source + "\n"
	if out, err := registryCLI(t, input, "platform", "add"); err != nil {
		t.Fatal(out, err)
	}
	return dir, root
}

func TestRegistryListsShowsAndLifecycle(t *testing.T) {
	dir, root := customWorkspace(t)
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"language", "list"}, []string{"cpp", "kotlin", "kt", "user", "none"}},
		{[]string{"language", "show", "kotlin"}, []string{"Name: kotlin", "Main.kt", "codeforces.tmpl"}},
		{[]string{"language", "show", "kt"}, []string{"Name: kotlin", "Environment: none"}},
		{[]string{"language", "show", "rs"}, []string{"Name: rust", "rust-analyzer", "embedded"}},
		{[]string{"platform", "list"}, []string{"codeforces", "baekjoon", "boj", "user"}},
		{[]string{"platform", "show", "boj"}, []string{"Name: baekjoon", "Main.java", "Main.kt"}},
	} {
		out, err := registryCLI(t, "", tc.args...)
		if err != nil {
			t.Fatal(out, err)
		}
		for _, want := range tc.want {
			if !strings.Contains(out, want) {
				t.Fatalf("missing %q: %s", want, out)
			}
		}
	}
	for _, tc := range []struct{ kind, alias string }{{"language", "kt"}, {"platform", "boj"}, {"language", "py"}, {"platform", "pg"}} {
		if _, err := registryCLI(t, "", tc.kind, "disable", tc.alias); err != nil {
			t.Fatal(err)
		}
		out, err := registryCLI(t, "", tc.kind, "list", "--all")
		if err != nil || !strings.Contains(out, "disabled") {
			t.Fatal(out, err)
		}
		if _, err := registryCLI(t, "", tc.kind, "enable", tc.alias); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct{ kind, name string }{{"language", "cpp"}, {"platform", "cf"}} {
		if _, err := registryCLI(t, "y\n", tc.kind, "remove", tc.name); err == nil {
			t.Fatal("removed built-in")
		}
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	solution := filepath.Join(root, "keep.kt")
	os.WriteFile(solution, []byte("solution"), 0600)
	for _, tc := range []struct{ kind, alias string }{{"language", "kt"}, {"platform", "boj"}} {
		if out, err := registryCLI(t, "\n", tc.kind, "remove", tc.alias); err != nil || !strings.Contains(out, "cancelled") {
			t.Fatal(out, err)
		}
		if _, err := registryCLI(t, "", tc.kind, "show", tc.alias); err != nil {
			t.Fatal(err)
		}
		if _, err := registryCLI(t, "y\n", tc.kind, "remove", tc.alias); err != nil {
			t.Fatal(err)
		}
		if _, err := registryCLI(t, "", tc.kind, "show", tc.alias); err == nil {
			t.Fatal("removed entry still visible")
		}
	}
	if data, err := os.ReadFile(solution); err != nil || string(data) != "solution" {
		t.Fatal(err)
	}
	if _, err := registry.Load(dir); err != nil {
		t.Fatal(err)
	}
}

func TestCustomProblemBindingsAndOverwrite(t *testing.T) {
	dir, root := customWorkspace(t)
	for _, tc := range []struct{ p, l, id, file, content string }{
		{"boj", "cpp", "1000", "main.cpp", ""},
		{"cf", "kt", "71A", "Main.kt", "fun main() { println(42) }\n"},
		{"boj", "kt", "1000", "Main.kt", "fun main() { println(42) }\n"},
	} {
		var out bytes.Buffer
		wantPlatform := "codeforces"
		if tc.p == "boj" {
			wantPlatform = "baekjoon"
		}
		want := filepath.Join(root, wantPlatform, tc.id, tc.file)
		open := func(_ string, gotRoot, target, _ string) error {
			if gotRoot != root || target != want {
				t.Fatalf("wrong target %s", target)
			}
			return nil
		}
		opts := options{id: tc.id, platform: tc.p, language: tc.l}
		if err := run(opts, strings.NewReader(""), &out, filepath.Join(dir, "config.json"), open); err != nil {
			t.Fatal(err)
		}
		if data, err := os.ReadFile(want); err != nil || string(data) != tc.content {
			t.Fatal(string(data), err)
		}
		os.WriteFile(want, []byte("edited solution"), 0600)
		if err := run(opts, strings.NewReader(""), &out, filepath.Join(dir, "config.json"), open); err != nil {
			t.Fatal(err)
		}
		if data, _ := os.ReadFile(want); string(data) != "edited solution" {
			t.Fatal("overwritten")
		}
	}
	for _, tc := range []struct{ kind, name string }{{"language", "kt"}, {"platform", "boj"}} {
		if _, err := registryCLI(t, "", tc.kind, "disable", tc.name); err != nil {
			t.Fatal(err)
		}
		if _, err := registryCLI(t, "", "1001", "-p", "boj", "-l", "kt"); err == nil || !strings.Contains(err.Error(), "disabled") {
			t.Fatal(err)
		}
		if _, err := registryCLI(t, "", tc.kind, "enable", tc.name); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRegistryConfigDefaultsAndRepair(t *testing.T) {
	dir, root := customWorkspace(t)
	path := filepath.Join(dir, "config.json")
	var out bytes.Buffer
	noEditor := func(string, string) error { t.Fatal("unexpected editor query"); return nil }
	cfg, err := configure(path, strings.NewReader("\nboj\nkt\n\n"), &out, noEditor, saveConfiguration)
	if err != nil || cfg.Platform != "baekjoon" || cfg.Language != "kotlin" {
		t.Fatal(cfg, err)
	}
	before, _ := os.ReadFile(path)
	if _, err := registryCLI(t, "", "language", "disable", "kt"); err != nil {
		t.Fatal(err)
	}
	if err := config.Update(path, cfg); err == nil {
		t.Fatal("saved disabled default")
	}
	err = run(options{id: "1000"}, strings.NewReader(""), &out, path, func(string, string, string, string) error { t.Fatal("opened disabled default"); return nil })
	if err == nil || !strings.Contains(err.Error(), "enable kotlin") {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("disabled default rewritten")
	}
	// A disabled current default is shown and must be explicitly repaired.
	out.Reset()
	cfg, err = configure(path, strings.NewReader("\n\n\nunknown\npy\n\n"), &out, noEditor, saveConfiguration)
	if err != nil || cfg.Language != "python" || cfg.Root != root || !strings.Contains(out.String(), "Current default:") || !strings.Contains(out.String(), "enabled entries:") {
		t.Fatal(cfg, err, out.String())
	}
}

func TestCustomLanguageDoctorSummaryAndSetup(t *testing.T) {
	checks := []environment.Check{
		{Section: "Language", Name: "selected", Status: environment.OK, Detail: "kotlin"},
		{Section: "Environment", Name: "provider", Status: environment.Skip, Detail: "no built-in environment provider"},
	}
	var out bytes.Buffer
	if err := reportDoctor(&out, checks); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Environment ready.") || !strings.Contains(out.String(), "diagnostics are unavailable") {
		t.Fatal(out.String())
	}
	for _, dry := range []bool{false, true} {
		out.Reset()
		err := runSetup(options{dryRun: dry}, strings.NewReader(""), &out, func() []environment.Check { return checks }, func([]environment.Check) (installer.Plan, error) {
			t.Fatal("custom setup planned installation")
			return installer.Plan{}, nil
		})
		if err != nil || !strings.Contains(out.String(), `Automatic setup is not available for user language "kotlin".`) {
			t.Fatal(out.String(), err)
		}
	}
}
