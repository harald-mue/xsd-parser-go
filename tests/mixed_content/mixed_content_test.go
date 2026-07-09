package mixed_content

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/harald-mue/xsd-parser-go/internal/pipeline"
)

// Mixed content cannot be round-tripped with xml.MarshalIndent because the
// indentation whitespace becomes significant text content. This test compiles
// the generated model and verifies an order-preserving unmarshal/marshal cycle
// using non-indented marshalling.
func TestMixedContentOrderPreserving(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "mixed_content", "schema.xsd")
	if err := pipeline.Generate([]string{schema}, "model", dir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "input.xml"), []byte(`<tns:Para xmlns:tns="http://example.com">Hello <tns:Em>world</tns:Em>!</tns:Para>`), 0o644); err != nil {
		t.Fatalf("write input.xml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mixed_test.go"), []byte(mixedTestSource()), 0o644); err != nil {
		t.Fatalf("write test: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "mixed"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func mixedTestSource() string {
	return `package model

import (
	"encoding/xml"
	"os"
	"reflect"
	"testing"
)

func TestMixedContentRoundTrip(t *testing.T) {
	in, err := os.ReadFile("input.xml")
	if err != nil {
		t.Fatal(err)
	}
	var v ParaType
	if err := xml.Unmarshal(in, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(v.Content) != 3 {
		t.Fatalf("Content tokens = %d, want 3: %#v", len(v.Content), v.Content)
	}
	if v.Content[0].Text != "Hello " {
		t.Fatalf("first text node = %q, want %q", v.Content[0].Text, "Hello ")
	}
	if v.Content[2].Text != "!" {
		t.Fatalf("last text node = %q, want %q", v.Content[2].Text, "!")
	}
	if _, ok := v.Content[1].Element.(*ParaTypeContentEm); !ok {
		t.Fatalf("middle token is not Em element: %#v", v.Content[1])
	}
	out, err := xml.Marshal(&v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var v2 ParaType
	if err := xml.Unmarshal(out, &v2); err != nil {
		t.Fatalf("re-unmarshal: %v\n%s", err, out)
	}
	if !reflect.DeepEqual(v, v2) {
		t.Fatalf("round-trip mismatch:\nv1=%#v\nv2=%#v\nout=%s", v, v2, out)
	}
}
`
}

func run(dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}
