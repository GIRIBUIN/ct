// Package registry merges shipped metadata with machine-local workspace definitions.
// Registry data never defines commands or executable environment providers.
package registry

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GIRIBUIN/ct/internal/language"
	"github.com/GIRIBUIN/ct/internal/platform"
)

type Entry struct {
	Name      string   `json:"name"`
	Aliases   []string `json:"aliases,omitempty"`
	Extension string   `json:"extension,omitempty"`
}

type Definition struct {
	Entry
	Builtin         bool
	Enabled         bool
	Environment     string
	EditorExtension language.Extension
}

type Binding struct {
	Platform string `json:"platform"`
	Language string `json:"language"`
	Filename string `json:"filename"`
	Template string `json:"template,omitempty"` // canonical relative path, never arbitrary
	Embedded bool   `json:"-"`
}

type Data struct {
	Version           int       `json:"version"`
	Languages         []Entry   `json:"languages,omitempty"`
	Platforms         []Entry   `json:"platforms,omitempty"`
	DisabledLanguages []string  `json:"disabled_languages,omitempty"`
	DisabledPlatforms []string  `json:"disabled_platforms,omitempty"`
	Bindings          []Binding `json:"bindings,omitempty"`
}

type Registry struct {
	Dir      string
	Data     Data
	original []byte
}

func Builtins() *Registry { return &Registry{Data: Data{Version: 1}} }

func (r *Registry) Entries(kind string) []Definition {
	var result []Definition
	users, disabled := r.Data.Languages, r.Data.DisabledLanguages
	if kind == "language" {
		for _, d := range language.Definitions() {
			var aliases []string
			for _, a := range d.Aliases {
				if a != d.Name {
					aliases = append(aliases, a)
				}
			}
			result = append(result, Definition{Entry: Entry{d.Name, aliases, d.FileExtension}, Builtin: true, Enabled: true, Environment: d.Environment, EditorExtension: d.Extension})
		}
	} else {
		users, disabled = r.Data.Platforms, r.Data.DisabledPlatforms
		for _, d := range platform.Definitions() {
			result = append(result, Definition{Entry: Entry{Name: d.Name, Aliases: d.Aliases}, Builtin: true, Enabled: true})
		}
	}
	for _, e := range users {
		result = append(result, Definition{Entry: e, Enabled: true, Environment: "none"})
	}
	for i := range result {
		for _, name := range disabled {
			if result[i].Name == name {
				result[i].Enabled = false
			}
		}
	}
	return result
}

func (r *Registry) Lookup(kind, value string, enabled bool) (Definition, error) {
	key := strings.ToLower(strings.TrimSpace(value))
	var available []string
	for _, d := range r.Entries(kind) {
		if d.Enabled {
			available = append(available, d.Name)
		}
		for _, name := range append([]string{d.Name}, d.Aliases...) {
			if key == name {
				if enabled && !d.Enabled {
					return Definition{}, fmt.Errorf("%s %q is disabled; run ct %s enable %s or choose an enabled default with ct config", kind, d.Name, kind, d.Name)
				}
				return d, nil
			}
		}
	}
	return Definition{}, fmt.Errorf("unsupported %s %q; enabled entries: %s; choose a valid default with ct config", kind, value, strings.Join(available, ", "))
}

func (r *Registry) Normalize(kind, value string) (string, error) {
	d, err := r.Lookup(kind, value, true)
	return d.Name, err
}

func (r *Registry) Binding(p, l string) (Binding, error) {
	pd, err := r.Lookup("platform", p, true)
	if err != nil {
		return Binding{}, err
	}
	ld, err := r.Lookup("language", l, true)
	if err != nil {
		return Binding{}, err
	}
	return r.DescribeBinding(pd.Name, ld.Name), nil
}

// DescribeBinding also works for disabled entries shown by registry commands.
func (r *Registry) DescribeBinding(p, l string) Binding {
	for _, b := range r.Data.Bindings {
		if b.Platform == p && b.Language == l {
			return b
		}
	}
	ld, _ := r.Lookup("language", l, false)
	b := Binding{Platform: p, Language: l, Filename: "main." + ld.Extension}
	if d, err := language.Lookup(l); err == nil {
		b.Filename = d.CodeforcesFile
	}
	if p == "programmers" {
		b.Filename = "solution." + ld.Extension
		if d, err := language.Lookup(l); err == nil {
			b.Filename = d.ProgrammersFile
		}
	}
	pd, _ := r.Lookup("platform", p, false)
	b.Embedded = pd.Builtin && ld.Builtin
	return b
}

