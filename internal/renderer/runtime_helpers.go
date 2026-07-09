package renderer

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/harald-mue/xsd-parser-go/internal/generator"
)

func fileNeedsXSDScalarHelpers(file generator.File) bool {
	for _, typ := range file.Types {
		if typeUsesXSDScalarHelper(typ.Alias) || typeUsesXSDScalarHelper(typ.Name) {
			return true
		}
		for _, field := range typ.Fields {
			if typeUsesXSDScalarHelper(field.Type) {
				return true
			}
		}
		if typ.List != nil && typeUsesXSDScalarHelper(typ.List.ItemType) {
			return true
		}
		if typ.Union != nil {
			for _, member := range typ.Union.MemberTypes {
				if typeUsesXSDScalarHelper(member) {
					return true
				}
			}
		}
	}
	return false
}

func typeUsesXSDScalarHelper(typ string) bool {
	for _, name := range []string{"XSDID", "XSDIDREF", "XSDIDREFS", "XSDDateTime", "XSDDate", "XSDTime", "XSDDuration", "XSDGYear", "XSDGYearMonth", "XSDGMonth", "XSDGMonthDay", "XSDGDay", "XSDDecimal", "XSDHexBinary", "XSDBase64Binary"} {
		if typ == name || strings.Contains(typ, "."+name) || strings.Contains(typ, "["+name+"]") || strings.Contains(typ, "*"+name) || strings.Contains(typ, "[]"+name) {
			return true
		}
	}
	return false
}

func fileNeedsStringFacetValidation(file generator.File) bool {
	for _, typ := range file.Types {
		for _, facet := range typ.Facets {
			switch facet.Name {
			case "whiteSpace", "totalDigits", "fractionDigits":
				return true
			}
		}
	}
	return false
}

