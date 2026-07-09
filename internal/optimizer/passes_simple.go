package optimizer

import (
	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func (o *Optimizer) removeEmptyEnumVariants() {
	if o.enumOnlyTypes == nil {
		o.enumOnlyTypes = make(map[typeKey]struct{})
	}
	for i := range o.meta.Types {
		typ := &o.meta.Types[i]
		if typ.Kind != "simpleType" || typ.Alias == nil {
			continue
		}
		if onlyEnumerationFacets(typ.Alias.Facets) {
			o.enumOnlyTypes[typeKey{namespace: typ.Namespace, local: typ.Name}] = struct{}{}
		}
		filtered := typ.Alias.Facets[:0]
		for _, facet := range typ.Alias.Facets {
			if facet.Name == "enumeration" && facet.Value == "" {
				continue
			}
			filtered = append(filtered, facet)
		}
		typ.Alias.Facets = filtered
	}
}

func (o *Optimizer) removeEmptyEnums() {
	for i := range o.meta.Types {
		typ := &o.meta.Types[i]
		if typ.Kind != "simpleType" || typ.Alias == nil || typ.DuplicateOf.Local != "" {
			continue
		}
		if typ.List != nil || typ.Union != nil {
			continue
		}
		key := typeKey{namespace: typ.Namespace, local: typ.Name}
		if _, ok := o.enumOnlyTypes[key]; !ok {
			continue
		}
		if hasEnumerationFacets(typ.Alias.Facets) {
			continue
		}
		if typ.Alias.BaseName.Local == "" {
			continue
		}
		o.invalidateTypedefs()
		markDuplicate(typ, typ.Alias.BaseName)
	}
}

func (o *Optimizer) useUnrestrictedBaseType() {
	typedefs := o.getTypedefs()
	for i := range o.meta.Types {
		typ := &o.meta.Types[i]
		if typ.DuplicateOf.Local != "" {
			continue
		}
		switch typ.Kind {
		case "simpleType":
			o.useUnrestrictedSimpleType(typ, typedefs)
		case "complexType":
			if o.flags&FlagUseUnrestrictedBaseComplex != 0 {
				o.useUnrestrictedComplexType(typ, typedefs)
			}
		}
	}
}

func (o *Optimizer) useUnrestrictedSimpleType(typ *interpreter.MetaType, typedefs map[typeKey]typeKey) {
	if typ.List != nil {
		return
	}
	if typ.Union != nil {
		if o.flags&FlagUseUnrestrictedBaseUnion == 0 {
			return
		}
		if len(typ.Union.MemberNames) == 0 {
			return
		}
		base := unrestrictedQName(o.meta.Types, typedefs, typ.Union.MemberNames[0])
		o.invalidateTypedefs()
		markDuplicate(typ, base)
		return
	}
	if typ.Alias == nil {
		return
	}
	if hasEnumerationFacets(typ.Alias.Facets) {
		if o.flags&FlagUseUnrestrictedBaseEnum == 0 {
			return
		}
		base := unrestrictedQName(o.meta.Types, typedefs, typ.Alias.BaseName)
		o.invalidateTypedefs()
		markDuplicate(typ, base)
		return
	}
	if o.flags&FlagUseUnrestrictedBaseSimple == 0 {
		return
	}
	if len(typ.Alias.Facets) > 0 {
		return
	}
	base := unrestrictedQName(o.meta.Types, typedefs, typ.Alias.BaseName)
	if base.Local == typ.Name && base.Namespace == typ.Namespace {
		return
	}
	o.invalidateTypedefs()
	markDuplicate(typ, base)
}

func (o *Optimizer) useUnrestrictedComplexType(typ *interpreter.MetaType, typedefs map[typeKey]typeKey) {
	// Complex restrictions are already interpreted into fields in Go; only
	// duplicate-marked extension-only shells can be collapsed here.
	if len(typ.Fields) == 0 {
		return
	}
	for _, field := range typ.Fields {
		if field.Chardata || field.AnyElement {
			return
		}
	}
	// Without an explicit complex base chain in meta, this pass is a no-op.
	_ = typedefs
}

func (o *Optimizer) replaceXSAnyType() {
	anyType := schema.QName{Local: "anyType", Namespace: "http://www.w3.org/2001/XMLSchema"}
	for i := range o.meta.Types {
		typ := &o.meta.Types[i]
		for j := range typ.Fields {
			field := &typ.Fields[j]
			if field.AnyElement {
				continue
			}
			if field.TypeName == anyType || (isXSDNamespace(field.TypeName.Namespace) && field.TypeName.Local == "anyType") {
				field.AnyElement = true
				field.TypeName = schema.QName{}
				field.TypeRef = ""
			}
		}
	}
}

// mergeDynamicTypes is a no-op in Go because polymorphism is modeled via
// MetaInterfaces during interpretation.
func (o *Optimizer) mergeDynamicTypes() {}

// convertDynamicsToChoices is a no-op in Go because dynamic types are lowered
// to interfaces rather than choice enums.
func (o *Optimizer) convertDynamicsToChoices() {}

// flattenComplexTypes is a no-op in Go because the interpreter already flattens
// nested particle groups into MetaField slices.
func (o *Optimizer) flattenComplexTypes() {}

// mergeChoiceCardinalities is a no-op in Go because choice cardinality is
// propagated onto fields during interpretation and consumed by the generator.
func (o *Optimizer) mergeChoiceCardinalities() {}

// simplifyMixedTypes is a no-op in Go because mixed content is lowered in the
// generator via MixedContentDecl.
func (o *Optimizer) simplifyMixedTypes() {}
