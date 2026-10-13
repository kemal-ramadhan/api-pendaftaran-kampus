package internal_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureIsolation(t *testing.T) {
	root := filepath.Join("modules")
	fset := token.NewFileSet()

	modules := []string{
		"admission", "exam", "finance", "identity",
		"nim", "notification", "organization",
		"scoring", "selection", "student",
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Temukan modul pemilik file ini
		var currentModule string
		for _, m := range modules {
			prefix := filepath.Join("modules", m)
			if strings.HasPrefix(path, prefix) {
				currentModule = m
				break
			}
		}

		if currentModule == "" {
			return nil
		}

		node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Logf("Warning: gagal parse file %s: %v", path, err)
			return nil
		}

		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			for _, otherModule := range modules {
				if otherModule == currentModule {
					continue
				}
				// Aturan: dilarang mengimpor internal milik modul lain
				forbiddenPrefix := "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/" + otherModule + "/internal"
				if strings.HasPrefix(importPath, forbiddenPrefix) {
					t.Errorf("Pelanggaran Batas Arsitektur di %s: Modul '%s' mengimpor internal terlarang '%s' dari modul '%s'",
						path, currentModule, importPath, otherModule)
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Gagal menjelajahi direktori modules: %v", err)
	}
}
