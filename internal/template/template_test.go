package template

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/GIRIBUIN/ct/internal/problem"
)

// Opt-in keeps ordinary CI independent of unspecified host toolchains/linkers.
// CT_TEMPLATE_SMOKE=1 go test ./internal/template compiles with installed tools only.
func TestCompileJavaRustTemplates(t *testing.T) {
	if os.Getenv("CT_TEMPLATE_SMOKE") != "1" {
		t.Skip("set CT_TEMPLATE_SMOKE=1 for installed-compiler smoke tests")
	}
	for _, lang := range []string{"java", "rust"} {
		for _, platform := range []string{"codeforces", "programmers"} {
			t.Run(platform+"/"+lang, func(t *testing.T) {
				compiler := "javac"
				if lang == "rust" {
					compiler = "rustc"
				}
				path, err := exec.LookPath(compiler)
				if err != nil {
					t.Skip(compiler + " unavailable")
				}
				dir := t.TempDir()
				id := "71A"
				if platform == "programmers" {
					id = "181188"
				}
				target, err := problem.Target(dir, platform, lang, id)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
					t.Fatal(err)
				}
				data, err := Load(platform, lang)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, data, 0600); err != nil {
					t.Fatal(err)
				}
				args := []string{"--release", "17", "-d", dir, target}
				if lang == "rust" {
					args = []string{"--edition=2021", target, "-o", filepath.Join(dir, "output")}
					if platform == "programmers" {
						args = append(args, "--crate-type=lib")
					}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, path, args...)
				cmd.Env = append(os.Environ(), "RUSTUP_AUTO_INSTALL=0")
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("template compile: %v\n%s", err, output)
				}
			})
		}
	}
}
