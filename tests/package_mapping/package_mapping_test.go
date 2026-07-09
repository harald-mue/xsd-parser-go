package package_mapping

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harald-mue/xsd-parser-go/internal/pipeline"
)

func TestNamespacePackageMappingPolymorphismCompiles(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "package_mapping_polymorphic", "schema.xsd")
	err := pipeline.GenerateWithOptions([]string{schema}, pipeline.GenerateOptions{
		Package:    "model",
		OutDir:     dir,
		ModulePath: "tmp/pkgpoly",
		NamespacePackages: map[string]string{
			"http://example.com/root": "root",
			"http://example.com/base": "base",
			"http://example.com/impl": "impl",
		},
	})
	if err != nil {
		t.Fatalf("GenerateWithOptions returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "integration_test.go"), []byte(`package pkgpoly_test

import (
	"encoding/xml"
	"testing"

	"tmp/pkgpoly/base"
	"tmp/pkgpoly/impl"
	"tmp/pkgpoly/root"
)

func TestCrossPackagePolymorphicMarshal(t *testing.T) {
	v := root.ZooType{Animal: []base.Animal{&impl.DogType{Id: 7, Name: "Rex"}}}
	out, err := xml.Marshal(&v)
	if err != nil {
		t.Fatal(err)
	}
	var v2 root.ZooType
	if err := xml.Unmarshal(out, &v2); err != nil {
		t.Fatalf("re-unmarshal: %v\n%s", err, out)
	}
	if len(v2.Animal) != 1 {
		t.Fatalf("Animal count = %d, want 1; xml=%s", len(v2.Animal), out)
	}
}
`), 0o644); err != nil {
		t.Fatalf("write integration test: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "tmp/pkgpoly"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestNamespacePackageMappingWithRuntimePackageCompiles(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "package_mapping_polymorphic", "schema.xsd")
	err := pipeline.GenerateWithOptions([]string{schema}, pipeline.GenerateOptions{
		Package:        "model",
		OutDir:         dir,
		ModulePath:     "tmp/runtime",
		RuntimePackage: "xsdxml",
		NamespacePackages: map[string]string{
			"http://example.com/root": "root",
			"http://example.com/base": "base",
			"http://example.com/impl": "impl",
		},
	})
	if err != nil {
		t.Fatalf("GenerateWithOptions returned error: %v", err)
	}
	for _, path := range []string{"root/models.go", "base/models.go", "impl/models.go", "xsdxml/models.go"} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("missing generated file %s: %v", path, err)
		}
	}
	assertRuntimeConsolidated(t, dir)
	if out, err := run(dir, "go", "mod", "init", "tmp/runtime"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func assertRuntimeConsolidated(t *testing.T, dir string) {
	t.Helper()
	runtimeData, err := os.ReadFile(filepath.Join(dir, "xsdxml", "models.go"))
	if err != nil {
		t.Fatalf("read runtime file: %v", err)
	}
	for _, want := range []string{"func ValidateField(", "func PrefixedStart(", "func PrefixedAttr(", "func WildcardNamespaceAllowed(", "func WildcardElementProcessContentsAllowed(", "func WildcardAttributeProcessContentsAllowed("} {
		if !strings.Contains(string(runtimeData), want) {
			t.Fatalf("runtime file missing %s", want)
		}
	}
	for _, pkg := range []string{"root", "base", "impl"} {
		data, err := os.ReadFile(filepath.Join(dir, pkg, "models.go"))
		if err != nil {
			t.Fatalf("read %s: %v", pkg, err)
		}
		for _, forbidden := range []string{"func validateField(", "func prefixedStart(", "func prefixedAttr(", "type validatable"} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("%s still contains local helper %s", pkg, forbidden)
			}
		}
	}
}

func TestNamespacePackageMappingCompiles(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "package_mapping", "schema.xsd")
	err := pipeline.GenerateWithOptions([]string{schema}, pipeline.GenerateOptions{
		Package:    "model",
		OutDir:     dir,
		ModulePath: "tmp/pkgmap",
		NamespacePackages: map[string]string{
			"http://example.com/root": "root",
			"http://example.com/base": "base",
		},
	})
	if err != nil {
		t.Fatalf("GenerateWithOptions returned error: %v", err)
	}
	for _, path := range []string{"root/models.go", "base/models.go"} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("missing generated file %s: %v", path, err)
		}
	}
	if out, err := run(dir, "go", "mod", "init", "tmp/pkgmap"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func run(dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}
