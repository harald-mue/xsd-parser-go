package generator

import (
	"net/url"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func needsCustomXML(file File) bool {
	for _, typ := range file.Types {
		if typ.CustomXML != nil {
			return true
		}
	}
	return false
}

func needsNillable(file File) bool {
	for _, typ := range file.Types {
		if strings.Contains(typ.Alias, "AnyElement") {
			return true
		}
		if typ.AliasEqual && strings.HasPrefix(typ.Alias, "Nillable[") {
			return true
		}
		for _, field := range typ.Fields {
			if field.Nillable {
				return true
			}
		}
	}
	return false
}

func needsAny(file File) bool {
	for _, typ := range file.Types {
		for _, field := range typ.Fields {
			if field.AnyElement || field.AnyAttribute || strings.Contains(field.Type, "AnyElement") {
				return true
			}
		}
	}
	return false
}

func collectNamespacePrefixes(file File, options Options) []NamespacePrefixDecl {
	seen := make(map[string]bool)
	var namespaces []string
	add := func(ns string) {
		if ns == "" || seen[ns] {
			return
		}
		seen[ns] = true
		namespaces = append(namespaces, ns)
	}
	for _, typ := range file.Types {
		if typ.XMLName != "" {
			if ns, _, ok := strings.Cut(typ.XMLName, " "); ok {
				add(ns)
			}
		}
		for _, field := range typ.Fields {
			add(field.XMLNamespace)
		}
		if typ.CustomXML != nil {
			add(typ.CustomXML.ElementNS)
		}
		if typ.Choice != nil {
			for _, variant := range typ.Choice.Variants {
				add(variant.XMLNamespace)
			}
		}
	}
	for _, registry := range file.Registries {
		for _, entry := range registry.Entries {
			add(entry.ElementNS)
			add(entry.XSITypeNS)
		}
	}
	sort.Strings(namespaces)
	prefixes := make([]NamespacePrefixDecl, 0, len(namespaces))
	used := make(map[string]int)
	for i, ns := range namespaces {
		prefix := options.NamespacePrefixes[ns]
		if prefix == "" {
			prefix = "tns"
			if i > 0 {
				prefix = namespacePrefix(ns)
				if prefix == "" {
					prefix = "ns"
				}
				prefix = strings.ToLower(prefix[:1]) + prefix[1:]
			}
		}
		prefix = uniqueName(prefix, used)
		prefixes = append(prefixes, NamespacePrefixDecl{Namespace: ns, Prefix: prefix})
	}
	return prefixes
}

func needsValidation(file File) (bool, bool) {
	needs := false
	needsRegexp := false
	for _, typ := range file.Types {
		if typ.Validate || typ.Union != nil {
			needs = true
		}
		for _, facet := range typ.Facets {
			needs = true
			if facet.Name == "pattern" {
				needsRegexp = true
			}
		}
	}
	return needs, needsRegexp
}

func needsStrings(file File) bool {
	for _, typ := range file.Types {
		if typ.List != nil {
			return true
		}
	}
	return false
}

func needsValidationHelper(file File) bool {
	for _, typ := range file.Types {
		if typ.Validate || typ.Union != nil {
			return true
		}
	}
	return false
}

func aliasBaseName(metaType interpreter.MetaType) schema.QName {
	if metaType.Alias != nil {
		return metaType.Alias.BaseName
	}
	return schema.QName{Local: "string", Namespace: "http://www.w3.org/2001/XMLSchema"}
}

func scalarType(qname schema.QName) string {
	switch qname.Local {
	case "anyType":
		return "AnyElement"
	case "string", "token", "normalizedString", "anySimpleType", "anyURI", "QName", "NCName", "Name", "NMTOKEN", "NMTOKENS", "language":
		return "string"
	case "ID":
		return "XSDID"
	case "IDREF":
		return "XSDIDREF"
	case "IDREFS":
		return "XSDIDREFS"
	case "dateTime":
		return "XSDDateTime"
	case "date":
		return "XSDDate"
	case "time":
		return "XSDTime"
	case "duration":
		return "XSDDuration"
	case "decimal":
		return "XSDDecimal"
	case "hexBinary":
		return "XSDHexBinary"
	case "base64Binary":
		return "XSDBase64Binary"
	case "gYear":
		return "XSDGYear"
	case "gYearMonth":
		return "XSDGYearMonth"
	case "gMonth":
		return "XSDGMonth"
	case "gMonthDay":
		return "XSDGMonthDay"
	case "gDay":
		return "XSDGDay"
	case "boolean", "bool":
		return "bool"
	case "byte":
		return "int8"
	case "unsignedByte":
		return "uint8"
	case "short":
		return "int16"
	case "unsignedShort":
		return "uint16"
	case "int", "integer", "nonPositiveInteger", "negativeInteger", "long", "nonNegativeInteger", "positiveInteger", "unsignedInt", "unsignedLong":
		return "int"
	case "float":
		return "float32"
	case "double":
		return "float64"
	default:
		if qname.Local == "" {
			return "string"
		}
		return goName(qname.Local)
	}
}

func isXSDNamespace(namespace string) bool {
	return namespace == "" || namespace == "http://www.w3.org/2001/XMLSchema"
}

func namespacePrefix(namespace string) string {
	parts := namespaceParts(namespace)
	if len(parts) == 0 {
		return "Global"
	}
	for _, part := range parts {
		if !isCommonNamespacePart(part) {
			return goName(part)
		}
	}
	return goName(parts[0])
}

func namespaceParts(namespace string) []string {
	if parsed, err := url.Parse(namespace); err == nil && parsed.Host != "" {
		labels := strings.Split(parsed.Host, ".")
		labels = append(labels, strings.FieldsFunc(parsed.Path, isNamespaceSeparator)...)
		return cleanNamespaceParts(labels)
	}
	return cleanNamespaceParts(strings.FieldsFunc(namespace, isNamespaceSeparator))
}

func cleanNamespaceParts(parts []string) []string {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			clean = append(clean, part)
		}
	}
	return clean
}

