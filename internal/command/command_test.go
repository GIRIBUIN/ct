package command

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GIRIBUIN/ct/internal/config"
	"github.com/GIRIBUIN/ct/internal/problem"
)

func TestParse(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want options
	}{
		{[]string{"71A"}, options{id: "71A"}},
		{[]string{"71A", "-l", "py"}, options{id: "71A", language: "python"}},
		{[]string{"71A", "--language", "python"}, options{id: "71A", language: "python"}},
		{[]string{"181188", "-p", "pg"}, options{id: "181188", platform: "programmers"}},
		{[]string{"181188", "--platform", "programmers"}, options{id: "181188", platform: "programmers"}},
		{[]string{"181188", "-p", "pg", "-l", "py"}, options{id: "181188", platform: "programmers", language: "python"}},
		{[]string{"-l", "c++", "71A", "--platform=cf"}, options{id: "71A", platform: "codeforces", language: "cpp"}},
		{[]string{"--help"}, options{help: true}},
	} {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			got, err := parse(tt.args)
			if err != nil || got != tt.want {
				t.Fatalf("parse = %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
	for _, args := range [][]string{nil, {"71A", "72A"}, {"71A", "-l"}, {"71A", "-p", "-l", "py"}, {"71A", "--unknown"}, {"71A", "-p", "bad"}, {"71A", "-l=java"}, {"71A", "-l="}} {
		if _, err := parse(args); err == nil {
			t.Errorf("invalid arguments accepted: %v", args)
		}
	}
}

func TestCreateTemplatesAndPreserveExisting(t *testing.T) {
	root := t.TempDir()
	for _, tt := range []struct{ platform, language, id, marker string }{
		{"codeforces", "cpp", "71A", "void solve()"},
		{"codeforces", "python", "71A", "def solve():"},
		{"programmers", "cpp", "181188", "int solution()"},
		{"programmers", "python", "181188", "def solution():"},
	} {
		t.Run(tt.platform+"/"+tt.language, func(t *testing.T) {
			target, err := problem.Target(root, tt.platform, tt.language, tt.id)
			if err != nil {
				t.Fatal(err)
			}
			created, err := create(target, tt.platform, tt.language)
			if err != nil || !created {
				t.Fatalf("create = %v, %v", created, err)
			}
			data, err := os.ReadFile(target)
			if err != nil || !bytes.Contains(data, []byte(tt.marker)) {
				t.Fatalf("incorrect template: %q, %v", data, err)
			}
			if tt.platform == "codeforces" && tt.language == "cpp" {
				for _, marker := range []string{"using ll = long long;", "int t = 1;", "while (t--)", "solve();"} {
					if !bytes.Contains(data, []byte(marker)) {
						t.Errorf("Codeforces C++ template is missing %q", marker)
					}
				}
			}
			if tt.platform == "programmers" && (bytes.Contains(data, []byte("main(")) || bytes.Contains(data, []byte("__main__")) || bytes.Contains(data, []byte("stdin"))) {
				t.Fatal("Programmers template contains main/stdin logic")
			}
			// User edits and adjacent files must survive a second invocation.
			original := []byte("my existing solution\n\x00\xff")
			if err := os.WriteFile(target, original, 0644); err != nil {
				t.Fatal(err)
			}
			notes := filepath.Join(filepath.Dir(target), "notes.txt")
			if err := os.WriteFile(notes, []byte("notes"), 0644); err != nil {
				t.Fatal(err)
			}
			created, err = create(target, tt.platform, tt.language)
			if err != nil || created {
				t.Fatalf("existing create = %v, %v", created, err)
			}
			data, err = os.ReadFile(target)
			if err != nil || !bytes.Equal(data, original) {
				t.Fatalf("existing solution changed: %q, %v", data, err)
			}
			if data, err := os.ReadFile(notes); err != nil || string(data) != "notes" {
				t.Fatalf("adjacent file changed: %q, %v", data, err)
			}
		})
	}
}

func TestRunFirstAndExistingSolution(t *testing.T) {
	for _, tt := range []struct{ name, id, platform, language, dir, file string }{
		{"defaults", "71A", "", "", "codeforces", "main.cpp"},
		{"overrides", "181188", "programmers", "python", "programmers", "solution.py"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "solutions")
			configPath := filepath.Join(base, "config", "config.json")
			target := filepath.Join(root, tt.dir, tt.id, tt.file)
			calls := 0
			open := func(editor, folder, file string) error {
				calls++
				if editor != "code" || folder != root || file != target {
					t.Fatalf("editor args: %q %q %q", editor, folder, file)
				}
				if _, err := os.Stat(file); err != nil {
					t.Fatal(err)
				}
				return nil
			}
			opts := options{id: tt.id, platform: tt.platform, language: tt.language}
			var output bytes.Buffer
			if err := run(opts, strings.NewReader(root+"\n"), &output, configPath, open); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "Created:") {
				t.Fatalf("missing creation message: %q", output.String())
			}
			cfg, err := config.Load(configPath)
			if err != nil || cfg != config.Defaults(root) {
				t.Fatalf("CLI overrides changed saved defaults: %+v, %v", cfg, err)
			}
			output.Reset()
			if err := run(opts, strings.NewReader(""), &output, configPath, open); err != nil {
				t.Fatal(err)
			}
			if calls != 2 || !strings.Contains(output.String(), "Already exists:") {
				t.Fatalf("existing file not reported/opened: calls=%d, output=%q", calls, output.String())
			}
		})
	}
}

func TestEditorFailurePreservesCreatedSolution(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "solutions")
	configPath := filepath.Join(base, "config.json")
	wantErr := errors.New("editor unavailable")
	err := run(options{id: "71A"}, strings.NewReader(root+"\n"), &bytes.Buffer{}, configPath, func(string, string, string) error { return wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("editor error lost: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "codeforces", "71A", "main.cpp")); err != nil {
		t.Fatalf("created solution lost: %v", err)
	}
}
