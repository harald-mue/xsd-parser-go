package enterprise

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harald-mue/xsd-parser-go/internal/pipeline"
)

func TestIDREFAndEnumHelpers(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "idref", "schema.xsd")
	if err := pipeline.Generate([]string{schema}, "model", dir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "idref_test.go"), []byte(`package model

import "testing"

func TestIDREFHelpers(t *testing.T) {
	var id XSDID
	if err := id.UnmarshalText([]byte("node1")); err != nil {
		t.Fatalf("valid ID rejected: %v", err)
	}
	if err := id.UnmarshalText([]byte("1bad")); err == nil {
		t.Fatalf("invalid ID accepted")
	}
	var refs XSDIDREFS
	if err := refs.UnmarshalText([]byte("a b")); err != nil {
		t.Fatalf("valid IDREFS rejected: %v", err)
	}
	if len(refs) != 2 || refs[0] != "a" || refs[1] != "b" {
		t.Fatalf("IDREFS decoded = %#v", refs)
	}
}
`), 0o644); err != nil {
		t.Fatalf("write idref test: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "idreftest"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestCatalogResolutionAndEnumHelpers(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "catalog", "schema.xsd")
	catalog := filepath.Join("..", "fixtures", "catalog", "catalog.xml")
	if err := pipeline.GenerateWithOptions([]string{schema}, pipeline.GenerateOptions{Package: "model", OutDir: dir, Catalogs: []string{catalog}}); err != nil {
		t.Fatalf("GenerateWithOptions returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "models.go"))
	if err != nil {
		t.Fatalf("read generated model: %v", err)
	}
	for _, want := range []string{"func ExternalCodeValues() []ExternalCode", "func ParseExternalCode(value string)"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("generated enum helper missing %q", want)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "enum_test.go"), []byte(`package model

import "testing"

func TestEnumHelpers(t *testing.T) {
	if got := ExternalCodeValues(); len(got) != 1 || got[0] != ExternalCodeA {
		t.Fatalf("values = %#v", got)
	}
	if v, err := ParseExternalCode("A"); err != nil || v != ExternalCodeA {
		t.Fatalf("parse = %v, %v", v, err)
	}
	if _, err := ParseExternalCode("B"); err == nil {
		t.Fatalf("invalid enum accepted")
	}
}
`), 0o644); err != nil {
		t.Fatalf("write enum test: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "catalogenumtest"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestTypeMappingAndBindingSubset(t *testing.T) {
	t.Run("type_mapping", func(t *testing.T) {
		dir := t.TempDir()
		schema := filepath.Join("..", "fixtures", "xsd_scalar_types", "schema.xsd")
		if err := pipeline.GenerateWithOptions([]string{schema}, pipeline.GenerateOptions{Package: "model", OutDir: dir, TypeMappings: map[string]string{"xs:dateTime": "time.Time"}}); err != nil {
			t.Fatalf("GenerateWithOptions returned error: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "models.go"))
		if err != nil {
			t.Fatalf("read generated model: %v", err)
		}
		if !strings.Contains(string(data), "\"time\"") || !strings.Contains(string(data), "Created  time.Time") {
			t.Fatalf("type mapping not reflected in generated code")
		}
		if out, err := run(dir, "go", "mod", "init", "typemappingtest"); err != nil {
			t.Fatalf("go mod init: %v\n%s", err, out)
		}
		if out, err := run(dir, "go", "test", "./..."); err != nil {
			t.Fatalf("go test: %v\n%s", err, out)
		}
	})
	t.Run("binding", func(t *testing.T) {
		dir := t.TempDir()
		schema := filepath.Join("..", "fixtures", "binding", "schema.xsd")
		binding := filepath.Join("..", "fixtures", "binding", "bindings.xjb")
		if err := pipeline.GenerateWithOptions([]string{schema}, pipeline.GenerateOptions{Package: "model", OutDir: dir, BindingFiles: []string{binding}}); err != nil {
			t.Fatalf("GenerateWithOptions returned error: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "models.go"))
		if err != nil {
			t.Fatalf("read generated model: %v", err)
		}
		if !strings.Contains(string(data), "type RenamedType struct") || !strings.Contains(string(data), "BetterName string") {
			t.Fatalf("binding overrides not reflected in generated code:\n%s", data)
		}
	})
}

func run(dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}
