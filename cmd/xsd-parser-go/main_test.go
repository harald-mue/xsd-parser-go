package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateDefaultOutputFile(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "..", "examples", "minimal", "schema.xsd")
	if err := run([]string{
		"generate",
		"--package", "model",
		"--out", dir,
		schema,
	}); err != nil {
		t.Fatalf("run generate returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "models.go")); err != nil {
		t.Fatalf("models.go was not generated: %v", err)
	}
}

func TestConvertAutoDetectsNamespacePackages(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "..", "tests", "fixtures", "imported_type_element", "schema.xsd")
	if err := run([]string{
		"convert",
		schema,
		"tmp/legacy",
		dir,
	}); err != nil {
		t.Fatalf("run convert returned error: %v", err)
	}
	monitor, err := os.ReadFile(filepath.Join(dir, "monitor", "models.go"))
	if err != nil {
		t.Fatalf("read monitor models.go: %v", err)
	}
	if !strings.Contains(string(monitor), "data.Incident") {
		t.Fatalf("generated monitor model does not import/use data.Incident")
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "models.go")); err != nil {
		t.Fatalf("data/models.go was not generated: %v", err)
	}
}

func TestGenerateOutputFileExplicit(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "..", "examples", "minimal", "schema.xsd")
	if err := run([]string{
		"generate",
		"--output-file", "model_gen.go",
		"--package", "model",
		"--out", dir,
		schema,
	}); err != nil {
		t.Fatalf("run generate returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "model_gen.go")); err != nil {
		t.Fatalf("model_gen.go was not generated: %v", err)
	}
}
