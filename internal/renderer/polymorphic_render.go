package renderer

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/harald-mue/xsd-parser-go/internal/generator"
)

func renderMixedContent(buf *bytes.Buffer, file generator.File, typ generator.TypeDecl) {
	if typ.Mixed == nil {
		return
	}
	m := typ.Mixed
	token := m.TokenTypeName
	// Token type
	fmt.Fprintf(buf, "type %s struct {\n", token)
	if m.HasText {
		fmt.Fprintln(buf, "\tText string")
	}
	if len(m.Elements) > 0 {
		fmt.Fprintf(buf, "\tElement %s\n", m.InterfaceName)
	}
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
	if len(m.Elements) > 0 {
		fmt.Fprintf(buf, "type %s interface {\n\t%s()\n}\n\n", m.InterfaceName, m.MarkerMethod)
		for _, el := range m.Elements {
			fmt.Fprintf(buf, "type %s struct {\n\tValue %s\n}\n\n", el.StructName, el.FieldType)
			fmt.Fprintf(buf, "func (*%s) %s() {}\n\n", el.StructName, m.MarkerMethod)
		}
	}

	// UnmarshalXML
	fmt.Fprintf(buf, "func (t *%s) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {\n", typ.Name)
	fmt.Fprintln(buf, "\tt.XMLName = start.Name")
	fmt.Fprintln(buf, "\tfor {")
	fmt.Fprintln(buf, "\t\ttok, err := d.Token()")
	fmt.Fprintln(buf, "\t\tif err != nil {")
	fmt.Fprintln(buf, "\t\t\treturn err")
	fmt.Fprintln(buf, "\t\t}")
	fmt.Fprintln(buf, "\t\tswitch e := tok.(type) {")
	if m.HasText {
		fmt.Fprintln(buf, "\t\tcase xml.CharData:")
		fmt.Fprintf(buf, "\t\t\tt.Content = append(t.Content, %s{Text: string(e)})\n", token)
	}
	fmt.Fprintln(buf, "\t\tcase xml.StartElement:")
	if len(m.Elements) == 0 {
		fmt.Fprintln(buf, "\t\t\tif err := d.Skip(); err != nil {")
		fmt.Fprintln(buf, "\t\t\t\treturn err")
		fmt.Fprintln(buf, "\t\t\t}")
	} else {
		fmt.Fprintln(buf, "\t\t\tswitch {")
		for _, el := range m.Elements {
			fmt.Fprintf(buf, "\t\t\tcase e.Name.Space == %q && e.Name.Local == %q:\n", el.XMLNamespace, el.XMLName)
			fmt.Fprintf(buf, "\t\t\t\tvar v %s\n", el.FieldType)
			fmt.Fprintln(buf, "\t\t\t\tif err := d.DecodeElement(&v, &e); err != nil {")
			fmt.Fprintln(buf, "\t\t\t\t\treturn err")
			fmt.Fprintln(buf, "\t\t\t\t}")
			fmt.Fprintf(buf, "\t\t\t\tt.Content = append(t.Content, %s{Element: &%s{Value: v}})\n", token, el.StructName)
		}
		fmt.Fprintln(buf, "\t\t\tdefault:")
		fmt.Fprintln(buf, "\t\t\t\tif err := d.Skip(); err != nil {")
		fmt.Fprintln(buf, "\t\t\t\t\treturn err")
		fmt.Fprintln(buf, "\t\t\t\t}")
		fmt.Fprintln(buf, "\t\t\t}")
	}
	fmt.Fprintln(buf, "\t\tcase xml.EndElement:")
	if file.ValidateOnUnmarshal && typ.Validate {
		fmt.Fprintln(buf, "\t\t\treturn t.Validate()")
	} else {
		fmt.Fprintln(buf, "\t\t\treturn nil")
	}
	fmt.Fprintln(buf, "\t\t}")
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)

	// MarshalXML
	fmt.Fprintf(buf, "func (t %s) MarshalXML(e *xml.Encoder, start xml.StartElement) error {\n", typ.Name)
	fmt.Fprintln(buf, "\tstart.Name = t.XMLName")
	fmt.Fprintln(buf, "\tif err := e.EncodeToken(start); err != nil {")
	fmt.Fprintln(buf, "\t\treturn err")
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintln(buf, "\tfor _, token := range t.Content {")
	if len(m.Elements) > 0 {
		fmt.Fprintln(buf, "\t\tif token.Element != nil {")
		fmt.Fprintln(buf, "\t\t\tswitch x := token.Element.(type) {")
		for _, el := range m.Elements {
			fmt.Fprintf(buf, "\t\t\tcase *%s:\n", el.StructName)
			fmt.Fprintf(buf, "\t\t\t\tchild := xml.StartElement{Name: xml.Name{Space: %q, Local: %q}}\n", el.XMLNamespace, el.XMLName)
			fmt.Fprintln(buf, "\t\t\t\tif err := e.EncodeElement(x.Value, child); err != nil {")
			fmt.Fprintln(buf, "\t\t\t\t\treturn err")
			fmt.Fprintln(buf, "\t\t\t\t}")
		}
		fmt.Fprintln(buf, "\t\t\tdefault:")
		fmt.Fprintf(buf, "\t\t\t\treturn fmt.Errorf(\"unknown mixed content element %%T\", token.Element)\n")
		fmt.Fprintln(buf, "\t\t\t}")
		fmt.Fprintln(buf, "\t\t\tcontinue")
		fmt.Fprintln(buf, "\t\t}")
	}
	if m.HasText {
		fmt.Fprintln(buf, "\t\tif token.Text != \"\" {")
		fmt.Fprintln(buf, "\t\t\tif err := e.EncodeToken(xml.CharData(token.Text)); err != nil {")
		fmt.Fprintln(buf, "\t\t\t\treturn err")
		fmt.Fprintln(buf, "\t\t\t}")
		fmt.Fprintln(buf, "\t\t}")
	}
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintln(buf, "\treturn e.EncodeToken(start.End())")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func renderChoice(buf *bytes.Buffer, file generator.File, typ generator.TypeDecl) {
	if typ.Choice == nil {
		return
	}
	choice := typ.Choice
	fmt.Fprintf(buf, "type %s interface {\n\t%s()\n}\n\n", choice.InterfaceName, choice.MarkerMethod)
	for _, variant := range choice.Variants {
		fmt.Fprintf(buf, "type %s struct {\n\tValue %s\n}\n\n", variant.StructName, variant.FieldType)
		fmt.Fprintf(buf, "func (*%s) %s() {}\n\n", variant.StructName, variant.MarkerMethod)
	}

	fmt.Fprintf(buf, "func (t *%s) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {\n", typ.Name)
	fmt.Fprintln(buf, "\tfor {")
	fmt.Fprintln(buf, "\t\ttok, err := d.Token()")
	fmt.Fprintln(buf, "\t\tif err != nil {")
	fmt.Fprintln(buf, "\t\t\treturn err")
	fmt.Fprintln(buf, "\t\t}")
	fmt.Fprintln(buf, "\t\tswitch e := tok.(type) {")
	fmt.Fprintln(buf, "\t\tcase xml.StartElement:")
	fmt.Fprintln(buf, "\t\t\tswitch {")
	for _, variant := range choice.Variants {
		fmt.Fprintf(buf, "\t\t\tcase e.Name.Space == %q && e.Name.Local == %q:\n", variant.XMLNamespace, variant.XMLName)
		fmt.Fprintf(buf, "\t\t\t\tvar v %s\n", variant.FieldType)
		fmt.Fprintln(buf, "\t\t\t\tif err := d.DecodeElement(&v, &e); err != nil {")
		fmt.Fprintln(buf, "\t\t\t\t\treturn err")
		fmt.Fprintln(buf, "\t\t\t\t}")
		if choice.Repeated {
			fmt.Fprintf(buf, "\t\t\t\tt.Content = append(t.Content, &%s{Value: v})\n", variant.StructName)
		} else {
			fmt.Fprintf(buf, "\t\t\t\tt.Content = &%s{Value: v}\n", variant.StructName)
		}
	}
	fmt.Fprintln(buf, "\t\t\tdefault:")
	fmt.Fprintln(buf, "\t\t\t\tif err := d.Skip(); err != nil {")
	fmt.Fprintln(buf, "\t\t\t\t\treturn err")
	fmt.Fprintln(buf, "\t\t\t\t}")
	fmt.Fprintln(buf, "\t\t\t}")
	fmt.Fprintln(buf, "\t\tcase xml.EndElement:")
	if file.ValidateOnUnmarshal && typ.Validate {
		fmt.Fprintln(buf, "\t\t\treturn t.Validate()")
	} else {
		fmt.Fprintln(buf, "\t\t\treturn nil")
	}
	fmt.Fprintln(buf, "\t\t}")
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)

	fmt.Fprintf(buf, "func (t %s) MarshalXML(e *xml.Encoder, start xml.StartElement) error {\n", typ.Name)
	fmt.Fprintln(buf, "\tif err := e.EncodeToken(start); err != nil {")
	fmt.Fprintln(buf, "\t\treturn err")
	fmt.Fprintln(buf, "\t}")
	if choice.Repeated {
		fmt.Fprintln(buf, "\tfor _, content := range t.Content {")
	} else {
		fmt.Fprintln(buf, "\tcontent := t.Content")
		fmt.Fprintln(buf, "\t{")
	}
	fmt.Fprintln(buf, "\t\tswitch x := content.(type) {")
	for _, variant := range choice.Variants {
		fmt.Fprintf(buf, "\t\tcase *%s:\n", variant.StructName)
		if file.RuntimePackage != "" {
			fmt.Fprintf(buf, "\t\t\tchild := %s(namespacePrefixes, %q, %q)\n", prefixedStartFunc(file), variant.XMLNamespace, variant.XMLName)
		} else {
			fmt.Fprintf(buf, "\t\t\tchild := prefixedStart(%q, %q)\n", variant.XMLNamespace, variant.XMLName)
		}
		fmt.Fprintln(buf, "\t\t\tif err := e.EncodeElement(x.Value, child); err != nil {")
		fmt.Fprintln(buf, "\t\t\t\treturn err")
		fmt.Fprintln(buf, "\t\t\t}")
	}
	fmt.Fprintln(buf, "\t\tcase nil:")
	fmt.Fprintln(buf, "\t\tdefault:")
	fmt.Fprintf(buf, "\t\t\treturn fmt.Errorf(\"unknown choice value %%T\", content)\n")
	fmt.Fprintln(buf, "\t\t}")
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintln(buf, "\treturn e.EncodeToken(start.End())")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func renderCustomXML(buf *bytes.Buffer, file generator.File, typ generator.TypeDecl) {
	if typ.CustomXML == nil || len(typ.CustomXML.Fields) == 0 {
		return
	}
	c := typ.CustomXML
	fmt.Fprintf(buf, "func (t *%s) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {\n", typ.Name)
	fmt.Fprintln(buf, "\tfor {")
	fmt.Fprintln(buf, "\t\ttok, err := d.Token()")
	fmt.Fprintln(buf, "\t\tif err != nil {")
	fmt.Fprintln(buf, "\t\t\treturn err")
	fmt.Fprintln(buf, "\t\t}")
	fmt.Fprintln(buf, "\t\tswitch e := tok.(type) {")
	fmt.Fprintln(buf, "\t\tcase xml.StartElement:")
	fmt.Fprintln(buf, "\t\t\thandled := false")
	for _, field := range c.Fields {
		qnameType := qualifiedQNameType(file, field.RegistryVar)
		xsiFunc := qualifiedXSITypeFunc(file, field.RegistryVar)
		fmt.Fprintln(buf, "\t\t\tif !handled {")
		fmt.Fprintf(buf, "\t\t\t\tdispatch := %s{Namespace: e.Name.Space, Local: e.Name.Local}\n", qnameType)
		fmt.Fprintf(buf, "\t\t\t\tif q, ok := %s(e); ok {\n", xsiFunc)
		fmt.Fprintln(buf, "\t\t\t\t\tdispatch = q")
		fmt.Fprintln(buf, "\t\t\t\t}")
		fmt.Fprintf(buf, "\t\t\t\tif entry, ok := %s[dispatch]; ok {\n", field.RegistryVar)
		fmt.Fprintln(buf, "\t\t\t\t\tv := entry.New()")
		fmt.Fprintln(buf, "\t\t\t\t\tif err := d.DecodeElement(v, &e); err != nil {")
		fmt.Fprintln(buf, "\t\t\t\t\t\treturn err")
		fmt.Fprintln(buf, "\t\t\t\t\t}")
		if field.Repeated {
			fmt.Fprintf(buf, "\t\t\t\t\tt.%s = append(t.%s, v)\n", field.FieldName, field.FieldName)
		} else {
			fmt.Fprintf(buf, "\t\t\t\t\tt.%s = v\n", field.FieldName)
		}
		fmt.Fprintln(buf, "\t\t\t\t\thandled = true")
		fmt.Fprintln(buf, "\t\t\t\t}")
		fmt.Fprintln(buf, "\t\t\t}")
	}
	fmt.Fprintln(buf, "\t\t\tif !handled {")
	fmt.Fprintln(buf, "\t\t\t\tif err := d.Skip(); err != nil {")
	fmt.Fprintln(buf, "\t\t\t\t\treturn err")
	fmt.Fprintln(buf, "\t\t\t\t}")
	fmt.Fprintln(buf, "\t\t\t}")
	fmt.Fprintln(buf, "\t\tcase xml.EndElement:")
	if file.ValidateOnUnmarshal && typ.Validate {
		fmt.Fprintln(buf, "\t\t\treturn t.Validate()")
	} else {
		fmt.Fprintln(buf, "\t\t\treturn nil")
	}
	fmt.Fprintln(buf, "\t\t}")
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)

	fmt.Fprintf(buf, "func (t %s) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {\n", typ.Name)
	renderDeclarePrefixedStart(buf, file, "start", c.ElementNS, c.ElementLocal, "\t")
	fmt.Fprintln(buf, "\tif err := e.EncodeToken(start); err != nil {")
	fmt.Fprintln(buf, "\t\treturn err")
	fmt.Fprintln(buf, "\t}")
	for _, field := range c.Fields {
		if field.Repeated {
			fmt.Fprintf(buf, "\tfor _, v := range t.%s {\n", field.FieldName)
		} else {
			fmt.Fprintf(buf, "\tif t.%s != nil {\n", field.FieldName)
			fmt.Fprintf(buf, "\t\tv := t.%s\n", field.FieldName)
		}
		fmt.Fprintln(buf, "\t\tmatched := false")
		fmt.Fprintf(buf, "\t\tfor _, entry := range %s {\n", field.RegistryVar)
		fmt.Fprintln(buf, "\t\t\tif reflect.TypeOf(v) != reflect.TypeOf(entry.New()) {")
		fmt.Fprintln(buf, "\t\t\t\tcontinue")
		fmt.Fprintln(buf, "\t\t\t}")
		if file.RuntimePackage != "" {
			fmt.Fprintf(buf, "\t\t\tchild := %s(namespacePrefixes, entry.Element.Namespace, entry.Element.Local)\n", prefixedStartFunc(file))
		} else {
			fmt.Fprintln(buf, "\t\t\tchild := prefixedStart(entry.Element.Namespace, entry.Element.Local)")
		}
		fmt.Fprintln(buf, "\t\t\tif entry.UseXSIType {")
		fmt.Fprintf(buf, "\t\t\t\tchild.Attr = append(child.Attr, xml.Attr{Name: xml.Name{Local: \"xmlns:xsi\"}, Value: %s})\n", xsiNamespaceRef(file))
		fmt.Fprintln(buf, "\t\t\t\tchild.Attr = append(child.Attr, xml.Attr{Name: xml.Name{Local: \"xsi:type\"}, Value: entry.XSIType.Local})")
		fmt.Fprintln(buf, "\t\t\t}")
		fmt.Fprintln(buf, "\t\t\tif err := e.EncodeElement(v, child); err != nil {")
		fmt.Fprintln(buf, "\t\t\t\treturn err")
		fmt.Fprintln(buf, "\t\t\t}")
		fmt.Fprintln(buf, "\t\t\tmatched = true")
		fmt.Fprintln(buf, "\t\t\tbreak")
		fmt.Fprintln(buf, "\t\t}")
		fmt.Fprintln(buf, "\t\tif !matched {")
		fmt.Fprintf(buf, "\t\t\treturn fmt.Errorf(\"unknown polymorphic value %%T\", v)\n")
		fmt.Fprintln(buf, "\t\t}")
		fmt.Fprintln(buf, "\t}")
	}
	fmt.Fprintln(buf, "\treturn e.EncodeToken(start.End())")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func qualifiedQNameType(file generator.File, registryVar string) string {
	if file.RuntimePackage != "" {
		return file.RuntimePackage + ".QName"
	}
	if idx := strings.LastIndex(registryVar, "."); idx >= 0 {
		return registryVar[:idx] + ".QName"
	}
	return "QName"
}

func qualifiedXSITypeFunc(file generator.File, registryVar string) string {
	if file.RuntimePackage != "" {
		return file.RuntimePackage + ".XSITypeQName"
	}
	if idx := strings.LastIndex(registryVar, "."); idx >= 0 {
		return registryVar[:idx] + ".XSITypeQName"
	}
	return "XSITypeQName"
}

func xmlTag(field generator.Field) string {
	if field.Polymorphic || field.ChoiceModel {
		return `xml:"-"`
	}
	if field.Chardata {
		return `xml:",chardata"`
	}
	if field.AnyElement {
		return `xml:",any"`
	}
	if field.AnyAttribute {
		return `xml:",any,attr"`
	}

	name := field.XMLName
	if name == "" {
		name = strings.ToLower(field.Name[:1]) + field.Name[1:]
	}

	var parts []string
	xmlName := name
	if field.XMLNamespace != "" && !field.Attribute {
		xmlName = field.XMLNamespace + " " + name
	}
	parts = append(parts, xmlName)
	if field.Attribute {
		parts = append(parts, "attr")
	}
	if field.Optional || field.Repeated {
		parts = append(parts, "omitempty")
	}
	return `xml:"` + strings.Join(parts, ",") + `"`
}
