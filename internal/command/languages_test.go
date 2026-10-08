package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GIRIBUIN/ct/internal/problem"
)

func TestLanguageCommandFlags(t *testing.T) {
	for _, command := range []string{"doctor", "setup"} {
		for _, alias := range []string{"cpp", "py", "java", "rust", "rs"} {
			args := []string{command, "--language=" + alias}
			if command == "setup" {
				args = []string{command, "--dry-run", "-l", alias, "--yes"}
			}
			got, err := parse(args)
			want, _ := problem.NormalizeLanguage(alias)
			if err != nil || got.language != want || (command == "setup" && (!got.dryRun || !got.yes)) {
				t.Fatalf("%v -> %+v %v", args, got, err)
			}
		}
	}
	for _, args := range [][]string{{"doctor", "-l"}, {"setup", "--language=kotlin"}, {"setup", "-l", "--yes"}} {
		if _, err := parse(args); err == nil {
			t.Fatal(args)
		}
	}
}

func TestAllLanguagesCoexistAndRemainUnchanged(t *testing.T) {
	root := t.TempDir()
	for platform, id := range map[string]string{"cf": "71A", "pg": "181188"} {
		var paths []string
		for _, language := range []string{"cpp", "python", "java", "rust"} {
			path, err := problem.Target(root, platform, language, id)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := create(path, platform, language); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("user "+language), 0600); err != nil {
				t.Fatal(err)
			}
			paths = append(paths, path)
		}
		for i, language := range []string{"cpp", "python", "java", "rust"} {
			if created, err := create(paths[i], platform, language); err != nil || created {
				t.Fatal(created, err)
			}
			data, err := os.ReadFile(paths[i])
			if err != nil || !bytes.Equal(data, []byte("user "+language)) {
				t.Fatal(string(data), err)
			}
		}
		entries, _ := os.ReadDir(filepath.Dir(paths[0]))
		if len(entries) != 4 {
			t.Fatal("languages did not coexist")
		}
	}
	var help bytes.Buffer
	if err := Run([]string{"--help"}, strings.NewReader(""), &help); err != nil || !strings.Contains(help.String(), "java, rust (rs)") {
		t.Fatal(help.String(), err)
	}
}
