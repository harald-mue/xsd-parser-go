package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func TestParseTopLevelDeclarations(t *testing.T) {
	input := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:test">
  <xs:element name="document" type="DocumentType"/>
  <xs:complexType name="DocumentType"/>
  <xs:simpleType name="CodeType"/>
</xs:schema>`)

	file, err := Parse("test.xsd", input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if file.TargetNamespace != "urn:test" {
		t.Fatalf("TargetNamespace = %q, want urn:test", file.TargetNamespace)
	}
	if got, want := len(file.Elements), 1; got != want {
		t.Fatalf("len(Elements) = %d, want %d", got, want)
	}
	if got, want := len(file.ComplexTypes), 1; got != want {
		t.Fatalf("len(ComplexTypes) = %d, want %d", got, want)
	}
	if got, want := len(file.SimpleTypes), 1; got != want {
		t.Fatalf("len(SimpleTypes) = %d, want %d", got, want)
	}
}

func TestParseNestedComplexTypeContent(t *testing.T) {
	input := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns="urn:test"
           targetNamespace="urn:test"
           elementFormDefault="qualified">
  <xs:complexType name="DocumentType">
    <xs:sequence>
      <xs:element name="title" type="xs:string"/>
      <xs:element name="count" type="xs:int" minOccurs="0"/>
      <xs:element name="item" type="ItemType" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
    <xs:attribute name="id" type="xs:string" use="required"/>
  </xs:complexType>
</xs:schema>`)

	file, err := Parse("test.xsd", input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if got, want := file.ElementFormDefault, "qualified"; got != want {
		t.Fatalf("ElementFormDefault = %q, want %q", got, want)
	}
	if got, want := len(file.ComplexTypes), 1; got != want {
		t.Fatalf("len(ComplexTypes) = %d, want %d", got, want)
	}

	documentType := file.ComplexTypes[0]
	if documentType.QName.Namespace != "urn:test" || documentType.QName.Local != "DocumentType" {
		t.Fatalf("complex type QName = %#v", documentType.QName)
	}
	if got, want := len(documentType.Content), 1; got != want {
		t.Fatalf("len(Content) = %d, want %d", got, want)
	}
	sequence := documentType.Content[0].Group
	if sequence == nil {
		t.Fatalf("Content[0].Group is nil")
	}
	if got, want := sequence.Kind, schema.GroupSequence; got != want {
		t.Fatalf("sequence.Kind = %q, want %q", got, want)
	}
	if got, want := len(sequence.Particles), 3; got != want {
		t.Fatalf("len(sequence.Particles) = %d, want %d", got, want)
	}

	title := sequence.Particles[0].Element
	if title == nil {
		t.Fatalf("first sequence particle element is nil")
	}
	if got, want := title.TypeName.Namespace, xsdNamespace; got != want {
		t.Fatalf("title.TypeName.Namespace = %q, want %q", got, want)
	}
	if got, want := title.TypeName.Local, "string"; got != want {
		t.Fatalf("title.TypeName.Local = %q, want %q", got, want)
	}

	count := sequence.Particles[1].Element
	if count.MinOccurs != 0 || count.MaxOccurs != schema.OccursDefault {
		t.Fatalf("count occurs = (%d,%d), want (0,1)", count.MinOccurs, count.MaxOccurs)
	}

	item := sequence.Particles[2].Element
	if item.MinOccurs != 0 || item.MaxOccurs != schema.OccursUnbounded {
		t.Fatalf("item occurs = (%d,%d), want (0,unbounded)", item.MinOccurs, item.MaxOccurs)
	}
	if got, want := item.TypeName.Namespace, "urn:test"; got != want {
		t.Fatalf("item.TypeName.Namespace = %q, want %q", got, want)
	}

	if got, want := len(documentType.Attributes), 1; got != want {
		t.Fatalf("len(Attributes) = %d, want %d", got, want)
	}
	attribute := documentType.Attributes[0]
	if attribute.Name != "id" || attribute.Use != "required" {
		t.Fatalf("attribute = %#v", attribute)
	}
}

