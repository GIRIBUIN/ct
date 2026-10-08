package environment

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/GIRIBUIN/ct/internal/registry"
)

func TestCustomLanguageCommonChecksOnly(t *testing.T) {
	c, path, _ := healthyChecker(t)
	r, err := registry.Load(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Add("language", registry.Entry{Name: "kotlin", Aliases: []string{"kt"}, Extension: "kt"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.Save(); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	c.lookup = func(name string) (string, error) { t.Fatalf("custom language tool lookup: %s", name); return "", nil }
	c.run = func(path string, args ...string) ([]byte, error) {
		t.Fatalf("custom language executed %s %v", path, args)
		return nil, nil
	}
	queries := 0
	c.extensions = func(code, profile string) ([]byte, error) {
		queries++
		return []byte("divyanshuagrawal.competitive-programming-helper\n"), nil
	}
	checks := c.check("kt")
	if queries != 1 || status(t, checks, "provider") != Skip || status(t, checks, "CPH") != OK || status(t, checks, "root") != OK {
		t.Fatal(checks)
	}
	for _, check := range checks {
		if check.Name == "selected" && check.Detail != "kotlin" {
			t.Fatal(check)
		}
		if check.Status == Fail {
			t.Fatal(check)
		}
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("override changed config")
	}
}
