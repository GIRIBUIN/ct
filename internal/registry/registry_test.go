package registry

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testRegistry(t *testing.T) *Registry {
	t.Helper()
	r, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestMissingAndRoundTrip(t *testing.T) {
	r := testRegistry(t)
	if len(r.Entries("language")) != 4 || len(r.Entries("platform")) != 2 {
		t.Fatal("built-ins missing")
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "registry.json")); !os.IsNotExist(err) {
		t.Fatal("read created registry")
	}
	if err := r.Add("language", Entry{"Kotlin", []string{"KT"}, "KT"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.Add("platform", Entry{Name: "Baekjoon", Aliases: []string{"BOJ"}}, nil); err != nil {
		t.Fatal(err)
	}
	b := Binding{Platform: "baekjoon", Language: "kotlin", Filename: "Main.kt", Template: TemplatePath("platform", "baekjoon", "kotlin")}
	r.Data.Bindings = append(r.Data.Bindings, b)
	if err := r.SetEnabled("language", "py", false); err != nil {
		t.Fatal(err)
	}
	if err := r.SaveTemplates(map[string][]byte{b.Template: []byte("fun main() {}\n")}); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	for kind, alias := range map[string]string{"language": "KT", "platform": "BOJ"} {
		d, err := loaded.Lookup(kind, alias, true)
		if err != nil || d.Builtin || !d.Enabled {
			t.Fatal(d, err)
		}
	}
	got, err := loaded.Binding("boj", "kt")
	if err != nil || got != b {
		t.Fatal(got, err)
	}
	data, err := loaded.ReadTemplate(got)
	if err != nil || string(data) != "fun main() {}\n" {
		t.Fatal(string(data), err)
	}
	if _, err := loaded.Lookup("language", "python", true); err == nil {
		t.Fatal("disabled accepted")
	}
	if err := loaded.SetEnabled("language", "py", true); err != nil {
		t.Fatal(err)
	}
	if err := loaded.Save(); err != nil {
		t.Fatal(err)
	}
}

func TestCollisionsAndInvalidNames(t *testing.T) {
	for _, tc := range []struct {
		kind  string
		entry Entry
	}{
		{"language", Entry{Name: "cpp", Extension: "x"}},
		{"language", Entry{Name: "kotlin", Aliases: []string{"py"}, Extension: "kt"}},
		{"language", Entry{Name: "kotlin", Aliases: []string{"rs"}, Extension: "kt"}},
		{"language", Entry{Name: "kotlin", Aliases: []string{"x", "x"}, Extension: "kt"}},
		{"language", Entry{Name: "kotlin", Aliases: []string{" "}, Extension: "kt"}},
		{"platform", Entry{Name: "codeforces"}},
		{"platform", Entry{Name: "baekjoon", Aliases: []string{"cf"}}},
		{"platform", Entry{Name: " "}},
		{"platform", Entry{Name: "../outside"}},
		{"platform", Entry{Name: "con"}},
	} {
		r := testRegistry(t)
		if err := r.Add(tc.kind, tc.entry, nil); err == nil {
			t.Fatalf("accepted %+v", tc)
		}
	}
	r := testRegistry(t)
	if err := r.Add("language", Entry{Name: "kotlin", Aliases: []string{"kt"}, Extension: "kt"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.Add("language", Entry{Name: "other", Aliases: []string{"kt"}, Extension: "x"}, nil); err == nil {
		t.Fatal("user alias shadowed")
	}
	if err := r.Add("platform", Entry{Name: "baekjoon", Aliases: []string{"boj"}}, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.Add("platform", Entry{Name: "other", Aliases: []string{"boj"}}, nil); err == nil {
		t.Fatal("platform alias shadowed")
	}
}

func TestAtomicUpdatePreservesPreviousRegistry(t *testing.T) {
	r := testRegistry(t)
	if err := r.Save(); err != nil {
		t.Fatal(err)
	}
	stale, err := Load(r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Add("language", Entry{Name: "kotlin", Extension: "kt"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.Save(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(r.Dir, "registry.json")
	before, _ := os.ReadFile(path)
	if err := stale.Save(); err == nil {
		t.Fatal("stale registry replaced valid data")
	}
	r.Data.Bindings = []Binding{{Platform: "codeforces", Language: "kotlin", Filename: "../solution.kt"}}
	if err := r.Save(); err == nil {
		t.Fatal("invalid update saved")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("valid registry destroyed")
	}
	files, _ := filepath.Glob(filepath.Join(r.Dir, ".ct-registry-*.tmp"))
	if len(files) != 0 {
		t.Fatal(files)
	}
}

func TestCorruptRegistryAndUnsafeTemplates(t *testing.T) {
	for _, content := range []string{"broken", "null", "{}", `{"version":99}`, `{"version":1,"run":"arbitrary command"}`, `{"version":1} {}`} {
		r := testRegistry(t)
		path := filepath.Join(r.Dir, "registry.json")
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(r.Dir); err == nil || !strings.Contains(err.Error(), "repair") {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(path)
		if string(data) != content {
			t.Fatal("corrupt data overwritten")
		}
	}
	for _, path := range []string{"../outside.tmpl", "/tmp/outside.tmpl", `C:\outside.tmpl`, "templates/languages/other/codeforces.tmpl"} {
		r := testRegistry(t)
		if err := r.Add("language", Entry{Name: "kotlin", Extension: "kt"}, nil); err != nil {
			t.Fatal(err)
		}
		r.Data.Bindings = []Binding{{Platform: "codeforces", Language: "kotlin", Filename: "main.kt", Template: path}}
		if err := r.Save(); err == nil {
			t.Fatalf("accepted %s", path)
		}
		if err := r.Remove("language", "kotlin"); err == nil {
			t.Fatalf("unsafe removal accepted %s", path)
		}
	}
}

func TestRemovalOnlyOwnedTemplates(t *testing.T) {
	r := testRegistry(t)
	if err := r.Add("language", Entry{Name: "kotlin", Extension: "kt"}, nil); err != nil {
		t.Fatal(err)
	}
	b := Binding{Platform: "codeforces", Language: "kotlin", Filename: "main.kt", Template: TemplatePath("language", "kotlin", "codeforces")}
	r.Data.Bindings = []Binding{b}
	if err := r.SaveTemplates(map[string][]byte{b.Template: {}}); err != nil {
		t.Fatal(err)
	}
	owned := filepath.Join(r.Dir, filepath.FromSlash(b.Template))
	unowned := filepath.Join(filepath.Dir(owned), "notes.txt")
	os.WriteFile(unowned, []byte("keep"), 0600)
	for _, kind := range []string{"language", "platform"} {
		name := "cpp"
		if kind == "platform" {
			name = "codeforces"
		}
		if err := r.Remove(kind, name); err == nil || !strings.Contains(err.Error(), "disable") {
			t.Fatal(err)
		}
	}
	if err := r.Remove("language", "kotlin"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(owned); !os.IsNotExist(err) {
		t.Fatal("owned template remains", err)
	}
	if data, err := os.ReadFile(unowned); err != nil || string(data) != "keep" {
		t.Fatal("unowned file removed", err)
	}
}

func TestLinkedTemplateRefused(t *testing.T) {
	r := testRegistry(t)
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(r.Dir, "templates", "languages"), 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(r.Dir, "templates", "languages", "kotlin")
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	b := Binding{Platform: "codeforces", Language: "kotlin", Filename: "main.kt", Template: TemplatePath("language", "kotlin", "codeforces")}
	if err := r.Add("language", Entry{Name: "kotlin", Extension: "kt"}, []Binding{b}); err == nil {
		t.Fatal("linked template directory accepted")
	}
}