func renderXSDScalarHelpers(buf *bytes.Buffer) {
	buf.WriteString(`// XSDID represents xs:ID.
type XSDID string

func (v *XSDID) UnmarshalText(text []byte) error {
	s := string(text)
	if err := validatePattern(s, xsdNCNamePattern, "xs:ID"); err != nil {
		return err
	}
	*v = XSDID(s)
	return nil
}
func (v XSDID) MarshalText() ([]byte, error) { return []byte(v), nil }

// XSDIDREF represents xs:IDREF.
type XSDIDREF string

func (v *XSDIDREF) UnmarshalText(text []byte) error {
	s := string(text)
	if err := validatePattern(s, xsdNCNamePattern, "xs:IDREF"); err != nil {
		return err
	}
	*v = XSDIDREF(s)
	return nil
}
func (v XSDIDREF) MarshalText() ([]byte, error) { return []byte(v), nil }
func (v XSDIDREF) Resolve(ids map[XSDID]any) (any, bool) { value, ok := ids[XSDID(v)]; return value, ok }

// XSDIDREFS represents xs:IDREFS.
type XSDIDREFS []XSDIDREF

func (v *XSDIDREFS) UnmarshalText(text []byte) error {
	parts := strings.Fields(string(text))
	refs := make([]XSDIDREF, 0, len(parts))
	for _, part := range parts {
		if err := validatePattern(part, xsdNCNamePattern, "xs:IDREFS"); err != nil {
			return err
		}
		refs = append(refs, XSDIDREF(part))
	}
	*v = refs
	return nil
}
func (v XSDIDREFS) MarshalText() ([]byte, error) {
	parts := make([]string, 0, len(v))
	for _, ref := range v { parts = append(parts, string(ref)) }
	return []byte(strings.Join(parts, " ")), nil
}

var xsdNCNamePattern = regexp.MustCompile("^[A-Za-z_][A-Za-z0-9_.-]*$")

// XSDDateTime represents xs:dateTime and preserves the XML lexical value.
type XSDDateTime string

func (v *XSDDateTime) UnmarshalText(text []byte) error {
	s := string(text)
	if _, err := parseXSDTime(s, time.RFC3339Nano, time.RFC3339); err != nil {
		return err
	}
	*v = XSDDateTime(s)
	return nil
}

func (v XSDDateTime) MarshalText() ([]byte, error) { return []byte(v), nil }
func (v XSDDateTime) Time() (time.Time, error)      { return parseXSDTime(string(v), time.RFC3339Nano, time.RFC3339) }

// XSDDate represents xs:date and preserves the XML lexical value.
type XSDDate string

func (v *XSDDate) UnmarshalText(text []byte) error {
	s := string(text)
	if _, err := parseXSDTime(s, "2006-01-02", "2006-01-02Z", "2006-01-02-07:00"); err != nil {
		return err
	}
	*v = XSDDate(s)
	return nil
}

func (v XSDDate) MarshalText() ([]byte, error) { return []byte(v), nil }
func (v XSDDate) Time() (time.Time, error)      { return parseXSDTime(string(v), "2006-01-02", "2006-01-02Z", "2006-01-02-07:00") }

// XSDTime represents xs:time and preserves the XML lexical value.
type XSDTime string

func (v *XSDTime) UnmarshalText(text []byte) error {
	s := string(text)
	if _, err := parseXSDTime(s, "15:04:05", "15:04:05Z", "15:04:05-07:00", "15:04:05.999999999", "15:04:05.999999999Z", "15:04:05.999999999-07:00"); err != nil {
		return err
	}
	*v = XSDTime(s)
	return nil
}

func (v XSDTime) MarshalText() ([]byte, error) { return []byte(v), nil }
func (v XSDTime) Time() (time.Time, error) {
	return parseXSDTime(string(v), "15:04:05", "15:04:05Z", "15:04:05-07:00", "15:04:05.999999999", "15:04:05.999999999Z", "15:04:05.999999999-07:00")
}

func parseXSDTime(value string, layouts ...string) (time.Time, error) {
	var last error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
		last = err
	}
	return time.Time{}, last
}

// XSDDuration represents xs:duration and preserves the XML lexical value.
type XSDDuration string

func (v *XSDDuration) UnmarshalText(text []byte) error {
	s := string(text)
	if !xsdDurationPattern.MatchString(s) {
		return fmt.Errorf("invalid xs:duration %q", s)
	}
	*v = XSDDuration(s)
	return nil
}

func (v XSDDuration) MarshalText() ([]byte, error) { return []byte(v), nil }

var xsdDurationPattern = regexp.MustCompile("^-?P(?:(?:[0-9]+Y)?(?:[0-9]+M)?(?:[0-9]+D)?(?:T(?:[0-9]+H)?(?:[0-9]+M)?(?:[0-9]+(?:\\.[0-9]+)?S)?)?)$")

// XSDDecimal represents xs:decimal exactly as its XML lexical value.
type XSDDecimal string

func (v *XSDDecimal) UnmarshalText(text []byte) error {
	s := string(text)
	if _, ok := new(big.Rat).SetString(s); !ok {
		return fmt.Errorf("invalid xs:decimal %q", s)
	}
	*v = XSDDecimal(s)
	return nil
}

func (v XSDDecimal) MarshalText() ([]byte, error) { return []byte(v), nil }
func (v XSDDecimal) Rat() (*big.Rat, error) {
	r, ok := new(big.Rat).SetString(string(v))
	if !ok {
		return nil, fmt.Errorf("invalid xs:decimal %q", string(v))
	}
	return r, nil
}

// XSDHexBinary represents xs:hexBinary.
type XSDHexBinary []byte

func (v *XSDHexBinary) UnmarshalText(text []byte) error {
	decoded, err := hex.DecodeString(string(text))
	if err != nil {
		return err
	}
	*v = decoded
	return nil
}

func (v XSDHexBinary) MarshalText() ([]byte, error) {
	encoded := make([]byte, hex.EncodedLen(len(v)))
	hex.Encode(encoded, v)
	return encoded, nil
}

func MustHexBinary(value string) XSDHexBinary {
	decoded, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return XSDHexBinary(decoded)
}

// XSDBase64Binary represents xs:base64Binary.
type XSDBase64Binary []byte

func (v *XSDBase64Binary) UnmarshalText(text []byte) error {
	decoded, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return err
	}
	*v = decoded
	return nil
}

func (v XSDBase64Binary) MarshalText() ([]byte, error) {
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(v)))
	base64.StdEncoding.Encode(encoded, v)
	return encoded, nil
}

func MustBase64Binary(value string) XSDBase64Binary {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return XSDBase64Binary(decoded)
}

// Partial-date XSD scalar types preserve their XML lexical values.
type XSDGYear string
type XSDGYearMonth string
type XSDGMonth string
type XSDGMonthDay string
type XSDGDay string

func (v *XSDGYear) UnmarshalText(text []byte) error {
	s := string(text)
	if err := validatePattern(s, xsdGYearPattern, "xs:gYear"); err != nil {
		return err
	}
	*v = XSDGYear(s)
	return nil
}
func (v XSDGYear) MarshalText() ([]byte, error) { return []byte(v), nil }
func (v *XSDGYearMonth) UnmarshalText(text []byte) error {
	s := string(text)
	if err := validatePattern(s, xsdGYearMonthPattern, "xs:gYearMonth"); err != nil {
		return err
	}
	*v = XSDGYearMonth(s)
	return nil
}
func (v XSDGYearMonth) MarshalText() ([]byte, error) { return []byte(v), nil }
func (v *XSDGMonth) UnmarshalText(text []byte) error {
	s := string(text)
	if err := validatePattern(s, xsdGMonthPattern, "xs:gMonth"); err != nil {
		return err
	}
	*v = XSDGMonth(s)
	return nil
}
func (v XSDGMonth) MarshalText() ([]byte, error) { return []byte(v), nil }
func (v *XSDGMonthDay) UnmarshalText(text []byte) error {
	s := string(text)
	if err := validatePattern(s, xsdGMonthDayPattern, "xs:gMonthDay"); err != nil {
		return err
	}
	*v = XSDGMonthDay(s)
	return nil
}
func (v XSDGMonthDay) MarshalText() ([]byte, error) { return []byte(v), nil }
func (v *XSDGDay) UnmarshalText(text []byte) error {
	s := string(text)
	if err := validatePattern(s, xsdGDayPattern, "xs:gDay"); err != nil {
		return err
	}
	*v = XSDGDay(s)
	return nil
}
func (v XSDGDay) MarshalText() ([]byte, error) { return []byte(v), nil }

func validatePattern(value string, pattern *regexp.Regexp, typeName string) error {
	if !pattern.MatchString(value) {
		return fmt.Errorf("invalid %s %q", typeName, value)
	}
	return nil
}

var (
	xsdGYearPattern      = regexp.MustCompile("^-?[0-9]{4,}(?:Z|[+-][0-9]{2}:[0-9]{2})?$")
	xsdGYearMonthPattern = regexp.MustCompile("^-?[0-9]{4,}-[0-9]{2}(?:Z|[+-][0-9]{2}:[0-9]{2})?$")
	xsdGMonthPattern     = regexp.MustCompile("^--[0-9]{2}(?:Z|[+-][0-9]{2}:[0-9]{2})?$")
	xsdGMonthDayPattern  = regexp.MustCompile("^--[0-9]{2}-[0-9]{2}(?:Z|[+-][0-9]{2}:[0-9]{2})?$")
	xsdGDayPattern       = regexp.MustCompile("^---[0-9]{2}(?:Z|[+-][0-9]{2}:[0-9]{2})?$")
)

`)
}

