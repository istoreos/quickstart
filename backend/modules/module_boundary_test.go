package modules

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestModulesDoNotImportServiceOrServer(t *testing.T) {
	const repoImport = "github.com/linkease/quick-start/istore-backend/"
	forbidden := map[string]struct{}{
		repoImport + "server":  {},
		repoImport + "service": {},
	}

	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if _, ok := forbidden[importPath]; ok {
				t.Errorf("%s imports forbidden package %q", path, importPath)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk modules: %v", err)
	}
}
