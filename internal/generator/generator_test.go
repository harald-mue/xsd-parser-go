package generator_test

import (
	"strings"
	"testing"

	"github.com/harald-mue/xsd-parser-go/internal/generator"
	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/parser"
	"github.com/harald-mue/xsd-parser-go/internal/renderer"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func TestGenerateRendersComplexTypeFields(t *testing.T) {
	input := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns="urn:test"
           targetNamespace="urn:test"
           elementFormDefault="qualified">
  <xs:element name="document" type="DocumentType"/>
  <xs:complexType name="DocumentType">
    <xs:sequence>
      <xs:element name="title" type="xs:string"/>
      <xs:element name="count" type="xs:int" minOccurs="0"/>
      <xs:element name="item" type="ItemType" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
    <xs:attribute name="id" type="xs:string" use="required"/>
  </xs:complexType>
  <xs:complexType name="ItemType">
    <xs:sequence>
      <xs:element name="name" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)

	file, err := parser.Parse("test.xsd", input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schema.Schemas{Files: []schema.File{file}})
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type Document DocumentType")
	assertLineContains(t, src, "Title", "string", "`xml:\"urn:test title\"`")
	assertLineContains(t, src, "Count", "*int", "`xml:\"urn:test count,omitempty\"`")
	assertLineContains(t, src, "Item", "[]ItemType", "`xml:\"urn:test item,omitempty\"`")
	assertLineContains(t, src, "Id", "string", "`xml:\"id,attr\"`")
	assertLineContains(t, src, "Name", "string", "`xml:\"urn:test name\"`")
}

func TestGeneratePrefersGlobalElementNameOnTypeCollision(t *testing.T) {
	input := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:tns="urn:test"
           targetNamespace="urn:test"
           elementFormDefault="qualified">
  <xs:element name="supportedProtocols" type="tns:SupportedProtocols"/>
  <xs:complexType name="SupportedProtocols">
    <xs:sequence>
      <xs:element name="protocols" type="xs:anyURI" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)

	file, err := parser.Parse("test.xsd", input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schema.Schemas{Files: []schema.File{file}})
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type SupportedProtocols SupportedProtocolsType")
	assertContains(t, src, "func (t SupportedProtocols) MarshalXML")
	assertContains(t, src, "supportedProtocols")
	assertNotContains(t, src, "type SupportedProtocols2")
}

func TestGenerateEnumConstantsSanitizePunctuationAndAvoidNumericSuffixCollisions(t *testing.T) {
	input := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="urn:test"
           elementFormDefault="qualified">
  <xs:simpleType name="TimeZoneEnum">
    <xs:restriction base="xs:string">
      <xs:enumeration value="Etc/GMT+12"/>
      <xs:enumeration value="Etc/GMT-1"/>
      <xs:enumeration value="Europe/Vienna"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`)

	file, err := parser.Parse("test.xsd", input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schema.Schemas{Files: []schema.File{file}})
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "TimeZoneEnumEtcGMT12")
	assertContains(t, src, "TimeZoneEnumEtcGMT1")
	assertContains(t, src, "TimeZoneEnumEuropeVienna")
	assertNotContains(t, src, "TimeZoneEnumEurope/Vienna")
}

func TestGenerateLocalAbstractEmptyBaseAsPolymorphicInterface(t *testing.T) {
	input := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:tns="urn:test"
           targetNamespace="urn:test"
           elementFormDefault="qualified">
  <xs:complexType name="DeploymentIdentifierBase" abstract="true"/>
  <xs:complexType name="DeploymentIdIdentifier">
    <xs:complexContent>
      <xs:extension base="tns:DeploymentIdentifierBase">
        <xs:sequence>
          <xs:element name="DeploymentId" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="ArtifactIdentifier">
    <xs:complexContent>
      <xs:extension base="tns:DeploymentIdentifierBase">
        <xs:sequence>
          <xs:element name="ArtifactName" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="Status">
    <xs:sequence>
      <xs:element name="DeploymentIdentifier" type="tns:DeploymentIdentifierBase"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)

	file, err := parser.Parse("test.xsd", input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schema.Schemas{Files: []schema.File{file}})
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "DeploymentIdentifier DeploymentIdentifier `xml:\"-\"`")
	assertContains(t, src, "type DeploymentIdentifier interface")
	assertContains(t, src, "IsDeploymentIdentifier()")
	assertContains(t, src, "var DeploymentIdentifierRegistry")
	assertContains(t, src, `QName{Namespace: "urn:test", Local: "DeploymentIdIdentifier"}`)
	assertContains(t, src, `Element: QName{Namespace: "urn:test", Local: "DeploymentIdentifier"}`)
	assertContains(t, src, "func (*DeploymentIdIdentifier) IsDeploymentIdentifier() {}")
	assertContains(t, src, "func (*ArtifactIdentifier) IsDeploymentIdentifier() {}")
	assertContains(t, src, "xsi:type")
}

