package template

import (
	"embed"
	"fmt"

	"github.com/GIRIBUIN/ct/internal/problem"
)

//go:embed templates/*/*.tmpl
var files embed.FS

func Load(platform, language string) ([]byte, error) {
	platform, err := problem.NormalizePlatform(platform)
	if err != nil {
		return nil, err
	}
	language, err = problem.NormalizeLanguage(language)
	if err != nil {
		return nil, err
	}
	data, err := files.ReadFile("templates/" + platform + "/" + language + ".tmpl")
	if err != nil {
		return nil, fmt.Errorf("load template: %w", err)
	}
	return data, nil
}
