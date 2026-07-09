package prefix_marshaling

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/harald-mue/xsd-parser-go/internal/pipeline"
)

func TestPrefixControlledMarshalForStructTags(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "prefix_marshaling", "schema.xsd")
	if err := pipeline.GenerateWithOptions([]string{schema}, pipeline.GenerateOptions{
		Package:           "model",
		OutDir:            dir,
		NamespacePrefixes: map[string]string{"http://example.com/prefix": "p"},
	}); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "prefix_test.go"), []byte(`package model

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestPrefixMarshal(t *testing.T) {
	value := Root{Code: "x", Child: "hello", Item: []int{1}}
	out, err := xml.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	xml := string(out)
	for _, want := range []string{
		"<p:Root",
		"xmlns:p=\"http://example.com/prefix\"",
		"<p:Child",
		"<p:Item",
		"code=\"x\"",
	} {
		if !strings.Contains(xml, want) {
			t.Fatalf("marshaled XML missing %s: %s", want, xml)
		}
	}
}
`), 0o644); err != nil {
		t.Fatalf("write prefix_test.go: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "prefixmarshal"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func run(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