func isNamespaceSeparator(r rune) bool {
	return r == ':' || r == '/' || r == '#' || r == '?' || r == '&' || r == '=' || r == '.' || r == '-' || r == '_' || r == ' '
}

func isCommonNamespacePart(part string) bool {
	switch strings.ToLower(part) {
	case "", "www", "com", "org", "net", "schema", "schemas", "xml", "xsd", "www3", "w3":
		return true
	default:
		return false
	}
}

func goName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = upperFirst(part)
	}
	joined := strings.Join(parts, "")
	if joined == "" {
		return ""
	}
	r, _ := utf8.DecodeRuneInString(joined)
	if unicode.IsDigit(r) {
		return "Value" + joined
	}
	if !unicode.IsLetter(r) && r != '_' {
		return "Value" + joined
	}
	return joined
}

func upperFirst(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 0 {
		return ""
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

func strconvSuffix(n int) string {
	// n is expected to be tiny for duplicate schema property names. Avoid pulling
	// string rendering concerns into the semantic model.
	if n >= 0 && n <= 9 {
		return string(rune('0' + n))
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}

type declarationDeclSet struct {
	elements   []XMLNameDecl
	attributes []XMLNameDecl
}

func declarationDecls(index interpreter.DeclarationIndex) declarationDeclSet {
	elements := make([]XMLNameDecl, 0, len(index.Elements))
	for _, q := range index.Elements {
		elements = append(elements, XMLNameDecl{Namespace: q.Namespace, Local: q.Local})
	}
	attributes := make([]XMLNameDecl, 0, len(index.Attributes))
	for _, q := range index.Attributes {
		attributes = append(attributes, XMLNameDecl{Namespace: q.Namespace, Local: q.Local})
	}
	sort.Slice(elements, func(i, j int) bool {
		if elements[i].Namespace != elements[j].Namespace {
			return elements[i].Namespace < elements[j].Namespace
		}
		return elements[i].Local < elements[j].Local
	})
	sort.Slice(attributes, func(i, j int) bool {
		if attributes[i].Namespace != attributes[j].Namespace {
			return attributes[i].Namespace < attributes[j].Namespace
		}
		return attributes[i].Local < attributes[j].Local
	})
	return declarationDeclSet{elements: elements, attributes: attributes}
}
