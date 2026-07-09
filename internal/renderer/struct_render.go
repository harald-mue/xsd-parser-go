package renderer

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/harald-mue/xsd-parser-go/internal/generator"
)

func renderMarkerMethod(buf *bytes.Buffer, typ generator.TypeDecl) {
	for _, method := range typ.MarkerMethods {
		if method == "" {
			continue
		}
		fmt.Fprintf(buf, "func (*%s) %s() {}\n\n", typ.Name, method)
	}
}

func renderStruct(buf *bytes.Buffer, typ generator.TypeDecl) {
	if len(typ.Fields) == 0 && typ.XMLName == "" {
		fmt.Fprintf(buf, "type %s struct{}\n\n", typ.Name)
		return
	}

	fmt.Fprintf(buf, "type %s struct {\n", typ.Name)
	if typ.XMLName != "" {
		fmt.Fprintf(buf, "\tXMLName xml.Name `xml:%q`\n", typ.XMLName)
	} else if typ.Mixed != nil {
		fmt.Fprintln(buf, "\tXMLName xml.Name")
	}
	for _, field := range typ.Fields {
		renderDocComment(buf, field.Name, field.Documentation, "\t")
		fmt.Fprintf(buf, "\t%s %s `%s`\n", field.Name, field.Type, xmlTag(field))
	}
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func renderDocComment(buf *bytes.Buffer, identifier string, docs []string, indent string) {
	if len(docs) == 0 {
		return
	}
	for _, doc := range docs {
		doc = strings.TrimSpace(strings.ReplaceAll(doc, "\n", " "))
		if doc == "" {
			continue
		}
		fmt.Fprintf(buf, "%s// %s %s\n", indent, identifier, doc)
	}
}

func renderElementAliasMarshal(buf *bytes.Buffer, file generator.File, typ generator.TypeDecl) {
	if typ.Alias == "" || typ.XMLName == "" || typ.AliasEqual {
		return
	}
	ns, local := splitXMLName(typ.XMLName)
	fmt.Fprintf(buf, "func (t %s) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {\n", typ.Name)
	renderDeclarePrefixedStart(buf, file, "start", ns, local, "\t")
	fmt.Fprintf(buf, "\tv := %s(t)\n", typ.Alias)
	fmt.Fprintf(buf, "\treturn e.EncodeElement(&v, start)\n")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func renderElementAliasUnmarshal(buf *bytes.Buffer, typ generator.TypeDecl) {
	if !typ.AliasGenerated || typ.Alias == "" || typ.AliasEqual {
		return
	}
	fmt.Fprintf(buf, "func (t *%s) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {\n", typ.Name)
	fmt.Fprintf(buf, "\tvar v %s\n", typ.Alias)
	fmt.Fprintln(buf, "\tif err := d.DecodeElement(&v, &start); err != nil {")
	fmt.Fprintln(buf, "\t\treturn err")
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintf(buf, "\t*t = %s(v)\n", typ.Name)
	fmt.Fprintln(buf, "\treturn nil")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func renderElementAliasValidate(buf *bytes.Buffer, typ generator.TypeDecl) {
	if !typ.AliasValidate || typ.Alias == "" || typ.AliasEqual {
		return
	}
	fmt.Fprintf(buf, "func (v %s) Validate() error {\n", typ.Name)
	fmt.Fprintf(buf, "\treturn %s(v).Validate()\n", typ.Alias)
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func renderPrefixMarshal(buf *bytes.Buffer, file generator.File, typ generator.TypeDecl) {
	if !needsNormalPrefixMarshal(typ) {
		return
	}
	if typ.XMLName != "" {
		fmt.Fprintf(buf, "func (t %s) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {\n", typ.Name)
		ns, local := splitXMLName(typ.XMLName)
		renderDeclarePrefixedStart(buf, file, "start", ns, local, "\t")
	} else {
		fmt.Fprintf(buf, "func (t %s) MarshalXML(e *xml.Encoder, start xml.StartElement) error {\n", typ.Name)
		fmt.Fprintln(buf, "\tif start.Name.Space != \"\" {")
		if file.RuntimePackage != "" {
			fmt.Fprintf(buf, "\t\tstart = %s(namespacePrefixes, start.Name.Space, start.Name.Local)\n", prefixedStartFunc(file))
		} else {
			fmt.Fprintln(buf, "\t\tstart = prefixedStart(start.Name.Space, start.Name.Local)")
		}
		fmt.Fprintln(buf, "\t}")
	}
	for _, field := range typ.Fields {
		if field.AnyAttribute {
			fmt.Fprintf(buf, "\tstart.Attr = append(start.Attr, t.%s...)\n", field.Name)
			continue
		}
		if !field.Attribute {
			continue
		}
		name := field.XMLName
		if name == "" {
			name = lowerFirst(field.Name)
		}
		if strings.HasPrefix(field.Type, "*") {
			fmt.Fprintf(buf, "\tif t.%s != nil {\n", field.Name)
			renderAppendPrefixedAttr(buf, file, "start", field.XMLNamespace, name, "fmt.Sprint(*t."+field.Name+")", "\t\t")
			fmt.Fprintln(buf, "\t}")
		} else {
			renderAppendPrefixedAttr(buf, file, "start", field.XMLNamespace, name, "fmt.Sprint(t."+field.Name+")", "\t")
		}
	}
	fmt.Fprintln(buf, "\tif err := e.EncodeToken(start); err != nil {")
	fmt.Fprintln(buf, "\t\treturn err")
	fmt.Fprintln(buf, "\t}")
	if prefixMarshalNeedsChildVar(typ) {
		fmt.Fprintln(buf, "\tvar child xml.StartElement")
	}
	for _, field := range typ.Fields {
		renderPrefixMarshalField(buf, file, field)
	}
	fmt.Fprintln(buf, "\treturn e.EncodeToken(start.End())")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func renderSetPrefixedStart(buf *bytes.Buffer, file generator.File, target, namespace, local, indent string) {
	if file.RuntimePackage != "" {
		fmt.Fprintf(buf, "%s%s = %s(namespacePrefixes, %q, %q)\n", indent, target, prefixedStartFunc(file), namespace, local)
		return
	}
	fmt.Fprintf(buf, "%s%s = prefixedStart(%q, %q)\n", indent, target, namespace, local)
}

func renderDeclarePrefixedStart(buf *bytes.Buffer, file generator.File, target, namespace, local, indent string) {
	if file.RuntimePackage != "" {
		fmt.Fprintf(buf, "%s%s := %s(namespacePrefixes, %q, %q)\n", indent, target, prefixedStartFunc(file), namespace, local)
		return
	}
	fmt.Fprintf(buf, "%s%s := prefixedStart(%q, %q)\n", indent, target, namespace, local)
}

func renderAppendPrefixedAttr(buf *bytes.Buffer, file generator.File, target, namespace, local, valueExpr, indent string) {
	if file.RuntimePackage != "" {
		fmt.Fprintf(buf, "%s%s.Attr = append(%s.Attr, %s(namespacePrefixes, %q, %q, %s)...)\n", indent, target, target, prefixedAttrFunc(file), namespace, local, valueExpr)
		return
	}
	fmt.Fprintf(buf, "%s%s.Attr = append(%s.Attr, prefixedAttr(%q, %q, %s)...)\n", indent, target, target, namespace, local, valueExpr)
}

func renderPrefixMarshalField(buf *bytes.Buffer, file generator.File, field generator.Field) {
	if field.Attribute || field.AnyAttribute {
		return
	}
	if field.Chardata {
		fmt.Fprintf(buf, "\tif err := e.EncodeToken(xml.CharData(fmt.Sprint(t.%s))); err != nil {\n", field.Name)
		fmt.Fprintln(buf, "\t\treturn err")
		fmt.Fprintln(buf, "\t}")
		return
	}
	if field.AnyElement {
		if field.Repeated {
			fmt.Fprintf(buf, "\tfor _, v := range t.%s {\n", field.Name)
			fmt.Fprintln(buf, "\t\tif err := e.Encode(v); err != nil {")
			fmt.Fprintln(buf, "\t\t\treturn err")
			fmt.Fprintln(buf, "\t\t}")
			fmt.Fprintln(buf, "\t}")
			return
		}
		if strings.HasPrefix(field.Type, "*") {
			fmt.Fprintf(buf, "\tif t.%s != nil {\n", field.Name)
			fmt.Fprintf(buf, "\t\tif err := e.Encode(t.%s); err != nil {\n", field.Name)
			fmt.Fprintln(buf, "\t\t\treturn err")
			fmt.Fprintln(buf, "\t\t}")
			fmt.Fprintln(buf, "\t}")
			return
		}
		fmt.Fprintf(buf, "\tif err := e.Encode(t.%s); err != nil {\n", field.Name)
		fmt.Fprintln(buf, "\t\treturn err")
		fmt.Fprintln(buf, "\t}")
		return
	}
	name := field.XMLName
	if name == "" {
		name = lowerFirst(field.Name)
	}
	if field.Repeated {
		fmt.Fprintf(buf, "\tfor _, v := range t.%s {\n", field.Name)
		renderSetPrefixedStart(buf, file, "child", field.XMLNamespace, name, "\t\t")
		fmt.Fprintln(buf, "\t\tif err := e.EncodeElement(v, child); err != nil {")
		fmt.Fprintln(buf, "\t\t\treturn err")
		fmt.Fprintln(buf, "\t\t}")
		fmt.Fprintln(buf, "\t}")
		return
	}
	if strings.HasPrefix(field.Type, "*") {
		fmt.Fprintf(buf, "\tif t.%s != nil {\n", field.Name)
		renderSetPrefixedStart(buf, file, "child", field.XMLNamespace, name, "\t\t")
		fmt.Fprintf(buf, "\t\tif err := e.EncodeElement(t.%s, child); err != nil {\n", field.Name)
		fmt.Fprintln(buf, "\t\t\treturn err")
		fmt.Fprintln(buf, "\t\t}")
		fmt.Fprintln(buf, "\t}")
		return
	}
	renderSetPrefixedStart(buf, file, "child", field.XMLNamespace, name, "\t")
	fmt.Fprintf(buf, "\tif err := e.EncodeElement(t.%s, child); err != nil {\n", field.Name)
	fmt.Fprintln(buf, "\t\treturn err")
	fmt.Fprintln(buf, "\t}")
}

func prefixMarshalNeedsChildVar(typ generator.TypeDecl) bool {
	for _, field := range typ.Fields {
		if !field.Attribute && !field.AnyAttribute && !field.Chardata && !field.AnyElement {
			return true
		}
	}
	return false
}

func needsNormalPrefixMarshal(typ generator.TypeDecl) bool {
	if typ.Kind != "complexType" || typ.CustomXML != nil || typ.Choice != nil || typ.Mixed != nil || len(typ.MarkerMethods) > 0 || len(typ.ImplInterfaces) > 0 {
		return false
	}
	if typ.XMLName != "" {
		return true
	}
	for _, field := range typ.Fields {
		if field.ChoiceModel || field.Polymorphic {
			return false
		}
		if field.XMLNamespace != "" || field.AnyAttribute || field.AnyElement {
			return true
		}
	}
	return false
}

func structValidationNeedsFmt(typ generator.TypeDecl) bool {
	if !typ.Validate {
		return false
	}
	for _, field := range typ.Fields {
		if field.Repeated && (field.MinOccurs > 0 || field.MaxOccurs > 0) {
			return true
		}
		if field.Required && !field.Repeated && (strings.HasPrefix(field.Type, "*") || field.ChoiceModel || field.Polymorphic) {
			return true
		}
		if field.AnyElement || field.AnyAttribute || field.Fixed != "" {
			return true
		}
	}
	return false
}

func normalPrefixMarshalNeedsFmt(typ generator.TypeDecl) bool {
	if !needsNormalPrefixMarshal(typ) {
		return false
	}
	for _, field := range typ.Fields {
		if field.Attribute || field.Chardata {
			return true
		}
	}
	return false
}

func splitXMLName(name string) (string, string) {
	if ns, local, ok := strings.Cut(name, " "); ok {
		return ns, local
	}
	return "", name
}

func lowerFirst(name string) string {
	if name == "" {
		return name
	}
	return strings.ToLower(name[:1]) + name[1:]
}

func renderStructUnmarshal(buf *bytes.Buffer, file generator.File, typ generator.TypeDecl) {
	if typ.CustomXML != nil || typ.Choice != nil || typ.Mixed != nil {
		return
	}
	if needsPresenceTracking(typ) || needsDefaultApplication(typ) {
		renderPresenceTrackingUnmarshal(buf, typ, file.ValidateOnUnmarshal)
		return
	}
	if !file.ValidateOnUnmarshal || !typ.Validate {
		return
	}
	fmt.Fprintf(buf, "func (t *%s) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {\n", typ.Name)
	fmt.Fprintf(buf, "\ttype alias %s\n", typ.Name)
	fmt.Fprintln(buf, "\tvar v alias")
	fmt.Fprintln(buf, "\tif err := d.DecodeElement(&v, &start); err != nil {")
	fmt.Fprintln(buf, "\t\treturn err")
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintf(buf, "\t*t = %s(v)\n", typ.Name)
	fmt.Fprintln(buf, "\treturn t.Validate()")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func renderPresenceTrackingUnmarshal(buf *bytes.Buffer, typ generator.TypeDecl, validateOnUnmarshal bool) {
	fmt.Fprintf(buf, "func (t *%s) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {\n", typ.Name)
	fmt.Fprintln(buf, "\tvar v struct {")
	for _, field := range typ.Fields {
		if field.ChoiceModel || field.Polymorphic {
			continue
		}
		fieldType := field.Type
		if needsFieldPresenceTracking(field) || needsFieldDefaultShadow(field) {
			fieldType = "*" + field.Type
		}
		fmt.Fprintf(buf, "\t\t%s %s `%s`\n", field.Name, fieldType, xmlTag(field))
	}
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintln(buf, "\tif err := d.DecodeElement(&v, &start); err != nil {")
	fmt.Fprintln(buf, "\t\treturn err")
	fmt.Fprintln(buf, "\t}")
	for _, field := range typ.Fields {
		if field.ChoiceModel || field.Polymorphic {
			continue
		}
		if needsFieldPresenceTracking(field) {
			fmt.Fprintf(buf, "\tif v.%s == nil {\n", field.Name)
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"field %s is required\")\n", field.Name)
			fmt.Fprintln(buf, "\t}")
			fmt.Fprintf(buf, "\tt.%s = *v.%s\n", field.Name, field.Name)
		} else if needsFieldDefaultShadow(field) {
			fmt.Fprintf(buf, "\tif v.%s == nil {\n", field.Name)
			fmt.Fprintf(buf, "\t\tt.%s = %s\n", field.Name, defaultValueExpr(field))
			fmt.Fprintln(buf, "\t} else {")
			fmt.Fprintf(buf, "\t\tt.%s = *v.%s\n", field.Name, field.Name)
			fmt.Fprintln(buf, "\t}")
		} else if (field.Default != "" || field.Fixed != "") && strings.HasPrefix(field.Type, "*") {
			fmt.Fprintf(buf, "\tif v.%s == nil {\n", field.Name)
			fmt.Fprintf(buf, "\t\tdefaultValue := %s\n", defaultValueExpr(fieldWithType(field, strings.TrimPrefix(field.Type, "*"))))
			fmt.Fprintf(buf, "\t\tt.%s = &defaultValue\n", field.Name)
			fmt.Fprintln(buf, "\t} else {")
			fmt.Fprintf(buf, "\t\tt.%s = v.%s\n", field.Name, field.Name)
			fmt.Fprintln(buf, "\t}")
		} else {
			fmt.Fprintf(buf, "\tt.%s = v.%s\n", field.Name, field.Name)
		}
	}
	if validateOnUnmarshal {
		fmt.Fprintln(buf, "\treturn t.Validate()")
	} else {
		fmt.Fprintln(buf, "\treturn nil")
	}
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func needsPresenceTracking(typ generator.TypeDecl) bool {
	for _, field := range typ.Fields {
		if needsFieldPresenceTracking(field) {
			return true
		}
	}
	return false
}

func needsDefaultApplication(typ generator.TypeDecl) bool {
	for _, field := range typ.Fields {
		if (field.Default != "" || field.Fixed != "") && !field.Repeated && !field.AnyElement && !field.AnyAttribute && !field.ChoiceModel && !field.Polymorphic && !field.Nillable {
			return true
		}
	}
	return false
}

func needsFieldDefaultShadow(field generator.Field) bool {
	return (field.Default != "" || field.Fixed != "") && !strings.HasPrefix(field.Type, "*") && !field.Repeated && !field.AnyElement && !field.AnyAttribute && !field.ChoiceModel && !field.Polymorphic && !field.Nillable
}

func fieldWithType(field generator.Field, typ string) generator.Field {
	field.Type = typ
	return field
}

func defaultValueExpr(field generator.Field) string {
	value := field.Default
	if field.Fixed != "" {
		value = field.Fixed
	}
	baseType := strings.TrimPrefix(field.Type, "*")
	if strings.HasSuffix(baseType, "XSDBase64Binary") {
		return qualifiedHelperCall(baseType, "MustBase64Binary", value)
	}
	if strings.HasSuffix(baseType, "XSDHexBinary") {
		return qualifiedHelperCall(baseType, "MustHexBinary", value)
	}
	switch baseType {
	case "string":
		return fmt.Sprintf("%q", value)
	case "bool":
		if value == "true" || value == "1" {
			return "true"
		}
		return "false"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return fmt.Sprintf("%s(%s)", baseType, value)
	default:
		return fmt.Sprintf("%s(%q)", baseType, value)
	}
}

func qualifiedHelperCall(typeName, helper, value string) string {
	prefix := ""
	if idx := strings.LastIndex(typeName, "."); idx >= 0 {
		prefix = typeName[:idx+1]
	}
	return fmt.Sprintf("%s%s(%q)", prefix, helper, value)
}

func needsFieldPresenceTracking(field generator.Field) bool {
	if !field.Required || field.Repeated || field.ChoiceModel || field.Polymorphic || field.AnyElement || field.AnyAttribute || field.Chardata || field.Nillable {
		return false
	}
	return !strings.HasPrefix(field.Type, "*")
}

func renderStructValidate(buf *bytes.Buffer, file generator.File, typ generator.TypeDecl) {
	if !typ.Validate || len(typ.Fields) == 0 {
		return
	}
	fmt.Fprintf(buf, "func (v %s) Validate() error {\n", typ.Name)
	for _, field := range typ.Fields {
		renderFieldStrictValidation(buf, file, typ, field)
		fmt.Fprintf(buf, "\tif err := %s(%q, v.%s); err != nil {\n", validateFieldFunc(file), field.Name, field.Name)
		fmt.Fprintln(buf, "\t\treturn err")
		fmt.Fprintln(buf, "\t}")
	}
	fmt.Fprintln(buf, "\treturn nil")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func renderFieldStrictValidation(buf *bytes.Buffer, file generator.File, typ generator.TypeDecl, field generator.Field) {
	if field.Repeated {
		if field.MinOccurs > 0 {
			fmt.Fprintf(buf, "\tif len(v.%s) < %d {\n", field.Name, field.MinOccurs)
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"field %s requires at least %d item(s)\")\n", field.Name, field.MinOccurs)
			fmt.Fprintln(buf, "\t}")
		}
		if field.MaxOccurs > 0 {
			fmt.Fprintf(buf, "\tif len(v.%s) > %d {\n", field.Name, field.MaxOccurs)
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"field %s allows at most %d item(s)\")\n", field.Name, field.MaxOccurs)
			fmt.Fprintln(buf, "\t}")
		}
	}
	if field.Required && !field.Repeated && (strings.HasPrefix(field.Type, "*") || field.ChoiceModel || field.Polymorphic) {
		fmt.Fprintf(buf, "\tif v.%s == nil {\n", field.Name)
		fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"field %s is required\")\n", field.Name)
		fmt.Fprintln(buf, "\t}")
	}
	renderFixedValidation(buf, field)
	renderAnyWildcardValidation(buf, file, typ, field)
}

func renderFixedValidation(buf *bytes.Buffer, field generator.Field) {
	if field.Fixed == "" || field.AnyElement || field.AnyAttribute || field.ChoiceModel || field.Polymorphic {
		return
	}
	if field.Repeated {
		fmt.Fprintf(buf, "\tfor i, item := range v.%s {\n", field.Name)
		fmt.Fprintf(buf, "\t\tif fmt.Sprint(item) != %q {\n", field.Fixed)
		fmt.Fprintf(buf, "\t\t\treturn fmt.Errorf(\"field %s[%%d] must have fixed value %%q\", i, %q)\n", field.Name, field.Fixed)
		fmt.Fprintln(buf, "\t\t}")
		fmt.Fprintln(buf, "\t}")
		return
	}
	if strings.HasPrefix(field.Type, "*") {
		fmt.Fprintf(buf, "\tif v.%s != nil && fmt.Sprint(*v.%s) != %q {\n", field.Name, field.Name, field.Fixed)
		fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"field %s must have fixed value %%q\", %q)\n", field.Name, field.Fixed)
		fmt.Fprintln(buf, "\t}")
		return
	}
	fmt.Fprintf(buf, "\tif fmt.Sprint(v.%s) != %q {\n", field.Name, field.Fixed)
	fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"field %s must have fixed value %%q\", %q)\n", field.Name, field.Fixed)
	fmt.Fprintln(buf, "\t}")
}

func renderAnyWildcardValidation(buf *bytes.Buffer, file generator.File, typ generator.TypeDecl, field generator.Field) {
	if !field.AnyElement && !field.AnyAttribute {
		return
	}
	constraint := field.AnyNamespace
	target := field.AnyTargetNamespace
	process := field.AnyProcessContents
	matchFunc := wildcardNamespaceAllowedFunc(file)
	elementProcessFunc := wildcardElementProcessContentsFunc(file)
	attributeProcessFunc := wildcardAttributeProcessContentsFunc(file)
	if field.AnyElement {
		if field.Repeated {
			fmt.Fprintf(buf, "\tfor i := range v.%s {\n", field.Name)
			fmt.Fprintf(buf, "\t\tif !%s(%q, %q, v.%s[i].XMLName.Space) {\n", matchFunc, constraint, target, field.Name)
			fmt.Fprintf(buf, "\t\t\treturn fmt.Errorf(\"field %s[%%d] namespace %%q is not allowed by xs:any namespace constraint %%q\", i, v.%s[i].XMLName.Space, %q)\n", field.Name, field.Name, constraint)
			fmt.Fprintln(buf, "\t\t}")
			fmt.Fprintf(buf, "\t\tif !%s(%q, v.%s[i].XMLName) {\n", elementProcessFunc, process, field.Name)
			fmt.Fprintf(buf, "\t\t\treturn fmt.Errorf(\"field %s[%%d] does not satisfy xs:any processContents %%q\", i, %q)\n", field.Name, process)
			fmt.Fprintln(buf, "\t\t}")
			fmt.Fprintln(buf, "\t}")
			return
		}
		access := "v." + field.Name
		if strings.HasPrefix(field.Type, "*") {
			fmt.Fprintf(buf, "\tif v.%s != nil {\n", field.Name)
			fmt.Fprintf(buf, "\t\tif !%s(%q, %q, %s.XMLName.Space) {\n", matchFunc, constraint, target, access)
			fmt.Fprintf(buf, "\t\t\treturn fmt.Errorf(\"field %s namespace %%q is not allowed by xs:any namespace constraint %%q\", %s.XMLName.Space, %q)\n", field.Name, access, constraint)
			fmt.Fprintln(buf, "\t\t}")
			fmt.Fprintf(buf, "\t\tif !%s(%q, %s.XMLName) {\n", elementProcessFunc, process, access)
			fmt.Fprintf(buf, "\t\t\treturn fmt.Errorf(\"field %s does not satisfy xs:any processContents %%q\", %q)\n", field.Name, process)
			fmt.Fprintln(buf, "\t\t}")
			fmt.Fprintln(buf, "\t}")
			return
		}
		fmt.Fprintf(buf, "\tif !%s(%q, %q, %s.XMLName.Space) {\n", matchFunc, constraint, target, access)
		fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"field %s namespace %%q is not allowed by xs:any namespace constraint %%q\", %s.XMLName.Space, %q)\n", field.Name, access, constraint)
		fmt.Fprintln(buf, "\t}")
		fmt.Fprintf(buf, "\tif !%s(%q, %s.XMLName) {\n", elementProcessFunc, process, access)
		fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"field %s does not satisfy xs:any processContents %%q\", %q)\n", field.Name, process)
		fmt.Fprintln(buf, "\t}")
		return
	}
	fmt.Fprintf(buf, "\tfor i, attr := range v.%s {\n", field.Name)
	fmt.Fprintf(buf, "\t\tif !%s(%q, %q, attr.Name.Space) {\n", matchFunc, constraint, target)
	fmt.Fprintf(buf, "\t\t\treturn fmt.Errorf(\"field %s[%%d] namespace %%q is not allowed by xs:anyAttribute namespace constraint %%q\", i, attr.Name.Space, %q)\n", field.Name, constraint)
	fmt.Fprintln(buf, "\t\t}")
	fmt.Fprintf(buf, "\t\tif !%s(%q, attr.Name) {\n", attributeProcessFunc, process)
	fmt.Fprintf(buf, "\t\t\treturn fmt.Errorf(\"field %s[%%d] does not satisfy xs:anyAttribute processContents %%q\", i, %q)\n", field.Name, process)
	fmt.Fprintln(buf, "\t\t}")
	fmt.Fprintln(buf, "\t}")
	_ = typ
}