func TestParseSimpleTypeRestriction(t *testing.T) {
	input := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:test">
  <xs:simpleType name="CodeType">
    <xs:restriction base="xs:string">
      <xs:enumeration value="A"/>
      <xs:enumeration value="B"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`)

	file, err := Parse("test.xsd", input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if got, want := len(file.SimpleTypes), 1; got != want {
		t.Fatalf("len(SimpleTypes) = %d, want %d", got, want)
	}
	simpleType := file.SimpleTypes[0]
	if simpleType.Restriction == nil {
		t.Fatalf("Restriction is nil")
	}
	if got, want := simpleType.Restriction.BaseName.Namespace, xsdNamespace; got != want {
		t.Fatalf("BaseName.Namespace = %q, want %q", got, want)
	}
	if got, want := len(simpleType.Restriction.Facets), 2; got != want {
		t.Fatalf("len(Facets) = %d, want %d", got, want)
	}
	if got, want := simpleType.Restriction.Facets[0].Value, "A"; got != want {
		t.Fatalf("first facet value = %q, want %q", got, want)
	}
}

func TestParseSimpleContentExtension(t *testing.T) {
	input := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:tns="urn:test"
           targetNamespace="urn:test">
  <xs:complexType name="CodeType">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="scheme" type="xs:string"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`)

	file, err := Parse("test.xsd", input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if got, want := len(file.ComplexTypes), 1; got != want {
		t.Fatalf("len(ComplexTypes) = %d, want %d", got, want)
	}
	complexType := file.ComplexTypes[0]
	if complexType.SimpleContent == nil || complexType.SimpleContent.Extension == nil {
		t.Fatalf("SimpleContent.Extension is nil")
	}
	extension := complexType.SimpleContent.Extension
	if got, want := extension.BaseName.Namespace, xsdNamespace; got != want {
		t.Fatalf("Extension.BaseName.Namespace = %q, want %q", got, want)
	}
	if got, want := extension.BaseName.Local, "string"; got != want {
		t.Fatalf("Extension.BaseName.Local = %q, want %q", got, want)
	}
	if got, want := len(extension.Attributes), 1; got != want {
		t.Fatalf("len(Extension.Attributes) = %d, want %d", got, want)
	}
}

func TestParseSimpleContentRestriction(t *testing.T) {
	input := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:tns="urn:test"
           targetNamespace="urn:test">
  <xs:complexType name="CodeType">
    <xs:simpleContent>
      <xs:restriction base="xs:string">
        <xs:minLength value="1"/>
        <xs:enumeration value="A"/>
        <xs:attribute name="scheme" type="xs:string"/>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`)

	file, err := Parse("test.xsd", input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	complexType := file.ComplexTypes[0]
	if complexType.SimpleContent == nil || complexType.SimpleContent.Restriction == nil {
		t.Fatalf("SimpleContent.Restriction is nil")
	}
	restriction := complexType.SimpleContent.Restriction
	if got, want := restriction.BaseName.Local, "string"; got != want {
		t.Fatalf("Restriction.BaseName.Local = %q, want %q", got, want)
	}
	if got, want := len(restriction.Facets), 2; got != want {
		t.Fatalf("len(Restriction.Facets) = %d, want %d", got, want)
	}
	if got, want := len(restriction.Attributes), 1; got != want {
		t.Fatalf("len(Restriction.Attributes) = %d, want %d", got, want)
	}
}

func TestParseChoiceAndAllAndGroupRef(t *testing.T) {
	input := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:tns="urn:test"
           targetNamespace="urn:test">
  <xs:group name="Inner">
    <xs:sequence>
      <xs:element name="Fizz" type="xs:string"/>
    </xs:sequence>
  </xs:group>
  <xs:complexType name="ChoiceType">
    <xs:choice>
      <xs:element name="Bar" type="xs:string"/>
      <xs:group ref="tns:Inner"/>
    </xs:choice>
  </xs:complexType>
  <xs:complexType name="AllType">
    <xs:all>
      <xs:element name="Once" type="xs:int"/>
      <xs:element name="Opt" type="xs:int" minOccurs="0"/>
    </xs:all>
  </xs:complexType>
</xs:schema>`)

	file, err := Parse("test.xsd", input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if got, want := len(file.Groups), 1; got != want {
		t.Fatalf("len(Groups) = %d, want %d", got, want)
	}
	if file.Groups[0].Name != "Inner" || file.Groups[0].Content.Kind != schema.GroupSequence {
		t.Fatalf("group = %#v", file.Groups[0])
	}

	choice := file.ComplexTypes[0]
	if got, want := choice.Content[0].Group.Kind, schema.GroupChoice; got != want {
		t.Fatalf("choice group kind = %q, want %q", got, want)
	}
	groupRef := choice.Content[0].Group.Particles[1].Group
	if groupRef == nil || groupRef.Ref != "tns:Inner" || groupRef.RefName.Local != "Inner" {
		t.Fatalf("group ref = %#v", groupRef)
	}

	all := file.ComplexTypes[1]
	if got, want := all.Content[0].Group.Kind, schema.GroupAll; got != want {
		t.Fatalf("all group kind = %q, want %q", got, want)
	}
}

