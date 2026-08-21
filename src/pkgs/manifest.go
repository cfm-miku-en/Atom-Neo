package pkgs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ModulesDir = "atom_modules"

type Manifest struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Native      string `json:"native,omitempty"`
	Main        string `json:"main,omitempty"`
}

// Normalize collapses the separators that would otherwise let two package names
// differing only by punctuation or case sit next to each other in the registry.
func Normalize(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return name
}

func (m *Manifest) validate() error {
	if m.Name == "" {
		return fmt.Errorf("atom.json is missing 'name'")
	}
	if m.Native == "" && m.Main == "" {
		return fmt.Errorf("package '%s' declares neither 'main' nor 'native'", m.Name)
	}
	return nil
}

func Resolve(name string) (*Manifest, string, error) {
	dir := filepath.Join(ModulesDir, Normalize(name))
	data, err := os.ReadFile(filepath.Join(dir, "atom.json"))
	if os.IsNotExist(err) {
		return nil, "", fmt.Errorf("package '%s' is not installed", name)
	}
	if err != nil {
		return nil, "", err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, "", fmt.Errorf("package '%s' has a malformed atom.json: %v", name, err)
	}
	if err := m.validate(); err != nil {
		return nil, "", err
	}
	return &m, dir, nil
}

// LoadProject reads the atom.json in the working directory, which uses the same
// shape as a package manifest so a project can be published as-is.
func LoadProject() (*Manifest, error) {
	data, err := os.ReadFile("atom.json")
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no atom.json here, so there is no project to run")
	}
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("malformed atom.json: %v", err)
	}
	if m.Main == "" {
		return nil, fmt.Errorf("atom.json does not set 'main'")
	}
	return &m, nil
}
