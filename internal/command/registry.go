package command

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/GIRIBUIN/ct/internal/registry"
)

func registryCommand(args []string, input io.Reader, output io.Writer) error {
	kind := args[0]
	if len(args) == 1 || (len(args) == 2 && (args[1] == "--help" || args[1] == "-h")) {
		fmt.Fprintf(output, "Usage: ct %s list [--all]\n       ct %s show|remove|enable|disable <name>\n       ct %s add\n\nList includes enabled and disabled entries. Add is interactive; templates may be empty or copied from a file.\n", kind, kind, kind)
		return nil
	}
	action := args[1]
	switch action {
	case "list":
		if len(args) > 3 || (len(args) == 3 && args[2] != "--all") {
			return fmt.Errorf("usage: ct %s list [--all]", kind)
		}
	case "add":
		if len(args) != 2 {
			return fmt.Errorf("usage: ct %s add (interactive)", kind)
		}
	case "show", "remove", "enable", "disable":
		if len(args) != 3 {
			return fmt.Errorf("usage: ct %s %s <name>", kind, action)
		}
	default:
		return fmt.Errorf("unknown %s command %q; use list, show, add, remove, enable or disable", kind, action)
	}
	r, err := registry.Current()
	if err != nil {
		return err
	}
	if action == "list" {
		w := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
		fmt.Fprintf(w, "%ss\n\nNAME\tALIASES\tSOURCE\tSTATUS", strings.ToUpper(kind[:1])+kind[1:])
		if kind == "language" {
			fmt.Fprint(w, "\tENV")
		}
		fmt.Fprintln(w)
		for _, d := range r.Entries(kind) {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s", d.Name, strings.Join(d.Aliases, ","), entrySource(d), entryStatus(d))
			if kind == "language" {
				fmt.Fprint(w, "\t", d.Environment)
			}
			fmt.Fprintln(w)
		}
		return w.Flush()
	}
	if action == "add" {
		return addRegistry(r, kind, input, output)
	}
	d, err := r.Lookup(kind, args[2], false)
	if err != nil {
		return err
	}
	switch action {
	case "show":
		fmt.Fprintf(output, "Name: %s\nAliases: %s\nSource: %s\nStatus: %s\n", d.Name, strings.Join(d.Aliases, ", "), entrySource(d), entryStatus(d))
		other := "language"
		if kind == "language" {
			other = "platform"
			fmt.Fprintf(output, "Extension: %s\nEnvironment: %s\n", d.Extension, d.Environment)
			if d.EditorExtension.ID != "" {
				fmt.Fprintln(output, "VS Code extension:", d.EditorExtension.ID)
			}
		}
		for _, counterpart := range r.Entries(other) {
			p, l := d.Name, counterpart.Name
			if kind == "language" {
				p, l = l, p
			}
			b := r.DescribeBinding(p, l)
			template := "empty (no custom binding)"
			if b.Embedded {
				template = "embedded"
			} else if b.Template != "" {
				template = filepath.Join(r.Dir, filepath.FromSlash(b.Template))
			}
			fmt.Fprintf(output, "%s/%s: %s; template: %s\n", p, l, b.Filename, template)
		}
		return nil
	case "enable", "disable":
		if err := r.SetEnabled(kind, d.Name, action == "enable"); err != nil {
			return err
		}
		if err := r.Save(); err != nil {
			return err
		}
		fmt.Fprintf(output, "%s %s %sd. Existing solution files are unchanged.\n", kind, d.Name, action)
		return nil
	case "remove":
		if d.Builtin {
			return fmt.Errorf("cannot remove built-in %s %q; use ct %s disable %s instead", kind, d.Name, kind, d.Name)
		}
		fmt.Fprintf(output, "Remove user %s %q?\n\nThis will remove:\n  - registry entry\n  - ct-owned templates for %s\n\nThis will NOT remove:\n  - existing coding-test solution files\n  - compiler/toolchain\n  - VS Code extensions\n\n", kind, d.Name, d.Name)
		answer, err := registryPrompt(bufio.NewReader(input), output, "Continue?", "y/N")
		if err != nil || (strings.ToLower(answer) != "y" && strings.ToLower(answer) != "yes") {
			fmt.Fprintln(output, "Removal cancelled.")
			return nil
		}
		if err := r.Remove(kind, d.Name); err != nil {
			return err
		}
		fmt.Fprintf(output, "Removed user %s %s. Existing solution files are unchanged.\n", kind, d.Name)
		return nil
	}
	return nil
}

