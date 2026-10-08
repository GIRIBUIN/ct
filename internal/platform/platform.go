// Package platform holds shipped platform metadata and problem ID rules.
package platform

import (
	"fmt"
	"regexp"
	"strings"
)

type Definition struct {
	Name        string
	Aliases     []string
	DefaultStem string
	IDPattern   string
}

func Definitions() []Definition {
	return []Definition{
		{"codeforces", []string{"cf"}, "main", `^[0-9]+[A-Za-z][0-9]*$`},
		{"programmers", []string{"pg"}, "solution", `^[0-9]+$`},
	}
}

func Lookup(value string) (Definition, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, d := range Definitions() {
		for _, name := range append([]string{d.Name}, d.Aliases...) {
			if name == value {
				return d, nil
			}
		}
	}
	return Definition{}, fmt.Errorf("unknown platform %q", value)
}

func ValidateID(name, id string) error {
	pattern := `^[A-Za-z0-9][A-Za-z0-9_-]*$`
	if d, err := Lookup(name); err == nil {
		pattern = d.IDPattern
	}
	if !regexp.MustCompile(pattern).MatchString(id) {
		return fmt.Errorf("invalid %s problem ID %q", name, id)
	}
	return nil
}
