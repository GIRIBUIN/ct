package problem

import (
	"fmt"
	"path/filepath"
	"strings"
)

func NormalizePlatform(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "cf", "codeforces":
		return "codeforces", nil
	case "pg", "programmers":
		return "programmers", nil
	default:
		return "", fmt.Errorf("unsupported platform %q (use codeforces or programmers)", value)
	}
}

func NormalizeLanguage(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "cpp", "c++":
		return "cpp", nil
	case "py", "python":
		return "python", nil
	default:
		return "", fmt.Errorf("unsupported language %q (use cpp or python)", value)
	}
}

// Target validates the ID before using it as a directory name.
func Target(root, platform, language, id string) (string, error) {
	platform, err := NormalizePlatform(platform)
	if err != nil {
		return "", err
	}
	language, err = NormalizeLanguage(language)
	if err != nil {
		return "", err
	}
	var name string
	switch platform {
	case "codeforces":
		name, err = codeforcesFilename(id)
	case "programmers":
		name, err = programmersFilename(id)
	}
	if err != nil {
		return "", err
	}
	extension := ".cpp"
	if language == "python" {
		extension = ".py"
	}
	return filepath.Join(root, platform, id, name+extension), nil
}
