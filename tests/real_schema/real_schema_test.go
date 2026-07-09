package real_schema

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/harald-mue/xsd-parser-go/internal/pipeline"
)

func realSchemaRoot() string {
	if root := os.Getenv("XSD_REAL_SCHEMA_ROOT"); root != "" {
		return root
	}
	return "schemas"
}

func schemaPath(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{realSchemaRoot()}, parts...)...)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("real schema not available at %s: %v", p, err)
	}
	return p
}

func TestRealSchemasGenerateAndCompile(t *testing.T) {
	cases := []struct {
		name    string
		schema  []string
		pkgName string
	}{
		{
			name:    "bpmn20",
			schema:  []string{"bpmn", "schema", "BPMN20.xsd"},
			pkgName: "bpmn",
		},
		{
			name:    "factur_x_minimum",
			schema:  []string{"factur_x", "schema", "0 1.07.2 MINIMUM", "Factur-X_1.07.2_MINIMUM.xsd"},
			pkgName: "factur",
		},
		{
			name:    "factur_x_basicwl",
			schema:  []string{"factur_x", "schema", "1 1.07.2 BASICWL", "Factur-X_1.07.2_BASICWL.xsd"},
			pkgName: "factur",
		},
		{
			name:    "factur_x_basic",
			schema:  []string{"factur_x", "schema", "2 1.07.2 BASIC", "Factur-X_1.07.2_BASIC.xsd"},
			pkgName: "factur",
		},
		{
			name:    "factur_x_en16931",
			schema:  []string{"factur_x", "schema", "3 1.07.2 EN16931", "Factur-X_1.07.2_EN16931.xsd"},
			pkgName: "factur",
		},
		{
			name:    "factur_x_extended",
			schema:  []string{"factur_x", "schema", "4 1.07.2 EXTENDED", "Factur-X_1.07.2_EXTENDED.xsd"},
			pkgName: "factur",
		},
		{
			name:    "factur_x_cii_d22b",
			schema:  []string{"factur_x", "schema", "5 CII D22B XSD", "CrossIndustryInvoice_100pD22B.xsd"},
			pkgName: "factur",
		},
		{
			name:    "bmecat_etim_310",
			schema:  []string{"bmecat_etim_310", "schema.xsd"},
			pkgName: "bmecat",
		},
		{
			name:    "bmecat_etim_501",
			schema:  []string{"bmecat_etim_501", "schema.xsd"},
			pkgName: "bmecat",
		},
		{
			name:    "xbrl_instance",
			schema:  []string{"vsme", "schema", "www.xbrl.org", "2003", "xbrl-instance-2003-12-31.xsd"},
			pkgName: "xbrl",
		},
		{
			name:    "xbrl_linkbase",
			schema:  []string{"vsme", "schema", "www.xbrl.org", "2003", "xbrl-linkbase-2003-12-31.xsd"},
			pkgName: "xbrl",
		},
		{
			name:    "vsme_all",
			schema:  []string{"vsme", "schema", "xbrl.efrag.org", "taxonomy", "vsme", "2025-07-30", "vsme-all.xsd"},
			pkgName: "vsme",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema := schemaPath(t, tc.schema...)
			dir := t.TempDir()
			if err := pipeline.Generate([]string{schema}, tc.pkgName, dir); err != nil {
				t.Fatalf("Generate returned error: %v", err)
			}
			if out, err := run(dir, "go", "mod", "init", "real/"+tc.name); err != nil {
				t.Fatalf("go mod init: %v\n%s", err, out)
			}
			if out, err := run(dir, "go", "test", "./..."); err != nil {
				t.Fatalf("go test: %v\n%s", err, out)
			}
		})
	}
}

func TestBPMNMinimalXMLSmoke(t *testing.T) {
	schema := schemaPath(t, "bpmn", "schema", "BPMN20.xsd")
	dir := t.TempDir()
	if err := pipeline.Generate([]string{schema}, "bpmn", dir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bpmn_smoke_test.go"), []byte(`package bpmn

import (
	"encoding/xml"
	"testing"
)

func TestMinimalDefinitionsXML(t *testing.T) {
	input := []byte("<definitions xmlns=\"http://www.omg.org/spec/BPMN/20100524/MODEL\" id=\"defs\" targetNamespace=\"urn:test\"></definitions>")
	var v Definitions
	if err := xml.Unmarshal(input, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, err := xml.Marshal(v); err != nil {
		t.Fatalf("marshal: %v", err)
	}
}
`), 0o644); err != nil {
		t.Fatalf("write smoke test: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "real/bpmn-smoke"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestFacturXXMLSmoke(t *testing.T) {
	cases := []struct {
		name       string
		schemaDir  string
		schemaFile string
		example    string
	}{
		{name: "minimum", schemaDir: "0 1.07.2 MINIMUM", schemaFile: "Factur-X_1.07.2_MINIMUM.xsd", example: "minimum.xml"},
		{name: "basicwl", schemaDir: "1 1.07.2 BASICWL", schemaFile: "Factur-X_1.07.2_BASICWL.xsd", example: "basicwl.xml"},
		{name: "basic", schemaDir: "2 1.07.2 BASIC", schemaFile: "Factur-X_1.07.2_BASIC.xsd", example: "basic.xml"},
		{name: "en16931", schemaDir: "3 1.07.2 EN16931", schemaFile: "Factur-X_1.07.2_EN16931.xsd", example: "en16931.xml"},
		{name: "extended", schemaDir: "4 1.07.2 EXTENDED", schemaFile: "Factur-X_1.07.2_EXTENDED.xsd", example: "extended.xml"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema := schemaPath(t, "factur_x", "schema", tc.schemaDir, tc.schemaFile)
			examplePath := schemaPath(t, "factur_x", "examples", tc.example)
			dir := t.TempDir()
			if err := pipeline.Generate([]string{schema}, "factur", dir); err != nil {
				t.Fatalf("Generate returned error: %v", err)
			}
			exampleData, err := os.ReadFile(examplePath)
			if err != nil {
				t.Fatalf("read %s: %v", tc.example, err)
			}
			if err := os.WriteFile(filepath.Join(dir, "input.xml"), exampleData, 0o644); err != nil {
				t.Fatalf("write input.xml: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, "factur_smoke_test.go"), []byte(`package factur

import (
	"encoding/xml"
	"os"
	"testing"
)

func TestCrossIndustryInvoiceXML(t *testing.T) {
	input, err := os.ReadFile("input.xml")
	if err != nil {
		t.Fatalf("read input.xml: %v", err)
	}
	var v CrossIndustryInvoice
	if err := xml.Unmarshal(input, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, err := xml.Marshal(v); err != nil {
		t.Fatalf("marshal: %v", err)
	}
}
`), 0o644); err != nil {
				t.Fatalf("write smoke test: %v", err)
			}
			if out, err := run(dir, "go", "mod", "init", "real/factur-"+tc.name+"-smoke"); err != nil {
				t.Fatalf("go mod init: %v\n%s", err, out)
			}
			if out, err := run(dir, "go", "test", "./..."); err != nil {
				t.Fatalf("go test: %v\n%s", err, out)
			}
		})
	}
}

func run(dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}
