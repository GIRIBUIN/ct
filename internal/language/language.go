// Package language defines local workspace languages independently of judge policy.
package language

import (
	"fmt"
	"strings"
)

type Extension struct{ Name, ID string }
type Definition struct {
	Name                            string
	Aliases                         []string
	CodeforcesFile, ProgrammersFile string
	Extension                       Extension
	FileExtension                   string
	Environment                     string
}

var definitions = []Definition{
	{"cpp", []string{"cpp", "c++"}, "main.cpp", "solution.cpp", Extension{"C/C++", "ms-vscode.cpptools"}, "cpp", "full"},
	{"python", []string{"python", "py"}, "main.py", "solution.py", Extension{"Python", "ms-python.python"}, "py", "detect"},
	{"java", []string{"java"}, "Main.java", "Solution.java", Extension{"Java", "redhat.java"}, "java", "full"},
	{"rust", []string{"rust", "rs"}, "main.rs", "solution.rs", Extension{"Rust", "rust-lang.rust-analyzer"}, "rs", "partial"},
}

func Definitions() []Definition {
	result := append([]Definition(nil), definitions...)
	for i := range result {
		result[i].Aliases = append([]string(nil), result[i].Aliases...)
	}
	return result
}

func Lookup(value string) (Definition, error) {
	key := strings.ToLower(strings.TrimSpace(value))
	for _, definition := range definitions {
		for _, alias := range definition.Aliases {
			if key == alias {
				return definition, nil
			}
		}
	}
	return Definition{}, fmt.Errorf("unsupported language %q (use cpp, python, java or rust)", value)
}

func Extensions(name string) []Extension {
	common := Extension{"CPH", "divyanshuagrawal.competitive-programming-helper"}
	definition, err := Lookup(name)
	if err != nil {
		return []Extension{common}
	}
	return []Extension{definition.Extension, common}
}
