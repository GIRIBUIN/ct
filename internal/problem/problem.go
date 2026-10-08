package problem

import (
	"fmt"
	"path/filepath"
	"strings"

	lang "github.com/GIRIBUIN/ct/internal/language"
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
	definition, err := lang.Lookup(value)
	return definition.Name, err
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
	definition, _ := lang.Lookup(language)
	var name string
	switch platform {
	case "codeforces":
		err = validateCodeforcesID(id)
		name = definition.CodeforcesFile
	case "programmers":
		err = validateProgrammersID(id)
		name = definition.ProgrammersFile
	}
	if err != nil {
		return "", err
	}
	return filepath.Join(root, platform, id, name), nil
}
