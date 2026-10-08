package command

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GIRIBUIN/ct/internal/registry"
)

func TestRegistryAddFieldRetries(t *testing.T) {
	for _, kind := range []string{"language", "platform"} {
		builtin, alias := "cpp", "py"
		if kind == "platform" {
			builtin, alias = "codeforces", "cf"
		}
		type testCase struct{ name, names, aliases, extensions, message string }
		cases := []testCase{
			{name: "empty name", names: "\n", message: "Name is required."},
			{name: "invalid name", names: "Bad Name\n", message: "Invalid " + kind + " name \"bad name\""},
			{name: "reserved name", names: "CON\n", message: "Invalid " + kind + " name"},
			{name: "builtin name", names: builtin + "\n", message: "already exists"},
			{name: "builtin alias as name", names: alias + "\n", message: "already exists"},
			{name: "user name", names: "existing\n", message: "already exists"},
			{name: "alias builtin name", aliases: builtin + "\n", message: "already exists"},
			{name: "alias builtin alias", aliases: alias + "\n", message: "already exists"},
			{name: "alias user name", aliases: "existing\n", message: "already exists"},
			{name: "alias user alias", aliases: "old\n", message: "already exists"},
			{name: "duplicate aliases", aliases: "fresh, FRESH\n", message: "already exists"},
			{name: "alias own name", aliases: "custom\n", message: "already exists"},
			{name: "invalid alias", aliases: "bad alias\n", message: "Invalid " + kind + " alias"},
			{name: "optional aliases"},
		}
		if kind == "language" {
			cases = append(cases,
				testCase{name: "empty extension", extensions: "\n", message: "File extension is required."},
				testCase{name: "invalid extension", extensions: ".kt\nbad/ext\n", message: "Invalid extension"},
			)
		}
		for _, tc := range cases {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				r, err := registry.Load(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				if err := r.Add(kind, registry.Entry{Name: "existing", Aliases: []string{"old"}, Extension: map[string]string{"language": "x"}[kind]}, nil); err != nil {
					t.Fatal(err)
				}
				input := tc.names + "CUSTOM\n" + tc.aliases + "\n"
				if kind == "language" {
					input += tc.extensions + "KT\n"
				}
				input += strings.Repeat("\n", 2*len(r.Entries(map[string]string{"language": "platform", "platform": "language"}[kind])))
				var out bytes.Buffer
				if err := addRegistry(r, kind, strings.NewReader(input), &out); err != nil {
					t.Fatal(err, out.String())
				}
				if tc.message != "" {
					index := strings.Index(out.String(), tc.message)
					if index < 0 {
						t.Fatalf("missing %q: %s", tc.message, out.String())
					}
					next := " filename"
					if tc.names != "" {
						next = "Aliases ("
					} else if tc.aliases != "" && kind == "language" {
						next = "File extension ("
					}
					if strings.Contains(out.String()[:index], next) {
						t.Fatalf("advanced to %q before validation: %s", next, out.String())
					}
				}
				loaded, err := registry.Load(r.Dir)
				if err != nil {
					t.Fatal(err)
				}
				d, err := loaded.Lookup(kind, "custom", true)
				if err != nil || len(d.Aliases) != 0 || (kind == "language" && d.Extension != "kt") {
					t.Fatal(d, err)
				}
				// Enter retains each displayed filename and creates an empty template.
				for _, b := range loaded.Data.Bindings {
					// Compare against defaults from the registry before any bindings.
					defaults := registry.Builtins()
					if err := defaults.Add(kind, d.Entry, nil); err != nil {
						t.Fatal(err)
					}
					want := defaults.DescribeBinding(b.Platform, b.Language).Filename
					data, err := os.ReadFile(filepath.Join(r.Dir, filepath.FromSlash(b.Template)))
					if err != nil || len(data) != 0 || b.Filename != want {
						t.Fatalf("default/empty template changed: %+v %q %v", b, data, err)
					}
				}
			})
		}
	}
}

func TestRegistryAddEOF(t *testing.T) {
	for _, kind := range []string{"language", "platform"} {
		for _, input := range []string{"", "\n", "bad name", "custom\n", "custom\npy\n", "custom\n\n", "custom\n\n\n"} {
			t.Run(kind+"/"+input, func(t *testing.T) {
				r, err := registry.Load(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				var out bytes.Buffer
				err = addRegistry(r, kind, strings.NewReader(input), &out)
				if !errors.Is(err, io.EOF) || !strings.Contains(err.Error(), "rerun the command with terminal input") {
					t.Fatalf("expected actionable EOF, got %v: %s", err, out.String())
				}
				if _, err := os.Stat(filepath.Join(r.Dir, "registry.json")); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("incomplete wizard wrote registry: %v", err)
				}
			})
		}
	}
}

func TestRegistryAddNormalizedAliasesAfterRetries(t *testing.T) {
	for _, kind := range []string{"language", "platform"} {
		t.Run(kind, func(t *testing.T) {
			r, err := registry.Load(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			input := "\nBad Name\nCustom\ncustom\n NEW , Other \n"
			if kind == "language" {
				input += "\n.kt\nKT\n"
			}
			input += strings.Repeat("\n", 8)
			var out bytes.Buffer
			if err := addRegistry(r, kind, strings.NewReader(input), &out); err != nil {
				t.Fatal(err, out.String())
			}
			for _, alias := range []string{"new", "other"} {
				d, err := r.Lookup(kind, alias, true)
				if err != nil || d.Name != "custom" {
					t.Fatal(d, err)
				}
			}
		})
	}
}