func renderXSIConst(buf *bytes.Buffer) {
	fmt.Fprintln(buf, `const XSINamespace = "http://www.w3.org/2001/XMLSchema-instance"`)
	fmt.Fprintln(buf, `const xsiNamespace = XSINamespace`)
	fmt.Fprintln(buf)
}

func renderQNameHelper(buf *bytes.Buffer) {
	fmt.Fprint(buf, `// QName is a namespace-aware XML qualified name used for polymorphic
// dispatch (substitution groups and xsi:type).
type QName struct {
	Namespace string
	Local     string
}

func XSITypeQName(start xml.StartElement) (QName, bool) {
	var raw string
	for _, attr := range start.Attr {
		if (attr.Name.Space == xsiNamespace && attr.Name.Local == "type") || attr.Name.Local == "xsi:type" {
			raw = attr.Value
			break
		}
	}
	if raw == "" {
		return QName{}, false
	}
	prefix, local, ok := cutQName(raw)
	if !ok {
		// Practical fallback for unprefixed xsi:type values used by many schemas:
		// interpret them in the element namespace.
		return QName{Namespace: start.Name.Space, Local: raw}, true
	}
	for _, attr := range start.Attr {
		if attr.Name.Space == "xmlns" && attr.Name.Local == prefix {
			return QName{Namespace: attr.Value, Local: local}, true
		}
	}
	// Fallback: the xsi:type prefix was likely declared on an ancestor element.
	// Go's decoder resolves element names via ancestor xmlns declarations but does
	// not include them in Attr of descendant elements. Use the element's own
	// resolved namespace as the best available fallback.
	return QName{Namespace: start.Name.Space, Local: local}, true
}

func cutQName(raw string) (string, string, bool) {
	for i, r := range raw {
		if r == ':' {
			return raw[:i], raw[i+1:], true
		}
	}
	return "", raw, false
}

`)
}