func entrySource(d registry.Definition) string {
	if d.Builtin {
		return "built-in"
	}
	return "user"
}
func entryStatus(d registry.Definition) string {
	if d.Enabled {
		return "enabled"
	}
	return "disabled"
}

func registryPrompt(reader *bufio.Reader, out io.Writer, label, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && len(line) > 0) {
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read %s: input ended; rerun the command with terminal input: %w", label, err)
		}
		return "", fmt.Errorf("read %s: %w", label, err)
	}
	return strings.TrimSpace(line), nil
}

func registryValidatedPrompt(reader *bufio.Reader, out io.Writer, label, required string, validate func(string) error) (string, error) {
	for {
		value, err := registryPrompt(reader, out, label, "")
		if err != nil {
			return "", err
		}
		if value == "" && required != "" {
			fmt.Fprintln(out, required+" is required.")
			continue
		}
		value = strings.ToLower(value)
		if err := validate(value); err != nil {
			message := err.Error()
			fmt.Fprintln(out, strings.ToUpper(message[:1])+message[1:]+".")
			continue
		}
		return value, nil
	}
}

func registryAliases(value string) []string {
	if value == "" {
		return nil
	}
	aliases := strings.Split(value, ",")
	for i := range aliases {
		aliases[i] = strings.TrimSpace(aliases[i])
	}
	return aliases
}

func addRegistry(r *registry.Registry, kind string, input io.Reader, output io.Writer) error {
	reader := bufio.NewReader(input)
	fmt.Fprintln(output, "Add", kind)
	name, err := registryValidatedPrompt(reader, output, "Name", "Name", func(value string) error {
		return r.ValidateNewName(kind, value)
	})
	if err != nil {
		return err
	}
	aliases, err := registryValidatedPrompt(reader, output, "Aliases (comma-separated, optional)", "", func(value string) error {
		return r.ValidateNewAliases(kind, name, registryAliases(value))
	})
	if err != nil {
		return err
	}
	e := registry.Entry{Name: name, Aliases: registryAliases(aliases)}
	if kind == "language" {
		e.Extension, err = registryValidatedPrompt(reader, output, "File extension (without dot)", "File extension", registry.ValidateExtension)
		if err != nil {
			return err
		}
	}
	if err := r.Add(kind, e, nil); err != nil {
		return err
	}
	other := "language"
	if kind == "language" {
		other = "platform"
	}
	templates := map[string][]byte{}
	for _, counterpart := range r.Entries(other) {
		if !counterpart.Enabled {
			continue
		}
		p, l := e.Name, counterpart.Name
		if kind == "language" {
			p, l = l, p
		}
		b := r.DescribeBinding(p, l)
		name, err := registryPrompt(reader, output, counterpart.Name+" filename", b.Filename)
		if err != nil {
			return err
		}
		if name != "" {
			b.Filename = name
		}
		if err := registry.Filename(b.Filename); err != nil {
			return err
		}
		b.Template = registry.TemplatePath(kind, e.Name, counterpart.Name)
		source, err := registryPrompt(reader, output, "Template source file for "+counterpart.Name+" (Enter creates empty template)", "")
		if err != nil {
			return err
		}
		var data []byte
		if source != "" {
			data, err = os.ReadFile(source)
			if err != nil {
				return fmt.Errorf("read template source: %w", err)
			}
		}
		templates[b.Template] = data
		r.Data.Bindings = append(r.Data.Bindings, b)
	}
	if err := r.SaveTemplates(templates); err != nil {
		return err
	}
	fmt.Fprintf(output, "Added user %s %s.\n", kind, e.Name)
	for _, b := range r.Data.Bindings {
		if _, ok := templates[b.Template]; ok {
			fmt.Fprintln(output, "Edit template:", filepath.Join(r.Dir, filepath.FromSlash(b.Template)))
		}
	}
	return nil
}
