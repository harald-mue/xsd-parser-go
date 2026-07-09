package optimizer

import (
	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func resolveTypedefs(meta interpreter.MetaTypes) interpreter.MetaTypes {
	typedefs := buildTypedefMap(meta.Types)
	for i := range meta.Types {
		resolveMetaType(&meta.Types[i], typedefs)
	}
	for i := range meta.Interfaces {
		for j := range meta.Interfaces[i].Members {
			member := &meta.Interfaces[i].Members[j]
			if q := resolveQName(typedefs, schema.QName{Namespace: member.ConcreteTypeNS, Local: member.ConcreteTypeLocal}); q.Local != "" {
				member.ConcreteTypeNS = q.Namespace
				member.ConcreteTypeLocal = q.Local
			}
		}
	}
	return meta
}

func buildTypedefMap(types []interpreter.MetaType) map[typeKey]typeKey {
	typedefs := make(map[typeKey]typeKey)
	for _, typ := range types {
		key := typeKey{namespace: typ.Namespace, local: typ.Name}
		if typ.DuplicateOf.Local != "" {
			typedefs[key] = typeKeyFromQName(typ.DuplicateOf)
			continue
		}
		if typ.Kind != "simpleType" || typ.Alias == nil || len(typ.Alias.Facets) > 0 || isXSDNamespace(typ.Alias.BaseName.Namespace) {
			continue
		}
		if typ.Alias.BaseName.Local == "" {
			continue
		}
		typedefs[key] = typeKey{
			namespace: typ.Alias.BaseName.Namespace,
			local:     typ.Alias.BaseName.Local,
		}
	}
	return typedefs
}

func resolveQName(typedefs map[typeKey]typeKey, qname schema.QName) schema.QName {
	if qname.Local == "" {
		return qname
	}
	resolved := resolveKey(typedefs, typeKey{namespace: qname.Namespace, local: qname.Local})
	if resolved.local == "" {
		return qname
	}
	return schema.QName{
		Raw:       resolved.local,
		Local:     resolved.local,
		Namespace: resolved.namespace,
	}
}

func resolveKey(typedefs map[typeKey]typeKey, key typeKey) typeKey {
	seen := make(map[typeKey]struct{})
	for {
		next, ok := typedefs[key]
		if !ok {
			return key
		}
		if _, loop := seen[key]; loop {
			return key
		}
		seen[key] = struct{}{}
		key = next
	}
}

func resolveMetaType(typ *interpreter.MetaType, typedefs map[typeKey]typeKey) {
	if typ.TypeName.Local != "" {
		typ.TypeName = resolveQName(typedefs, typ.TypeName)
	}
	if typ.Alias != nil {
		typ.Alias.BaseName = resolveQName(typedefs, typ.Alias.BaseName)
	}
	if typ.List != nil && typ.List.ItemName.Local != "" {
		typ.List.ItemName = resolveQName(typedefs, typ.List.ItemName)
	}
	if typ.Union != nil {
		for i := range typ.Union.MemberNames {
			typ.Union.MemberNames[i] = resolveQName(typedefs, typ.Union.MemberNames[i])
		}
	}
	for i := range typ.Fields {
		field := &typ.Fields[i]
		if field.TypeName.Local == "" || field.Chardata {
			// Preserve simple-content content types (restricted/enumerated simple types).
			continue
		}
		field.TypeName = resolveQName(typedefs, field.TypeName)
	}
}
