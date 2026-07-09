package optimizer

import (
	"reflect"

	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func removeDuplicates(meta interpreter.MetaTypes) interpreter.MetaTypes {
	changed := true
	for changed {
		changed = false
		canonical := make(map[typeKey]schema.QName)
		for i := range meta.Types {
			typ := &meta.Types[i]
			if !isDuplicateCandidate(*typ) {
				continue
			}
			key := typeKey{namespace: typ.Namespace, local: typ.Name}
			for otherKey, otherQName := range canonical {
				other := findMetaType(meta.Types, otherKey)
				if other == nil {
					continue
				}
				if metaTypesEqual(*typ, *other) {
					markDuplicate(typ, otherQName)
					changed = true
					break
				}
			}
			if typ.DuplicateOf.Local == "" {
				canonical[key] = schema.QName{Namespace: typ.Namespace, Local: typ.Name}
			}
		}
	}
	return meta
}

func isDuplicateCandidate(typ interpreter.MetaType) bool {
	if typ.DuplicateOf.Local != "" {
		return false
	}
	if typ.Kind != "complexType" && typ.Kind != "simpleType" {
		return false
	}
	if typ.CustomXML != nil || len(typ.PolymorphicImpls) > 0 || typ.IsSubstitutionHead {
		return false
	}
	if typ.Mixed {
		return false
	}
	return true
}

func metaTypesEqual(a, b interpreter.MetaType) bool {
	if a.Kind != b.Kind || a.Namespace != b.Namespace || a.Name == b.Name {
		return false
	}
	if a.Mixed != b.Mixed || a.Nillable != b.Nillable {
		return false
	}
	if !metaFieldsEqual(a.Fields, b.Fields) {
		return false
	}
	if !metaAliasEqual(a.Alias, b.Alias) {
		return false
	}
	if !metaListEqual(a.List, b.List) {
		return false
	}
	if !metaUnionEqual(a.Union, b.Union) {
		return false
	}
	return true
}

func metaFieldsEqual(a, b []interpreter.MetaField) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !metaFieldEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func metaFieldEqual(a, b interpreter.MetaField) bool {
	return a.Name == b.Name &&
		a.XMLName == b.XMLName &&
		a.Namespace == b.Namespace &&
		a.TypeRef == b.TypeRef &&
		a.TypeName == b.TypeName &&
		a.Attribute == b.Attribute &&
		a.AnyAttribute == b.AnyAttribute &&
		a.AnyElement == b.AnyElement &&
		a.AnyNamespace == b.AnyNamespace &&
		a.AnyProcessContents == b.AnyProcessContents &&
		a.AnyTargetNamespace == b.AnyTargetNamespace &&
		a.Chardata == b.Chardata &&
		a.Choice == b.Choice &&
		a.Nillable == b.Nillable &&
		a.Required == b.Required &&
		a.MinOccurs == b.MinOccurs &&
		a.MaxOccurs == b.MaxOccurs &&
		a.Default == b.Default &&
		a.Fixed == b.Fixed &&
		reflect.DeepEqual(a.Polymorphic, b.Polymorphic)
}

func metaAliasEqual(a, b *interpreter.MetaAlias) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.BaseRef == b.BaseRef &&
		a.BaseName == b.BaseName &&
		reflect.DeepEqual(a.Facets, b.Facets)
}

func metaListEqual(a, b *interpreter.MetaList) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.ItemRef == b.ItemRef && a.ItemName == b.ItemName
}

func metaUnionEqual(a, b *interpreter.MetaUnion) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.MemberRefs == b.MemberRefs && reflect.DeepEqual(a.MemberNames, b.MemberNames)
}