func renderNamespacePrefixHelper(buf *bytes.Buffer, file generator.File) {
	fmt.Fprintln(buf, "var namespacePrefixes = map[string]string{")
	for _, prefix := range file.NamespacePrefixes {
		fmt.Fprintf(buf, "\t%q: %q,\n", prefix.Namespace, prefix.Prefix)
	}
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
	if file.RuntimePackage != "" {
		return
	}
	fmt.Fprint(buf, `func prefixedStart(namespace, local string) xml.StartElement {
	if namespace == "" {
		return xml.StartElement{Name: xml.Name{Local: local}}
	}
	if prefix := namespacePrefixes[namespace]; prefix != "" {
		return xml.StartElement{
			Name: xml.Name{Local: prefix + ":" + local},
			Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: namespace}},
		}
	}
	return xml.StartElement{Name: xml.Name{Space: namespace, Local: local}}
}

func prefixedAttr(namespace, local, value string) []xml.Attr {
	if namespace == "" {
		return []xml.Attr{{Name: xml.Name{Local: local}, Value: value}}
	}
	if prefix := namespacePrefixes[namespace]; prefix != "" {
		return []xml.Attr{
			{Name: xml.Name{Local: "xmlns:" + prefix}, Value: namespace},
			{Name: xml.Name{Local: prefix + ":" + local}, Value: value},
		}
	}
	return []xml.Attr{{Name: xml.Name{Space: namespace, Local: local}, Value: value}}
}

`)
}