func TestGenerateKeepsCrossNamespaceElementAliasesOutOfTypeReferences(t *testing.T) {
	dataSchema := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/data"
           elementFormDefault="qualified">
  <xs:complexType name="Artifact">
    <xs:sequence>
      <xs:element name="name" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)
	definitionSchema := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:data="http://example.com/data"
           targetNamespace="http://example.com/definition"
           elementFormDefault="qualified">
  <xs:element name="Artifact" type="data:Artifact"/>
</xs:schema>`)
	commandSchema := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:data="http://example.com/data"
           targetNamespace="http://example.com/command"
           elementFormDefault="qualified">
  <xs:complexType name="Request">
    <xs:sequence>
      <xs:element name="Artifact" type="data:Artifact"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)

	dataFile, err := parser.Parse("data.xsd", dataSchema)
	if err != nil {
		t.Fatalf("Parse data returned error: %v", err)
	}
	definitionFile, err := parser.Parse("definition.xsd", definitionSchema)
	if err != nil {
		t.Fatalf("Parse definition returned error: %v", err)
	}
	commandFile, err := parser.Parse("command.xsd", commandSchema)
	if err != nil {
		t.Fatalf("Parse command returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schema.Schemas{Files: []schema.File{dataFile, definitionFile, commandFile}})
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.GenerateWithOptions(meta, generator.Options{
		Package:    "model",
		ModulePath: "example.com/model",
		NamespacePackages: map[string]string{
			"http://example.com/data":       "data",
			"http://example.com/definition": "definition",
			"http://example.com/command":    "command",
		},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	commandSrc := string(files["command/models.go"])
	assertContains(t, commandSrc, "Artifact data.ExampleArtifact")
	assertNotContains(t, commandSrc, "data.ExampleArtifact2")
}

func TestGenerateEnumerationConstantsFromFixture(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/enumeration/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type EnumType string")
	assertLineContains(t, src, "Enum", "EnumType", "`xml:\"http://example.com Enum\"`")
	assertContains(t, src, "EnumTypeOFF  EnumType = \"OFF\"")
	assertContains(t, src, "EnumTypeON   EnumType = \"ON\"")
	assertContains(t, src, "EnumTypeAUTO EnumType = \"AUTO\"")
}

func TestGenerateSimpleContentFromFixture(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/simple_content/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type Foo FooType")
	assertContains(t, src, "type EnumType string")
	assertContains(t, src, "EnumTypeOFF  EnumType = \"OFF\"")
	assertContains(t, src, "EnumTypeON   EnumType = \"ON\"")
	assertContains(t, src, "EnumTypeAUTO EnumType = \"AUTO\"")
	assertLineContains(t, src, "Value", "*string", "`xml:\"value,attr,omitempty\"`")
	assertLineContains(t, src, "Content", "EnumType", "`xml:\",chardata\"`")
}

