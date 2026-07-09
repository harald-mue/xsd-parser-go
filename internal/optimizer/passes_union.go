package optimizer

import (
	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func (o *Optimizer) removeDuplicateUnionVariants() {
	typedefs := o.getTypedefs()
	for i := range o.meta.Types {
		typ := &o.meta.Types[i]
		if typ.Kind != "simpleType" || typ.Union == nil || typ.DuplicateOf.Local != "" {
			continue
		}
		seen := make(map[string]struct{})
		filtered := typ.Union.MemberNames[:0]
		for _, member := range typ.Union.MemberNames {
			key := unionMemberKey(o.meta.Types, typedefs, member)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			filtered = append(filtered, member)
		}
		typ.Union.MemberNames = filtered
	}
}

func (o *Optimizer) removeEmptyUnions() {
	for i := range o.meta.Types {
		typ := &o.meta.Types[i]
		if typ.Kind != "simpleType" || typ.Union == nil || typ.DuplicateOf.Local != "" {
			continue
		}
		if len(typ.Union.MemberNames) > 1 {
			continue
		}
		var base schema.QName
		if len(typ.Union.MemberNames) == 1 {
			base = typ.Union.MemberNames[0]
		}
		if base.Local == "" {
			continue
		}
		o.invalidateTypedefs()
		markDuplicate(typ, base)
	}
}

func (o *Optimizer) flattenUnions() {
	for i := range o.meta.Types {
		o.flattenUnion(&o.meta.Types[i])
	}
}

func (o *Optimizer) flattenUnion(typ *interpreter.MetaType) {
	if typ.Kind != "simpleType" || typ.Union == nil || typ.DuplicateOf.Local != "" {
		return
	}
	typedefs := o.getTypedefs()
	var flat []schema.QName
	for _, member := range typ.Union.MemberNames {
		flat = append(flat, o.flattenUnionMember(member, typedefs)...)
	}
	typ.Union.MemberNames = flat
}

func (o *Optimizer) flattenUnionMember(member schema.QName, typedefs map[typeKey]typeKey) []schema.QName {
	member = resolveQName(typedefs, member)
	ref := findMetaType(o.meta.Types, typeKeyFromQName(member))
	if ref == nil || ref.DuplicateOf.Local != "" {
		if ref != nil && ref.DuplicateOf.Local != "" {
			return o.flattenUnionMember(ref.DuplicateOf, typedefs)
		}
		return []schema.QName{member}
	}
	if ref.Union != nil {
		var flat []schema.QName
		for _, nested := range ref.Union.MemberNames {
			flat = append(flat, o.flattenUnionMember(nested, typedefs)...)
		}
		return flat
	}
	return []schema.QName{member}
}

func (o *Optimizer) mergeEnumUnions() {
	for i := range o.meta.Types {
		o.mergeEnumUnion(&o.meta.Types[i])
	}
}

func (o *Optimizer) mergeEnumUnion(typ *interpreter.MetaType) {
	if typ.Kind != "simpleType" || typ.Union == nil || typ.DuplicateOf.Local != "" {
		return
	}
	typedefs := o.getTypedefs()

	var enumFacets []schema.Facet
	var unionMembers []schema.QName
	seenUnion := make(map[string]struct{})

	for _, member := range typ.Union.MemberNames {
		o.collectEnumUnionMember(typ, member, typedefs, &enumFacets, &unionMembers, seenUnion)
	}

	if len(enumFacets) == 0 {
		return
	}

	if len(unionMembers) == 0 {
		base := schema.QName{Local: "string", Namespace: "http://www.w3.org/2001/XMLSchema"}
		for _, member := range typ.Union.MemberNames {
			member = resolveQName(typedefs, member)
			if ref := findMetaType(o.meta.Types, typeKeyFromQName(member)); ref != nil && ref.Alias != nil && ref.Alias.BaseName.Local != "" {
				base = ref.Alias.BaseName
				break
			}
		}
		typ.Union = nil
		typ.Alias = &interpreter.MetaAlias{
			BaseName: base,
			Facets:   enumFacets,
		}
		return
	}

	// Mixed union: inline enum facets on the union type and replace enum members
	// with their unrestricted scalar bases.
	typ.Alias = &interpreter.MetaAlias{
		BaseName: schema.QName{Local: "string", Namespace: "http://www.w3.org/2001/XMLSchema"},
		Facets:   enumFacets,
	}
	typ.Union.MemberNames = unionMembers
}

func (o *Optimizer) collectEnumUnionMember(
	parent *interpreter.MetaType,
	member schema.QName,
	typedefs map[typeKey]typeKey,
	enumFacets *[]schema.Facet,
	unionMembers *[]schema.QName,
	seenUnion map[string]struct{},
) {
	member = resolveQName(typedefs, member)
	ref := findMetaType(o.meta.Types, typeKeyFromQName(member))
	if ref != nil && ref.DuplicateOf.Local != "" {
		o.collectEnumUnionMember(parent, ref.DuplicateOf, typedefs, enumFacets, unionMembers, seenUnion)
		return
	}
	if ref != nil && ref.Union != nil {
		for _, nested := range ref.Union.MemberNames {
			o.collectEnumUnionMember(parent, nested, typedefs, enumFacets, unionMembers, seenUnion)
		}
		return
	}
	if ref != nil && ref.Alias != nil && onlyEnumerationFacets(ref.Alias.Facets) {
		*enumFacets = append(*enumFacets, ref.Alias.Facets...)
		if ref.Name != parent.Name || ref.Namespace != parent.Namespace {
			markDuplicate(ref, schema.QName{Namespace: parent.Namespace, Local: parent.Name})
		}
		return
	}
	key := unionMemberKey(o.meta.Types, typedefs, member)
	if _, ok := seenUnion[key]; ok {
		return
	}
	seenUnion[key] = struct{}{}
	*unionMembers = append(*unionMembers, member)
}
