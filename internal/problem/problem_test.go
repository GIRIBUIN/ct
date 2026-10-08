package problem

import (
	"path/filepath"
	"testing"
)

func TestNormalizePlatform(t *testing.T) {
	for alias, want := range map[string]string{"cf": "codeforces", "codeforces": "codeforces", "pg": "programmers", "programmers": "programmers"} {
		t.Run(alias, func(t *testing.T) {
			got, err := NormalizePlatform(alias)
			if err != nil || got != want {
				t.Fatalf("NormalizePlatform(%q) = %q, %v; want %q", alias, got, err, want)
			}
		})
	}
	if _, err := NormalizePlatform("unknown"); err == nil {
		t.Fatal("unsupported platform accepted")
	}
}

func TestNormalizeLanguage(t *testing.T) {
	for alias, want := range map[string]string{"cpp": "cpp", "c++": "cpp", "py": "python", "python": "python", "java": "java", "rust": "rust", "rs": "rust"} {
		t.Run(alias, func(t *testing.T) {
			got, err := NormalizeLanguage(alias)
			if err != nil || got != want {
				t.Fatalf("NormalizeLanguage(%q) = %q, %v; want %q", alias, got, err, want)
			}
		})
	}
	if _, err := NormalizeLanguage("kotlin"); err == nil {
		t.Fatal("unsupported language accepted")
	}
}

func TestTarget(t *testing.T) {
	root := t.TempDir()
	for _, tt := range []struct{ platform, language, id, dir, filename string }{
		{"cf", "cpp", "71A", "codeforces", "main.cpp"},
		{"codeforces", "py", "71A", "codeforces", "main.py"},
		{"pg", "c++", "181188", "programmers", "solution.cpp"},
		{"programmers", "python", "181188", "programmers", "solution.py"},
		{"cf", "cpp", "123A1", "codeforces", "main.cpp"},
		{"cf", "java", "71A", "codeforces", "Main.java"},
		{"cf", "rs", "71A", "codeforces", "main.rs"},
		{"pg", "java", "181188", "programmers", "Solution.java"},
		{"pg", "rust", "181188", "programmers", "solution.rs"},
	} {
		t.Run(tt.dir+tt.filename+tt.id, func(t *testing.T) {
			got, err := Target(root, tt.platform, tt.language, tt.id)
			want := filepath.Join(root, tt.dir, tt.id, tt.filename)
			if err != nil || got != want {
				t.Fatalf("Target = %q, %v; want %q", got, err, want)
			}
		})
	}
	for _, platform := range []string{"cf", "pg"} {
		for _, id := range []string{"", "..", "../71A", `..\71A`, "/71A", `C:\71A`, "doctor", "setup"} {
			if _, err := Target(root, platform, "cpp", id); err == nil {
				t.Errorf("unsafe or invalid ID accepted: %q (%s)", id, platform)
			}
		}
	}
}
