package validation

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harald-mue/xsd-parser-go/internal/pipeline"
)

func TestPresenceTrackingWithoutValidateOnUnmarshal(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "presence_tracking", "schema.xsd")
	if err := pipeline.Generate([]string{schema}, "model", dir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "presence_default_test.go"), []byte(`package model

import (
	"encoding/xml"
	"testing"
)

func TestRequiredScalarPresenceOnDefaultUnmarshal(t *testing.T) {
	var ok RequiredScalarType
	if err := xml.Unmarshal([]byte("<RequiredScalar xmlns=\"http://example.com/presence\" code=\"A\"><Name></Name><Count>0</Count></RequiredScalar>"), &ok); err != nil {
		t.Fatalf("valid zero values rejected: %v", err)
	}
	var missingName RequiredScalarType
	if err := xml.Unmarshal([]byte("<RequiredScalar xmlns=\"http://example.com/presence\" code=\"A\"><Count>0</Count></RequiredScalar>"), &missingName); err == nil {
		t.Fatalf("missing required scalar element accepted")
	}
	var missingAttr RequiredScalarType
	if err := xml.Unmarshal([]byte("<RequiredScalar xmlns=\"http://example.com/presence\"><Name>x</Name><Count>1</Count></RequiredScalar>"), &missingAttr); err == nil {
		t.Fatalf("missing required scalar attribute accepted")
	}
}
`), 0o644); err != nil {
		t.Fatalf("write presence_default_test.go: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "presencetrackingdefault"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestValidateOnUnmarshalTracksRequiredScalarPresence(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "presence_tracking", "schema.xsd")
	if err := pipeline.GenerateWithOptions([]string{schema}, pipeline.GenerateOptions{Package: "model", OutDir: dir, ValidateOnUnmarshal: true}); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "presence_test.go"), []byte(`package model

import (
	"encoding/xml"
	"testing"
)

func TestRequiredScalarPresence(t *testing.T) {
	var ok RequiredScalarType
	if err := xml.Unmarshal([]byte("<RequiredScalar xmlns=\"http://example.com/presence\" code=\"A\"><Name></Name><Count>0</Count></RequiredScalar>"), &ok); err != nil {
		t.Fatalf("valid zero values rejected: %v", err)
	}
	var missingName RequiredScalarType
	if err := xml.Unmarshal([]byte("<RequiredScalar xmlns=\"http://example.com/presence\" code=\"A\"><Count>0</Count></RequiredScalar>"), &missingName); err == nil {
		t.Fatalf("missing required scalar element accepted")
	}
	var missingAttr RequiredScalarType
	if err := xml.Unmarshal([]byte("<RequiredScalar xmlns=\"http://example.com/presence\"><Name>x</Name><Count>1</Count></RequiredScalar>"), &missingAttr); err == nil {
		t.Fatalf("missing required scalar attribute accepted")
	}
}
`), 0o644); err != nil {
		t.Fatalf("write presence_test.go: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "presencetracking"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestValidateOnUnmarshalCoversCustomXMLAndChoice(t *testing.T) {
	choiceDir := t.TempDir()
	choiceSchema := filepath.Join("..", "fixtures", "strict_validation", "schema.xsd")
	if err := pipeline.GenerateWithOptions([]string{choiceSchema}, pipeline.GenerateOptions{Package: "model", OutDir: choiceDir, ValidateOnUnmarshal: true}); err != nil {
		t.Fatalf("Generate choice returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(choiceDir, "validate_on_unmarshal_test.go"), []byte(`package model

import (
	"encoding/xml"
	"testing"
)

func TestChoiceValidateOnUnmarshal(t *testing.T) {
	var v ChoiceType
	if err := xml.Unmarshal([]byte("<Choice xmlns=\"http://example.com\"></Choice>"), &v); err == nil {
		t.Fatalf("missing required choice accepted on unmarshal")
	}
}
`), 0o644); err != nil {
		t.Fatalf("write choice validate test: %v", err)
	}
	if out, err := run(choiceDir, "go", "mod", "init", "validatechoice"); err != nil {
		t.Fatalf("go mod init choice: %v\n%s", err, out)
	}
	if out, err := run(choiceDir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test choice: %v\n%s", err, out)
	}

	customDir := t.TempDir()
	customSchema := filepath.Join("..", "fixtures", "polymorphic_multi_field", "schema.xsd")
	if err := pipeline.GenerateWithOptions([]string{customSchema}, pipeline.GenerateOptions{Package: "model", OutDir: customDir, ValidateOnUnmarshal: true}); err != nil {
		t.Fatalf("Generate custom returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(customDir, "validate_on_unmarshal_test.go"), []byte(`package model

import (
	"encoding/xml"
	"testing"
)

func TestCustomXMLValidateOnUnmarshal(t *testing.T) {
	var v ContainerType
	if err := xml.Unmarshal([]byte("<container xmlns=\"http://example.com\"><dog id=\"1\" name=\"Rex\"/></container>"), &v); err == nil {
		t.Fatalf("missing required polymorphic vehicle accepted on unmarshal")
	}
}
`), 0o644); err != nil {
		t.Fatalf("write custom validate test: %v", err)
	}
	if out, err := run(customDir, "go", "mod", "init", "validatecustom"); err != nil {
		t.Fatalf("go mod init custom: %v\n%s", err, out)
	}
	if out, err := run(customDir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test custom: %v\n%s", err, out)
	}
}

func TestAnyNamespaceValidation(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "any_namespace", "schema.xsd")
	if err := pipeline.Generate([]string{schema}, "model", dir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "any_namespace_test.go"), []byte(`package model

import (
	"encoding/xml"
	"testing"
)

func TestAnyNamespaceValidate(t *testing.T) {
	valid := AnyNamespaceType{
		Any: []AnyElement{{XMLName: xml.Name{Space: "http://example.com/ext", Local: "item"}}},
		Any2: &AnyElement{XMLName: xml.Name{Local: "local"}},
		AnyAttribute: AnyAttributes{{Name: xml.Name{Space: "http://example.com/ext", Local: "code"}, Value: "x"}},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid wildcard rejected: %v", err)
	}
	badElement := valid
	badElement.Any = []AnyElement{{XMLName: xml.Name{Space: "http://example.com/base", Local: "item"}}}
	if err := badElement.Validate(); err == nil {
		t.Fatalf("target namespace element accepted for ##other")
	}
	badSecond := valid
	badSecond.Any2 = &AnyElement{XMLName: xml.Name{Space: "http://forbidden.example", Local: "x"}}
	if err := badSecond.Validate(); err == nil {
		t.Fatalf("forbidden namespace accepted for ##local/ext wildcard")
	}
	badAttr := valid
	badAttr.AnyAttribute = AnyAttributes{{Name: xml.Name{Space: "http://example.com/base", Local: "code"}, Value: "x"}}
	if err := badAttr.Validate(); err == nil {
		t.Fatalf("target namespace attribute accepted for ##other")
	}
}
`), 0o644); err != nil {
		t.Fatalf("write any_namespace_test.go: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "anynamespacevalidation"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestAnyStrictProcessContentsValidation(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "any_strict", "schema.xsd")
	if err := pipeline.Generate([]string{schema}, "model", dir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "any_strict_test.go"), []byte(`package model

import (
	"encoding/xml"
	"testing"
)

func TestAnyStrictProcessContentsValidate(t *testing.T) {
	valid := ContainerType{
		Name: "example",
		Any: []AnyElement{
			{XMLName: xml.Name{Space: "http://example.com", Local: "Known"}},
			{XMLName: xml.Name{Space: "http://example.com", Local: "Extension"}},
		},
		AnyAttribute: AnyAttributes{{Name: xml.Name{Space: "http://example.com", Local: "KnownAttr"}, Value: "x"}},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid strict wildcard rejected: %v", err)
	}
	badElement := valid
	badElement.Any = []AnyElement{{XMLName: xml.Name{Space: "http://example.com", Local: "Unknown"}}}
	if err := badElement.Validate(); err == nil {
		t.Fatalf("undeclared strict element accepted")
	}
	badAttr := valid
	badAttr.AnyAttribute = AnyAttributes{{Name: xml.Name{Space: "http://example.com", Local: "UnknownAttr"}, Value: "x"}}
	if err := badAttr.Validate(); err == nil {
		t.Fatalf("undeclared strict attribute accepted")
	}
}
`), 0o644); err != nil {
		t.Fatalf("write any_strict_test.go: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "anystrictvalidation"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestXSDScalarTypesAndDocumentation(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "xsd_scalar_types", "schema.xsd")
	if err := pipeline.Generate([]string{schema}, "model", dir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "models.go"))
	if err != nil {
		t.Fatalf("read generated model: %v", err)
	}
	for _, want := range []string{"// ScalarHolderType contains representative XSD scalar values.", "// Created is the creation timestamp.", "// AmountType is an exact decimal amount."} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("generated docs missing %q", want)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "scalar_test.go"), []byte(`package model

import (
	"encoding/xml"
	"testing"
)

func TestXSDScalarHelpers(t *testing.T) {
	input := []byte("<ScalarHolder xmlns=\"http://example.com/scalars\"><Created>2024-01-02T03:04:05Z</Created><Birthday>2024-01-02</Birthday><Clock>03:04:05Z</Clock><Period>P1Y2M3DT4H5M6S</Period><Amount>1234.56</Amount><Payload>AQID</Payload><Digest>0a0b0c</Digest><Year>2024</Year></ScalarHolder>")
	var v ScalarHolderType
	if err := xml.Unmarshal(input, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, err := v.Created.Time(); err != nil {
		t.Fatalf("created Time(): %v", err)
	}
	if _, err := v.Birthday.Time(); err != nil {
		t.Fatalf("birthday Time(): %v", err)
	}
	if rat, err := XSDDecimal(v.Amount).Rat(); err != nil || rat.String() != "30864/25" {
		t.Fatalf("amount Rat() = %v, %v", rat, err)
	}
	if got := []byte(v.Payload); len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("payload decoded = %#v", got)
	}
	if got := []byte(v.Digest); len(got) != 3 || got[0] != 0x0a || got[1] != 0x0b || got[2] != 0x0c {
		t.Fatalf("digest decoded = %#v", got)
	}
	var badDuration XSDDuration
	if err := badDuration.UnmarshalText([]byte("not-a-duration")); err == nil {
		t.Fatalf("invalid duration accepted")
	}
}
`), 0o644); err != nil {
		t.Fatalf("write scalar_test.go: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "xsdscalartypes"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestDefaultFixedAndAdditionalFacets(t *testing.T) {
	dir := t.TempDir()
	defaultSchema := filepath.Join("..", "fixtures", "default_fixed", "schema.xsd")
	facetSchema := filepath.Join("..", "fixtures", "facet_digits_whitespace", "schema.xsd")
	if err := pipeline.Generate([]string{defaultSchema, facetSchema}, "model", dir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "default_fixed_facets_test.go"), []byte(`package model

import (
	"encoding/xml"
	"testing"
)

func TestDefaultsAndFixedValues(t *testing.T) {
	var cfg ConfigType
	input := []byte("<Config xmlns=\"http://example.com/default-fixed\" category=\"general\"><Kind>standard</Kind></Config>")
	if err := xml.Unmarshal(input, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.Mode == nil || *cfg.Mode != "auto" {
		t.Fatalf("Mode default = %#v, want auto", cfg.Mode)
	}
	if cfg.Status == nil || *cfg.Status != "new" {
		t.Fatalf("Status default = %#v, want new", cfg.Status)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid fixed values rejected: %v", err)
	}
	if cfg.Category == nil || *cfg.Category != "general" {
		t.Fatalf("Category fixed default = %#v, want general", cfg.Category)
	}
	bad := cfg
	bad.Category = ptrString("other")
	if err := bad.Validate(); err == nil {
		t.Fatalf("invalid fixed attribute accepted")
	}
}

func TestAdditionalFacets(t *testing.T) {
	if err := CollapsedCode("ABC").Validate(); err != nil {
		t.Fatalf("collapsed code rejected: %v", err)
	}
	if err := CollapsedCode("A  B").Validate(); err == nil {
		t.Fatalf("non-collapsed whitespace accepted")
	}
	if err := AmountType("123.45").Validate(); err != nil {
		t.Fatalf("valid amount rejected: %v", err)
	}
	if err := AmountType("1234.56").Validate(); err == nil {
		t.Fatalf("totalDigits violation accepted")
	}
	if err := AmountType("12.345").Validate(); err == nil {
		t.Fatalf("fractionDigits violation accepted")
	}
}

func ptrString(v string) *string { return &v }
`), 0o644); err != nil {
		t.Fatalf("write default_fixed_facets_test.go: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "defaultfixedfacets"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestAnyStrictLocalDeclarationValidation(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "any_strict_local", "schema.xsd")
	if err := pipeline.Generate([]string{schema}, "model", dir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "any_strict_local_test.go"), []byte(`package model

import (
	"encoding/xml"
	"testing"
)

func TestLocalStrictDeclarationsValidate(t *testing.T) {
	valid := ContainerType{
		KnownLocal: "x",
		Any: []AnyElement{{XMLName: xml.Name{Space: "http://example.com/local", Local: "KnownLocal"}}},
		AnyAttribute: AnyAttributes{{Name: xml.Name{Local: "knownLocalAttr"}, Value: "x"}},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid local declarations rejected: %v", err)
	}
	bad := valid
	bad.Any = []AnyElement{{XMLName: xml.Name{Space: "http://example.com/local", Local: "UnknownLocal"}}}
	if err := bad.Validate(); err == nil {
		t.Fatalf("unknown local strict wildcard element accepted")
	}
}
`), 0o644); err != nil {
		t.Fatalf("write any_strict_local_test.go: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "anystrictlocalvalidation"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestStrictValidationRequiredAndOccurs(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "strict_validation", "schema.xsd")
	if err := pipeline.Generate([]string{schema}, "model", dir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "strict_test.go"), []byte(`package model

import "testing"

func TestStrictValidate(t *testing.T) {
	if err := (StrictType{Item: []string{"one"}}).Validate(); err != nil {
		t.Fatalf("valid strict rejected: %v", err)
	}
	if err := (StrictType{}).Validate(); err == nil {
		t.Fatalf("missing required repeated item accepted")
	}
	if err := (StrictType{Item: []string{"one", "two", "three"}}).Validate(); err == nil {
		t.Fatalf("too many repeated items accepted")
	}
	if err := (ChoiceType{}).Validate(); err == nil {
		t.Fatalf("missing required choice accepted")
	}
	if err := (ChoiceType{Content: &ChoiceTypeContentA{Value: "ok"}}).Validate(); err != nil {
		t.Fatalf("valid choice rejected: %v", err)
	}
}
`), 0o644); err != nil {
		t.Fatalf("write strict_test.go: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "strictvalidation"); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}
	if out, err := run(dir, "go", "test", "./..."); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestStructLevelValidateCallsFieldValidation(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join("..", "fixtures", "facet_validation", "schema.xsd")
	if err := pipeline.Generate([]string{schema}, "model", dir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "validate_test.go"), []byte(`package model

import "testing"

func TestStructValidate(t *testing.T) {
	valid := ItemType{Code: CodeType("ABC"), Size: SizeType(5), Color: ColorType("red")}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid item rejected: %v", err)
	}
	invalid := ItemType{Code: CodeType("x"), Size: SizeType(5), Color: ColorType("red")}
	if err := invalid.Validate(); err == nil {
		t.Fatalf("invalid item accepted")
	}
}
`), 0o644); err != nil {
		t.Fatalf("write validate_test.go: %v", err)
	}
	if out, err := run(dir, "go", "mod", "init", "validation"); err != nil {
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