func TestParseComplexContentExtension(t *testing.T) {
	input := strings.NewReader(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:tns="urn:test"
           targetNamespace="urn:test">
  <xs:complexType name="BaseType">
    <xs:sequence>
      <xs:element name="base" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="DerivedType">
    <xs:complexContent>
      <xs:extension base="tns:BaseType">
        <xs:sequence>
          <xs:element name="extra" type="xs:int"/>
        </xs:sequence>
        <xs:attribute name="code" type="xs:string"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`)

	file, err := Parse("test.xsd", input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if got, want := len(file.ComplexTypes), 2; got != want {
		t.Fatalf("len(ComplexTypes) = %d, want %d", got, want)
	}
	derived := file.ComplexTypes[1]
	if derived.Extension == nil {
		t.Fatalf("Extension is nil")
	}
	if got, want := derived.Extension.BaseName.Namespace, "urn:test"; got != want {
		t.Fatalf("Extension.BaseName.Namespace = %q, want %q", got, want)
	}
	if got, want := derived.Extension.BaseName.Local, "BaseType"; got != want {
		t.Fatalf("Extension.BaseName.Local = %q, want %q", got, want)
	}
	if got, want := len(derived.Extension.Content), 1; got != want {
		t.Fatalf("len(Extension.Content) = %d, want %d", got, want)
	}
	if got, want := len(derived.Extension.Attributes), 1; got != want {
		t.Fatalf("len(Extension.Attributes) = %d, want %d", got, want)
	}
}

func TestParseFilesResolvesLocalIncludes(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.xsd")
	commonPath := filepath.Join(dir, "common.xsd")

	if err := os.WriteFile(rootPath, []byte(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:test">
  <xs:include schemaLocation="common.xsd"/>
  <xs:element name="document" type="DocumentType"/>
</xs:schema>`), 0o644); err != nil {
		t.Fatalf("write root schema: %v", err)
	}
	if err := os.WriteFile(commonPath, []byte(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:test">
  <xs:complexType name="DocumentType"/>
</xs:schema>`), 0o644); err != nil {
		t.Fatalf("write common schema: %v", err)
	}

	schemas, err := ParseFiles([]string{rootPath})
	if err != nil {
		t.Fatalf("ParseFiles returned error: %v", err)
	}
	if got, want := len(schemas.Files), 2; got != want {
		t.Fatalf("len(Files) = %d, want %d", got, want)
	}
	if got, want := len(schemas.Files[0].Includes), 1; got != want {
		t.Fatalf("len(Includes) = %d, want %d", got, want)
	}
	if got, want := schemas.Stats().ComplexTypes, 1; got != want {
		t.Fatalf("Stats().ComplexTypes = %d, want %d", got, want)
	}
}