func TemplatePath(kind, name, counterpart string) string {
	return "templates/" + kind + "s/" + name + "/" + counterpart + ".tmpl"
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
var aliasPattern = regexp.MustCompile(`^[a-z][a-z0-9_+\-]*$`)
var extensionPattern = regexp.MustCompile(`^[a-z0-9]+$`)

func validateName(kind, name string) error {
	if !namePattern.MatchString(name) || Filename(name) != nil {
		return fmt.Errorf("invalid %s name %q (use lowercase letters, digits, - or _)", kind, name)
	}
	return nil
}

// ValidateNewName checks a proposed name against all names and aliases,
// including disabled entries, without changing the registry.
func (r *Registry) ValidateNewName(kind, name string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if err := validateName(kind, name); err != nil {
		return err
	}
	if _, err := r.Lookup(kind, name, false); err == nil {
		return fmt.Errorf("%s %q already exists", kind, name)
	}
	return nil
}

func validateAliases(kind string, aliases []string, seen map[string]bool) error {
	for _, key := range aliases {
		if !aliasPattern.MatchString(key) || Filename(key) != nil {
			return fmt.Errorf("invalid %s alias %q", kind, key)
		}
		if seen[key] {
			return fmt.Errorf("%s name/alias %q already exists", kind, key)
		}
		seen[key] = true
	}
	return nil
}

// ValidateNewAliases checks normalized aliases without changing the registry.
func (r *Registry) ValidateNewAliases(kind, name string, aliases []string) error {
	seen := map[string]bool{strings.ToLower(strings.TrimSpace(name)): true}
	for _, d := range r.Entries(kind) {
		for _, key := range append([]string{d.Name}, d.Aliases...) {
			seen[key] = true
		}
	}
	normalized := make([]string, len(aliases))
	for i, alias := range aliases {
		normalized[i] = strings.ToLower(strings.TrimSpace(alias))
	}
	return validateAliases(kind, normalized, seen)
}

func ValidateExtension(extension string) error {
	if !extensionPattern.MatchString(extension) {
		return fmt.Errorf("invalid extension %q (use letters/digits without a dot)", extension)
	}
	return nil
}

func Filename(name string) error {
	if name == "" || name == "." || name == ".." || strings.TrimSpace(name) != name || strings.HasSuffix(name, ".") || strings.ContainsAny(name, `/\:<>"|?*`) {
		return fmt.Errorf("invalid single filename %q", name)
	}
	for _, ch := range name {
		if ch < 32 || ch == 127 {
			return fmt.Errorf("invalid filename %q", name)
		}
	}
	stem := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	if regexp.MustCompile(`^(CON|PRN|AUX|NUL|COM[0-9]|LPT[0-9])$`).MatchString(stem) {
		return fmt.Errorf("reserved Windows filename %q", name)
	}
	return nil
}

func (r *Registry) Add(kind string, e Entry, bindings []Binding) error {
	e.Name = strings.ToLower(strings.TrimSpace(e.Name))
	e.Extension = strings.ToLower(strings.TrimSpace(e.Extension))
	for i := range e.Aliases {
		e.Aliases[i] = strings.ToLower(strings.TrimSpace(e.Aliases[i]))
	}
	next := r.clone()
	if kind == "language" {
		next.Data.Languages = append(next.Data.Languages, e)
	} else {
		next.Data.Platforms = append(next.Data.Platforms, e)
	}
	next.Data.Bindings = append(next.Data.Bindings, bindings...)
	if err := next.validate(); err != nil {
		return err
	}
	r.Data = next.Data
	return nil
}

func (r *Registry) SetEnabled(kind, value string, enabled bool) error {
	d, err := r.Lookup(kind, value, false)
	if err != nil {
		return err
	}
	list := &r.Data.DisabledLanguages
	if kind == "platform" {
		list = &r.Data.DisabledPlatforms
	}
	var updated []string
	for _, name := range *list {
		if name != d.Name {
			updated = append(updated, name)
		}
	}
	if !enabled {
		updated = append(updated, d.Name)
	}
	*list = updated
	return nil
}

func (r *Registry) clone() *Registry {
	n := *r
	n.Data.Languages = append([]Entry(nil), r.Data.Languages...)
	n.Data.Platforms = append([]Entry(nil), r.Data.Platforms...)
	n.Data.Bindings = append([]Binding(nil), r.Data.Bindings...)
	return &n
}

func (r *Registry) validate() error {
	if r.Data.Version != 1 {
		return fmt.Errorf("unsupported registry version %d", r.Data.Version)
	}
	for _, kind := range []string{"language", "platform"} {
		seen := map[string]bool{}
		for _, d := range r.Entries(kind) {
			if err := validateName(kind, d.Name); err != nil {
				return err
			}
			if kind == "language" {
				if err := ValidateExtension(d.Extension); err != nil {
					return err
				}
			}
			if kind == "platform" && d.Extension != "" {
				return fmt.Errorf("platform cannot define a language extension")
			}
			if err := validateAliases(kind, append([]string{d.Name}, d.Aliases...), seen); err != nil {
				return err
			}
		}
		disabled := r.Data.DisabledLanguages
		if kind == "platform" {
			disabled = r.Data.DisabledPlatforms
		}
		seenDisabled := map[string]bool{}
		for _, name := range disabled {
			d, err := r.Lookup(kind, name, false)
			if err != nil || d.Name != name || seenDisabled[name] {
				return fmt.Errorf("invalid disabled %s %q", kind, name)
			}
			seenDisabled[name] = true
		}
	}
	seen := map[string]bool{}
	for _, b := range r.Data.Bindings {
		p, pe := r.Lookup("platform", b.Platform, false)
		l, le := r.Lookup("language", b.Language, false)
		if pe != nil || le != nil || p.Name != b.Platform || l.Name != b.Language {
			return fmt.Errorf("binding uses unknown/noncanonical entry: %s/%s", b.Platform, b.Language)
		}
		if p.Builtin && l.Builtin {
			return fmt.Errorf("cannot override built-in binding %s/%s", p.Name, l.Name)
		}
		if err := Filename(b.Filename); err != nil {
			return err
		}
		key := b.Platform + "/" + b.Language
		if seen[key] {
			return fmt.Errorf("duplicate binding %s", key)
		}
		seen[key] = true
		if b.Template != "" {
			owned := (!l.Builtin && b.Template == TemplatePath("language", l.Name, p.Name)) || (!p.Builtin && b.Template == TemplatePath("platform", p.Name, l.Name))
			if !owned {
				return fmt.Errorf("unsafe template path %q; expected ct-owned binding path under templates/", b.Template)
			}
			if r.Dir != "" {
				if err := plainPath(filepath.Join(r.Dir, filepath.FromSlash(b.Template))); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