func TestGenerateExtensionSimpleContentFromFixture(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/extension_simple_content/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type Foo FooType")
	assertContains(t, src, "type BaseType struct")
	assertLineContains(t, src, "Value", "*string", "`xml:\"value,attr,omitempty\"`")
	assertLineContains(t, src, "AnotherValue", "*string", "`xml:\"anotherValue,attr,omitempty\"`")
	assertLineContains(t, src, "Content", "EnumType", "`xml:\",chardata\"`")
}

func TestGenerateSimpleContentRestrictionFromFixture(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/simple_content_with_extension/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type SupplierId SupplierIdType")
	assertContains(t, src, "type UnitName UnitNameType")
	assertContains(t, src, "type ExtendedString struct")
	assertLineContains(t, src, "Type", "*string", "`xml:\"type,attr,omitempty\"`")
	assertLineContains(t, src, "Lang", "*string", "`xml:\"lang,attr,omitempty\"`")
	assertLineContains(t, src, "Content", "UnitNameTypeContentType", "`xml:\",chardata\"`")
	assertContains(t, src, "type UnitNameTypeContentType CustomString")
	assertContains(t, src, "UnitNameTypeContentTypeUnit1 UnitNameTypeContentType = \"Unit1\"")
	assertContains(t, src, "UnitNameTypeContentTypeUnit2 UnitNameTypeContentType = \"Unit2\"")
	assertContains(t, src, "UnitNameTypeContentTypeUnit3 UnitNameTypeContentType = \"Unit3\"")
}

func TestGenerateChoiceMembersUseContentModelFromFixture(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/choice/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertLineContains(t, src, "Content", "FooTypeContent", "`xml:\"-\"`")
	assertContains(t, src, "type FooTypeContent interface")
	assertContains(t, src, "type FooTypeContentBar struct")
	assertContains(t, src, "Value BarType")
	assertContains(t, src, "type FooTypeContentBaz struct")
	assertContains(t, src, "Value BazType")
	assertContains(t, src, "func (t *FooType) UnmarshalXML")
	assertContains(t, src, "func (t FooType) MarshalXML")
}

func TestGenerateAllCardinalityFromFixture(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/all/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertLineContains(t, src, "Once", "int", "`xml:\"http://example.com Once\"`")
	assertLineContains(t, src, "Optional", "*int", "`xml:\"http://example.com Optional,omitempty\"`")
	assertLineContains(t, src, "OnceSpecify", "int", "`xml:\"http://example.com OnceSpecify\"`")
	assertLineContains(t, src, "TwiceOrMore", "[]int", "`xml:\"http://example.com TwiceOrMore,omitempty\"`")
}

func TestGenerateGroupRefInlinedFromFixture(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/complex_type_with_group/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	// The Outer group is referenced twice; both occurrences are inlined.
	assertLineContains(t, src, "Bar", "*string", "`xml:\"http://example.com Bar,omitempty\"`")
	assertLineContains(t, src, "Bar2", "*string", "`xml:\"http://example.com Bar,omitempty\"`")
	assertLineContains(t, src, "Fizz", "*string", "`xml:\"http://example.com Fizz,omitempty\"`")
	assertLineContains(t, src, "Buzz", "*int", "`xml:\"http://example.com Buzz,omitempty\"`")
}

