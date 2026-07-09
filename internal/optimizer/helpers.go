package optimizer

import (
	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

type typeKey struct {
	namespace string
	local     string
}

func typeKeyFromQName(q schema.QName) typeKey {
	return typeKey{namespace: q.Namespace, local: q.Local}
}

func (k typeKey) qname() schema.QName {
	return schema.QName{Namespace: k.namespace, Local: k.local}
}

func findMetaType(types []interpreter.MetaType, key typeKey) *interpreter.MetaType {
	for i := range types {
		if types[i].Namespace == key.namespace && types[i].Name == key.local {
			return &types[i]
		}
	}
	return nil
}

func isXSDNamespace(namespace string) bool {
	return namespace == "" || namespace == "http://www.w3.org/2001/XMLSchema"
}

func hasEnumerationFacets(facets []schema.Facet) bool {
	for _, facet := range facets {
		if facet.Name == "enumeration" {
			return true
		}
	}
	return false
}

func onlyEnumerationFacets(facets []schema.Facet) bool {
	if len(facets) == 0 {
		return false
	}
	for _, facet := range facets {
		if facet.Name != "enumeration" {
			return false
		}
	}
	return true
}

func clearTypeBody(typ *interpreter.MetaType) {
	typ.Fields = nil
	typ.Alias = nil
	typ.List = nil
	typ.Union = nil
	typ.CustomXML = nil
	typ.PolymorphicImpls = nil
}

func markDuplicate(typ *interpreter.MetaType, target schema.QName) {
	typ.DuplicateOf = target
	clearTypeBody(typ)
}

func scalarBucket(q schema.QName) string {
	if !isXSDNamespace(q.Namespace) {
		return q.Namespace + "\x00" + q.Local
	}
	switch q.Local {
	case "string", "token", "normalizedString", "anySimpleType", "anyURI", "QName", "ID", "IDREF", "IDREFS", "NCName", "Name", "NMTOKEN", "NMTOKENS", "language", "date", "dateTime", "time", "duration", "decimal", "hexBinary", "base64Binary", "gYear", "gYearMonth", "gMonth", "gMonthDay", "gDay":
		return "xsd:string"
	case "boolean", "bool":
		return "xsd:bool"
	case "byte":
		return "xsd:int8"
	case "unsignedByte":
		return "xsd:uint8"
	case "short":
		return "xsd:int16"
	case "unsignedShort":
		return "xsd:uint16"
	case "int", "integer", "nonPositiveInteger", "negativeInteger", "long", "nonNegativeInteger", "positiveInteger", "unsignedInt", "unsignedLong":
		return "xsd:int"
	case "float":
		return "xsd:float32"
	case "double":
		return "xsd:float64"
	default:
		return "xsd:" + q.Local
	}
}

func unionMemberKey(types []interpreter.MetaType, typedefs map[typeKey]typeKey, q schema.QName) string {
	q = resolveQName(typedefs, q)
	if typ := findMetaType(types, typeKeyFromQName(q)); typ != nil && typ.DuplicateOf.Local != "" {
		return unionMemberKey(types, typedefs, typ.DuplicateOf)
	}
	return scalarBucket(q)
}

func unrestrictedQName(types []interpreter.MetaType, typedefs map[typeKey]typeKey, q schema.QName) schema.QName {
	q = resolveQName(typedefs, q)
	for {
		if isXSDNamespace(q.Namespace) {
			return q
		}
		typ := findMetaType(types, typeKeyFromQName(q))
		if typ == nil || typ.Alias == nil || typ.Alias.BaseName.Local == "" {
			return q
		}
		next := resolveQName(typedefs, typ.Alias.BaseName)
		if next == q {
			return q
		}
		q = next
	}
}
