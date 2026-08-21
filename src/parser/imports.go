package parser

import (
	"path/filepath"

	"Atom3/src/builtins"
	"Atom3/src/pkgs"
)

var imported = make(map[string]bool)

func ImportModule(name string) {
	key := pkgs.Normalize(name)
	if imported[key] {
		return
	}
	imported[key] = true

	manifest, dir, err := pkgs.Resolve(name)
	if err != nil {
		errorf("[Import Error]: %v\n", err)
		return
	}

	if manifest.Native != "" {
		m, ok := builtins.Native(manifest.Native)
		if !ok {
			errorf("[Import Error]: '%s' needs native module '%s', which this build of Atom-Neo does not provide\n", name, manifest.Native)
			return
		}
		builtins.Activate(m)
		return
	}

	RunFile(filepath.Join(dir, manifest.Main))
}