func TestGenerateElementRefsWithNamespacesFromFixture(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/element_refs_with_ns/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	if got, want := len(schemas.Files), 4; got != want {
		t.Fatalf("len(Files) = %d, want %d", got, want)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type Outer OuterType")
	assertLineContains(t, src, "BarInner", "BarInnerType", "`xml:\"Bar Inner\"`")
	assertLineContains(t, src, "BazInner", "BazInnerType", "`xml:\"Baz Inner\"`")
	assertLineContains(t, src, "BizInner", "BizInnerType", "`xml:\"Biz Inner\"`")
	assertContains(t, src, "type BarInner BarInnerType")
	assertContains(t, src, "type BazInner BazInnerType")
	assertContains(t, src, "type BizInner BizInnerType")
}

func TestGenerateAttributeRefFromFixture(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/ref_to_attribute/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type Foo FooType")
	assertLineContains(t, src, "Id", "*string", "`xml:\"id,attr,omitempty\"`")
}

func TestGenerateDisambiguatesSameLocalTypeNamesAcrossNamespaces(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/name_collision/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type AAddressType struct")
	assertContains(t, src, "type BAddressType struct")
	assertLineContains(t, src, "HomeAddress", "AAddressType", "`xml:\"http://root.example.com homeAddress\"`")
	assertLineContains(t, src, "WorkAddress", "BAddressType", "`xml:\"http://root.example.com workAddress\"`")
}

func TestGenerateDisambiguatesCaseClashingTypeNamesFromFixture(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/type_name_clash/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type Foo FooType")
	assertContains(t, src, "type FooType struct")
	assertContains(t, src, "type FooType2 struct")
	assertLineContains(t, src, "A", "*string", "`xml:\"a,attr,omitempty\"`")
	assertLineContains(t, src, "B", "*string", "`xml:\"b,attr,omitempty\"`")
}

func TestGenerateImportedTypeReferenceFromFixture(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/extension_base_two_files/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	if got, want := len(schemas.Files), 2; got != want {
		t.Fatalf("len(Files) = %d, want %d", got, want)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type Foo FooType")
	assertLineContains(t, src, "A", "float32", "`xml:\"http://example.com a\"`")
	assertLineContains(t, src, "B", "BarType", "`xml:\"http://example.com b\"`")
	assertLineContains(t, src, "B", "int", "`xml:\"http://other.example.com b\"`")
	assertLineContains(t, src, "C", "string", "`xml:\"http://other.example.com c\"`")
}

func TestGenerateImportedComplexContentExtension(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/imported_extension_base/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	if got, want := len(schemas.Files), 2; got != want {
		t.Fatalf("len(Files) = %d, want %d", got, want)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type Foo FooType")
	assertLineContains(t, src, "B", "int", "`xml:\"http://base.example.com b\"`")
	assertLineContains(t, src, "C", "string", "`xml:\"http://base.example.com c\"`")
	assertLineContains(t, src, "A", "float32", "`xml:\"http://example.com a\"`")
}

func TestGenerateComplexContentExtension(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"../../tests/fixtures/extension_base/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret returned error: %v", err)
	}
	pkg, err := generator.Generate(meta, "model")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	files, err := renderer.RenderPackage(pkg)
	if err != nil {
		t.Fatalf("RenderPackage returned error: %v", err)
	}

	src := string(files["models.go"])
	assertContains(t, src, "type Foo FooType")
	assertLineContains(t, src, "B", "int", "`xml:\"http://example.com b\"`")
	assertLineContains(t, src, "C", "string", "`xml:\"http://example.com c\"`")
	assertLineContains(t, src, "A", "float32", "`xml:\"http://example.com a\"`")
}

func assertContains(t *testing.T, src string, want string) {
	t.Helper()
	if !strings.Contains(src, want) {
		t.Fatalf("generated source does not contain %q:\n%s", want, src)
	}
}

func assertNotContains(t *testing.T, src string, want string) {
	t.Helper()
	if strings.Contains(src, want) {
		t.Fatalf("generated source unexpectedly contains %q:\n%s", want, src)
	}
}

func assertLineContains(t *testing.T, src string, parts ...string) {
	t.Helper()
	for _, line := range strings.Split(src, "\n") {
		matched := true
		for _, part := range parts {
			if !strings.Contains(line, part) {
				matched = false
				break
			}
		}
		if matched {
			return
		}
	}
	t.Fatalf("generated source has no line containing %q:\n%s", parts, src)
}