func renderAnyHelper(buf *bytes.Buffer, file generator.File) {
	fmt.Fprint(buf, `// AnyElement captures an XML element allowed by xs:any.
type AnyElement struct {
	XMLName  xml.Name
	Attr     []xml.Attr `+"`"+`xml:",any,attr"`+"`"+`
	InnerXML string     `+"`"+`xml:",innerxml"`+"`"+`
}

func (a *AnyElement) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	a.XMLName = start.Name
	a.Attr = a.Attr[:0]
	for _, attr := range start.Attr {
		if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
			continue
		}
		a.Attr = append(a.Attr, attr)
	}
	var raw struct {
		InnerXML string `+"`"+`xml:",innerxml"`+"`"+`
	}
	if err := d.DecodeElement(&raw, &start); err != nil {
		return err
	}
	a.InnerXML = raw.InnerXML
	return nil
}

// AnyAttributes captures XML attributes allowed by xs:anyAttribute.
type AnyAttributes []xml.Attr

func (a *AnyAttributes) UnmarshalXMLAttr(attr xml.Attr) error {
	if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
		return nil
	}
	*a = append(*a, attr)
	return nil
}

func WildcardNamespaceAllowed(constraint, targetNamespace, actualNamespace string) bool {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" {
		constraint = "##any"
	}
	for _, token := range strings.Fields(constraint) {
		switch token {
		case "##any":
			return true
		case "##local":
			if actualNamespace == "" {
				return true
			}
		case "##targetNamespace":
			if actualNamespace == targetNamespace {
				return true
			}
		case "##other":
			if actualNamespace != targetNamespace {
				return true
			}
		default:
			if actualNamespace == token {
				return true
			}
		}
	}
	return false
}

func wildcardNamespaceAllowed(constraint, targetNamespace, actualNamespace string) bool {
	return WildcardNamespaceAllowed(constraint, targetNamespace, actualNamespace)
}

`)
	renderWildcardDeclarationIndex(buf, file)
	fmt.Fprint(buf, `func WildcardElementProcessContentsAllowed(processContents string, name xml.Name) bool {
	switch processContents {
	case "", "skip", "lax":
		return true
	case "strict":
		if name.Local == "" {
			return false
		}
		_, ok := wildcardElementDeclarations[name]
		return ok
	default:
		return false
	}
}

func WildcardAttributeProcessContentsAllowed(processContents string, name xml.Name) bool {
	switch processContents {
	case "", "skip", "lax":
		return true
	case "strict":
		if name.Local == "" {
			return false
		}
		_, ok := wildcardAttributeDeclarations[name]
		return ok
	default:
		return false
	}
}

func wildcardElementProcessContentsAllowed(processContents string, name xml.Name) bool {
	return WildcardElementProcessContentsAllowed(processContents, name)
}

func wildcardAttributeProcessContentsAllowed(processContents string, name xml.Name) bool {
	return WildcardAttributeProcessContentsAllowed(processContents, name)
}

`)
}

func renderWildcardDeclarationIndex(buf *bytes.Buffer, file generator.File) {
	fmt.Fprintln(buf, "var wildcardElementDeclarations = map[xml.Name]struct{}{")
	for _, decl := range file.ElementDeclarations {
		fmt.Fprintf(buf, "\t{Space: %q, Local: %q}: {},\n", decl.Namespace, decl.Local)
	}
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "var wildcardAttributeDeclarations = map[xml.Name]struct{}{")
	for _, decl := range file.AttributeDeclarations {
		fmt.Fprintf(buf, "\t{Space: %q, Local: %q}: {},\n", decl.Namespace, decl.Local)
	}
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func renderRuntimePrefixHelper(buf *bytes.Buffer) {
	fmt.Fprint(buf, `func PrefixedStart(prefixes map[string]string, namespace, local string) xml.StartElement {
	if namespace == "" {
		return xml.StartElement{Name: xml.Name{Local: local}}
	}
	if prefix := prefixes[namespace]; prefix != "" {
		return xml.StartElement{
			Name: xml.Name{Local: prefix + ":" + local},
			Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns:" + prefix}, Value: namespace}},
		}
	}
	return xml.StartElement{Name: xml.Name{Space: namespace, Local: local}}
}

func PrefixedAttr(prefixes map[string]string, namespace, local, value string) []xml.Attr {
	if namespace == "" {
		return []xml.Attr{{Name: xml.Name{Local: local}, Value: value}}
	}
	if prefix := prefixes[namespace]; prefix != "" {
		return []xml.Attr{
			{Name: xml.Name{Local: "xmlns:" + prefix}, Value: namespace},
			{Name: xml.Name{Local: prefix + ":" + local}, Value: value},
		}
	}
	return []xml.Attr{{Name: xml.Name{Space: namespace, Local: local}, Value: value}}
}

`)
}

func renderNillableHelper(buf *bytes.Buffer) {
	fmt.Fprint(buf, `// Nillable represents an XML element that may carry xsi:nil="true".
type Nillable[T any] struct {
	Value T
	Nil   bool
}

func (n *Nillable[T]) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if ((attr.Name.Space == xsiNamespace && attr.Name.Local == "nil") || attr.Name.Local == "xsi:nil") && (attr.Value == "true" || attr.Value == "1") {
			n.Nil = true
			return d.Skip()
		}
	}
	return d.DecodeElement(&n.Value, &start)
}

func (n Nillable[T]) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if n.Nil {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xmlns:xsi"}, Value: xsiNamespace})
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xsi:nil"}, Value: "true"})
		if err := e.EncodeToken(start); err != nil {
			return err
		}
		return e.EncodeToken(start.End())
	}
	return e.EncodeElement(n.Value, start)
}

`)
}

