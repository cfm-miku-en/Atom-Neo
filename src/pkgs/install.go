package pkgs

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Install accepts either a package zip or a folder holding an atom.json, so a
// package under development does not have to be zipped to be tried.
func Install(source string) (*Manifest, error) {
	info, err := os.Stat(source)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return installDir(source)
	}
	return installZip(source)
}

func readManifestFile(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "atom.json"))
	if err != nil {
		return nil, fmt.Errorf("no atom.json in %s", dir)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("malformed atom.json: %v", err)
	}
	if err := m.validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

func copyFile(from, to string) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}

	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(to)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func installDir(dir string) (*Manifest, error) {
	m, err := readManifestFile(dir)
	if err != nil {
		return nil, err
	}

	dest := filepath.Join(ModulesDir, Normalize(m.Name))
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}

	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, p)
		if err != nil || rel == "." {
			return err
		}

		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(p, target)
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

func installZip(archive string) (*Manifest, error) {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	entry, root := findManifest(r.File)
	if entry == nil {
		return nil, fmt.Errorf("no atom.json found in %s", archive)
	}

	m, err := readManifest(entry)
	if err != nil {
		return nil, err
	}

	dest := filepath.Join(ModulesDir, Normalize(m.Name))
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}

	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, root) {
			continue
		}
		if err := extract(f, strings.TrimPrefix(f.Name, root), dest); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// findManifest picks the shallowest atom.json so that archives zipped with a
// containing folder install the same as archives zipped from inside it.
func findManifest(files []*zip.File) (*zip.File, string) {
	var found *zip.File
	root := ""
	for _, f := range files {
		if path.Base(f.Name) != "atom.json" {
			continue
		}
		dir := path.Dir(f.Name)
		if dir == "." {
			dir = ""
		} else {
			dir += "/"
		}
		if found == nil || len(dir) < len(root) {
			found, root = f, dir
		}
	}
	return found, root
}

func readManifest(f *zip.File) (*Manifest, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("malformed atom.json: %v", err)
	}
	if err := m.validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

func extract(f *zip.File, rel, dest string) error {
	if rel == "" {
		return nil
	}

	target := filepath.Join(dest, filepath.FromSlash(rel))
	if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
		return fmt.Errorf("archive entry escapes the install directory: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
