package registry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/GIRIBUIN/ct/internal/configdir"
)

func Current() (*Registry, error) {
	dir, err := configdir.Dir()
	if err != nil {
		return nil, err
	}
	return Load(dir)
}

func Load(dir string) (*Registry, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	r := Builtins()
	r.Dir = dir
	path := filepath.Join(dir, "registry.json")
	// Legacy built-in usage does not require a registry or impose new path
	// restrictions on an existing user's config directory when it is absent.
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return r, nil
	} else if err != nil {
		return nil, err
	}
	if err := plainPath(path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return r, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read registry %s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	r.Data = Data{}
	err = decoder.Decode(&r.Data)
	if err == nil {
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			err = errors.New("expected one JSON object")
		}
	}
	if err == nil {
		err = r.validate()
	}
	if err != nil {
		return nil, fmt.Errorf("invalid registry %s: %w; repair this file or restore a backup (ct has not changed it)", path, err)
	}
	r.original = data
	return r, nil
}

// plainPath refuses symlinks/junctions in every existing path component. No
// registry-controlled external path is followed for reads, writes or removal.
func plainPath(path string) error {
	for {
		info, err := os.Lstat(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing linked registry/template path: %s", path)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return nil
		}
		path = parent
	}
}

func (r *Registry) Save() error { return r.SaveTemplates(nil) }

// SaveTemplates creates only new owned templates and publishes registry.json
// last. A failed registry write cleans up only files created by this call.
func (r *Registry) SaveTemplates(templates map[string][]byte) (result error) {
	if r.Dir == "" {
		return errors.New("registry storage directory is required")
	}
	if err := r.validate(); err != nil {
		return err
	}
	path := filepath.Join(r.Dir, "registry.json")
	if err := plainPath(path); err != nil {
		return err
	}
	if err := os.MkdirAll(r.Dir, 0700); err != nil {
		return err
	}
	lock := filepath.Join(r.Dir, ".registry.lock")
	f, err := os.OpenFile(lock, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("lock registry: %w; another ct may be updating it; remove %s only if no update is running", err, lock)
	}
	f.Close()
	defer os.Remove(lock)
	current, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if !bytes.Equal(current, r.original) {
		return errors.New("registry changed since it was loaded; retry the command")
	}
	var created []string
	defer func() {
		if result != nil {
			for _, path := range created {
				os.Remove(path)
			}
		}
	}()
	for relative, data := range templates {
		known := false
		for _, b := range r.Data.Bindings {
			if b.Template == relative {
				known = true
			}
		}
		if !known {
			return fmt.Errorf("unregistered template %q", relative)
		}
		path := filepath.Join(r.Dir, filepath.FromSlash(relative))
		if err := plainPath(path); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return fmt.Errorf("create template %s: %w; existing templates are never overwritten", path, err)
		}
		created = append(created, path)
		_, err = f.Write(data)
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	data, err := json.MarshalIndent(r.Data, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	f, err = os.CreateTemp(r.Dir, ".ct-registry-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("replace registry: %w", err)
	}
	r.original = data
	return nil
}

func (r *Registry) ReadTemplate(b Binding) ([]byte, error) {
	if b.Template == "" {
		return []byte{}, nil
	}
	if err := r.validate(); err != nil {
		return nil, err
	}
	// Accept only the exact binding from the validated registry.
	known := false
	for _, existing := range r.Data.Bindings {
		if existing == b {
			known = true
		}
	}
	if !known {
		return nil, errors.New("unregistered template binding")
	}
	path := filepath.Join(r.Dir, filepath.FromSlash(b.Template))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read custom template %s: %w; restore or edit this template", path, err)
	}
	return data, nil
}

func (r *Registry) Remove(kind, value string) error {
	d, err := r.Lookup(kind, value, false)
	if err != nil {
		return err
	}
	if d.Builtin {
		return fmt.Errorf("cannot remove built-in %s %q; use ct %s disable %s instead", kind, d.Name, kind, d.Name)
	}
	if err := r.validate(); err != nil {
		return err
	}
	next := r.clone()
	entries := &next.Data.Languages
	if kind == "platform" {
		entries = &next.Data.Platforms
	}
	var keep []Entry
	for _, e := range *entries {
		if e.Name != d.Name {
			keep = append(keep, e)
		}
	}
	// Remove state while lookup still recognizes the entry.
	if err := next.SetEnabled(kind, d.Name, true); err != nil {
		return err
	}
	*entries = keep
	var bindings []Binding
	var paths []string
	for _, b := range next.Data.Bindings {
		if (kind == "language" && b.Language == d.Name) || (kind == "platform" && b.Platform == d.Name) {
			if b.Template != "" {
				path := filepath.Join(r.Dir, filepath.FromSlash(b.Template))
				info, err := os.Lstat(path)
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					return err
				}
				if err == nil && !info.Mode().IsRegular() {
					return fmt.Errorf("not a regular owned template: %s", path)
				}
				paths = append(paths, path)
			}
		} else {
			bindings = append(bindings, b)
		}
	}
	next.Data.Bindings = bindings
	if err := next.Save(); err != nil {
		return err
	}
	*r = *next
	for _, path := range paths {
		if err := plainPath(path); err != nil {
			return fmt.Errorf("entry removed; template cleanup: %w", err)
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("entry removed; template cleanup: %w", err)
		}
		// Empty directories only; never recursively remove registry-supplied paths.
		for dir := filepath.Dir(path); dir != r.Dir; dir = filepath.Dir(dir) {
			if os.Remove(dir) != nil {
				break
			}
		}
	}
	return nil
}