func renderRegistry(buf *bytes.Buffer, file generator.File, reg generator.RegistryDecl) {
	qname := qNameType(file)
	entryType := reg.VarName + "Entry"
	fmt.Fprintf(buf, "type %s struct {\n", entryType)
	fmt.Fprintf(buf, "\tNew func() %s\n", reg.InterfaceName)
	fmt.Fprintf(buf, "\tElement %s\n", qname)
	fmt.Fprintf(buf, "\tXSIType %s\n", qname)
	fmt.Fprintln(buf, "\tUseXSIType bool")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)

	fmt.Fprintf(buf, "var %s = map[%s]%s{\n", reg.VarName, qname, entryType)
	for _, e := range reg.Entries {
		fmt.Fprintf(buf, "\t%s{Namespace: %q, Local: %q}: {New: func() %s { return &%s{} }, Element: %s{Namespace: %q, Local: %q}, XSIType: %s{Namespace: %q, Local: %q}, UseXSIType: %t},\n",
			qname, e.DispatchNS, e.DispatchLocal, reg.InterfaceName, e.ConcreteType,
			qname, e.ElementNS, e.ElementLocal, qname, e.XSITypeNS, e.XSITypeLocal, e.UseXSIType)
	}
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func qNameType(file generator.File) string {
	if file.RuntimePackage != "" {
		return file.RuntimePackage + ".QName"
	}
	return "QName"
}

func xsiTypeFunc(file generator.File) string {
	if file.RuntimePackage != "" {
		return file.RuntimePackage + ".XSITypeQName"
	}
	return "XSITypeQName"
}

func xsiNamespaceRef(file generator.File) string {
	if file.RuntimePackage != "" {
		return file.RuntimePackage + ".XSINamespace"
	}
	return "xsiNamespace"
}

func wildcardNamespaceAllowedFunc(file generator.File) string {
	if file.RuntimePackage != "" {
		return file.RuntimePackage + ".WildcardNamespaceAllowed"
	}
	return "wildcardNamespaceAllowed"
}

func wildcardElementProcessContentsFunc(file generator.File) string {
	if file.RuntimePackage != "" {
		return file.RuntimePackage + ".WildcardElementProcessContentsAllowed"
	}
	return "wildcardElementProcessContentsAllowed"
}

func wildcardAttributeProcessContentsFunc(file generator.File) string {
	if file.RuntimePackage != "" {
		return file.RuntimePackage + ".WildcardAttributeProcessContentsAllowed"
	}
	return "wildcardAttributeProcessContentsAllowed"
}

func validateFieldFunc(file generator.File) string {
	if file.RuntimePackage != "" {
		return file.RuntimePackage + ".ValidateField"
	}
	return "validateField"
}

func validatableType(file generator.File) string {
	if file.RuntimePackage != "" {
		return file.RuntimePackage + ".Validatable"
	}
	return "validatable"
}

func prefixedStartFunc(file generator.File) string {
	if file.RuntimePackage != "" {
		return file.RuntimePackage + ".PrefixedStart"
	}
	return "prefixedStart"
}

func prefixedAttrFunc(file generator.File) string {
	if file.RuntimePackage != "" {
		return file.RuntimePackage + ".PrefixedAttr"
	}
	return "prefixedAttr"
}

func renderValidationHelper(buf *bytes.Buffer) {
	buf.WriteString(`type Validatable interface {
	Validate() error
}

type validatable = Validatable

func ValidateField(name string, value any) error {
	if value == nil {
		return nil
	}
	if v, ok := value.(Validatable); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("field %s: %w", name, err)
		}
		return nil
	}
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		for i := 0; i < rv.Len(); i++ {
			if err := ValidateField(fmt.Sprintf("%s[%d]", name, i), rv.Index(i).Interface()); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateField(name string, value any) error {
	return ValidateField(name, value)
}

`)
}
