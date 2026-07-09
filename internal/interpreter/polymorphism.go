package interpreter

import (
	"unicode"

	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

// buildSubstitutionGroups computes, for each substitution-group head element
// key, the transitive set of member elements that substitute for it.
func buildSubstitutionGroups(schemas schema.Schemas) map[typeKey][]substMember {
	direct := map[typeKey][]typeKey{}
	elementByKey := map[typeKey]schema.Element{}
	for _, file := range schemas.Files {
		for _, el := range file.Elements {
			if el.Name == "" {
				continue
			}
			k := typeKey{namespace: file.TargetNamespace, local: el.Name}
			elementByKey[k] = el
			if el.SubstitutionGroupName.Local != "" {
				head := typeKey{namespace: el.SubstitutionGroupName.Namespace, local: el.SubstitutionGroupName.Local}
				direct[head] = append(direct[head], k)
			}
		}
	}

	result := map[typeKey][]substMember{}
	for head := range direct {
		seen := map[typeKey]struct{}{}
		var stack []typeKey
		stack = append(stack, direct[head]...)
		for len(stack) > 0 {
			m := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			result[head] = append(result[head], substMember{elementKey: m, element: elementByKey[m]})
			// transitive: members of m (when m is itself a head)
			stack = append(stack, direct[m]...)
		}
	}
	return result
}

func buildTypeBase(schemas schema.Schemas) map[typeKey]typeKey {
	baseOf := map[typeKey]typeKey{}
	for _, file := range schemas.Files {
		for _, ct := range file.ComplexTypes {
			if ct.Name == "" || ct.Extension == nil || ct.Extension.BaseName.Local == "" {
				continue
			}
			derived := typeKey{namespace: file.TargetNamespace, local: ct.Name}
			baseOf[derived] = typeKey{namespace: ct.Extension.BaseName.Namespace, local: ct.Extension.BaseName.Local}
		}
	}
	return baseOf
}

func buildDerivedTypes(schemas schema.Schemas) map[typeKey][]typeKey {
	direct := map[typeKey][]typeKey{}
	for _, file := range schemas.Files {
		for _, ct := range file.ComplexTypes {
			if ct.Name == "" || ct.Extension == nil || ct.Extension.BaseName.Local == "" {
				continue
			}
			base := typeKey{namespace: ct.Extension.BaseName.Namespace, local: ct.Extension.BaseName.Local}
			derived := typeKey{namespace: file.TargetNamespace, local: ct.Name}
			direct[base] = append(direct[base], derived)
		}
	}

	result := map[typeKey][]typeKey{}
	for base := range direct {
		seen := map[typeKey]struct{}{}
		stack := append([]typeKey(nil), direct[base]...)
		for len(stack) > 0 {
			d := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if _, ok := seen[d]; ok {
				continue
			}
			seen[d] = struct{}{}
			result[base] = append(result[base], d)
			stack = append(stack, direct[d]...)
		}
	}
	return result
}

// fieldPolymorphic returns polymorphic metadata when an element points at
// either a substitution-group head or a type with known derived types.
func fieldPolymorphic(ctx context, headKey typeKey, useElement schema.Element) *FieldPolymorphic {
	_, hasSubst := ctx.substs[headKey]
	baseType := useElement.TypeName
	baseKey := typeKey{namespace: baseType.Namespace, local: baseType.Local}
	derived := ctx.derivedTypes[baseKey]
	if !hasSubst && len(derived) == 0 {
		return nil
	}
	ctx.usedHeads[headKey] = true
	if len(derived) > 0 {
		ctx.usedDynamicHeads[headKey] = baseKey
	}
	repeated := useElement.MaxOccurs == schema.OccursUnbounded || useElement.MaxOccurs > 1
	return &FieldPolymorphic{
		InterfaceName: exportGoName(headKey.local),
		RegistryVar:   registryVar(headKey.local),
		Repeated:      repeated,
	}
}

// localFieldPolymorphic returns polymorphic metadata for local element slots
// whose declared type is an abstract complex type with derived implementations.
// Unlike substitution groups, the wire element remains the local slot name and
// concrete implementations are selected through xsi:type.
func localFieldPolymorphic(ctx context, headKey typeKey, useElement schema.Element) *FieldPolymorphic {
	baseKey := typeKey{namespace: useElement.TypeName.Namespace, local: useElement.TypeName.Local}
	base, ok := ctx.complexTypes[baseKey]
	if !ok || !base.type_.Abstract || len(ctx.derivedTypes[baseKey]) == 0 {
		return nil
	}
	return fieldPolymorphic(ctx, headKey, useElement)
}

// enrichPolymorphism adds interface metadata, marks concrete implementations,
// and flags container types that need custom MarshalXML/UnmarshalXML. Only
// substitution-group heads actually referenced by a polymorphic field produce
// an interface.
func enrichPolymorphism(meta *MetaTypes, ctx context) {
	heads := sortedKeys(ctx.usedHeads)

	for _, headKey := range heads {
		members := ctx.substs[headKey]
		headEl, hasHead := ctx.elements[headKey]
		interfaceName := exportGoName(headKey.local)
		marker := "Is" + interfaceName
		registryVar := registryVar(headKey.local)

		iface := MetaInterface{
			Name:         interfaceName,
			MarkerMethod: marker,
			RegistryVar:  registryVar,
			HeadNS:       headKey.namespace,
			HeadLocal:    headKey.local,
		}

		type implTarget struct {
			typeNS, typeLocal       string
			elementNS, elementLocal string
		}
		var impls []implTarget
		seenDispatch := map[typeKey]bool{}

		addSubstitutionMember := func(el schema.Element, elNS, elLocal string, head bool) {
			if el.TypeName.Local == "" || isXSDNamespace(el.TypeName.Namespace) {
				return
			}
			dispatch := typeKey{namespace: elNS, local: elLocal}
			if seenDispatch[dispatch] {
				return
			}
			seenDispatch[dispatch] = true
			iface.Members = append(iface.Members, MetaInterfaceMember{
				DispatchNS:        elNS,
				DispatchLocal:     elLocal,
				ElementNS:         elNS,
				ElementLocal:      elLocal,
				ConcreteTypeNS:    el.TypeName.Namespace,
				ConcreteTypeLocal: el.TypeName.Local,
				Head:              head,
			})
			impls = append(impls, implTarget{typeNS: el.TypeName.Namespace, typeLocal: el.TypeName.Local, elementNS: elNS, elementLocal: elLocal})
		}

		addDynamicMember := func(tk typeKey, head bool) {
			if tk.local == "" || seenDispatch[tk] || isAbstractComplexType(ctx, tk) {
				return
			}
			seenDispatch[tk] = true
			elementNS, elementLocal := dynamicElementForType(ctx, headKey, tk, members)
			iface.Members = append(iface.Members, MetaInterfaceMember{
				DispatchNS:        tk.namespace,
				DispatchLocal:     tk.local,
				ElementNS:         elementNS,
				ElementLocal:      elementLocal,
				ConcreteTypeNS:    tk.namespace,
				ConcreteTypeLocal: tk.local,
				XSITypeNS:         tk.namespace,
				XSITypeLocal:      tk.local,
				UseXSIType:        !head,
				Head:              head,
			})
			impls = append(impls, implTarget{typeNS: tk.namespace, typeLocal: tk.local, elementNS: elementNS, elementLocal: elementLocal})
		}

		if hasHead && !headEl.element.Abstract {
			addSubstitutionMember(headEl.element, headKey.namespace, headKey.local, true)
		}
		for _, member := range members {
			if member.element.Name == "" {
				continue
			}
			addSubstitutionMember(member.element, member.elementKey.namespace, member.element.Name, false)
		}
		if baseType, ok := ctx.usedDynamicHeads[headKey]; ok {
			if hasHead && !headEl.element.Abstract {
				addDynamicMember(baseType, true)
			}
			for _, derived := range ctx.derivedTypes[baseType] {
				addDynamicMember(derived, false)
			}
		}
		meta.Interfaces = append(meta.Interfaces, iface)

		// Skip element aliases for the head and all members; the interface and
		// concrete structs own those names.
		skipAlias := map[typeKey]bool{headKey: true}
		for _, member := range members {
			skipAlias[member.elementKey] = true
		}
		for i := range meta.Types {
			t := &meta.Types[i]
			if t.Kind == "element" && skipAlias[typeKey{namespace: t.Namespace, local: t.Name}] {
				t.SkipElementAlias = true
			}
		}
		// Mark concrete implementations.
		for _, impl := range impls {
			for i := range meta.Types {
				t := &meta.Types[i]
				if (t.Kind == "complexType" || t.Kind == "simpleType") &&
					t.Namespace == impl.typeNS && t.Name == impl.typeLocal {
					appendPolymorphicImpl(t, PolymorphicImpl{
						InterfaceName: interfaceName,
						MarkerMethod:  marker,
						ElementNS:     impl.elementNS,
						ElementLocal:  impl.elementLocal,
					})
				}
			}
		}
	}

	// Flag container types with a polymorphic field as needing custom XML.
	for i := range meta.Types {
		t := &meta.Types[i]
		if t.Kind != "complexType" {
			continue
		}
		for _, f := range t.Fields {
			if f.Polymorphic != nil {
				t.CustomXML = &CustomXML{
					ElementNS:    t.Namespace,
					ElementLocal: t.XMLName,
				}
				break
			}
		}
	}
}

func isAbstractComplexType(ctx context, key typeKey) bool {
	entry, ok := ctx.complexTypes[key]
	return ok && entry.type_.Abstract
}

func dynamicElementForType(ctx context, headKey typeKey, concrete typeKey, members []substMember) (string, string) {
	if _, hasGlobalHead := ctx.elements[headKey]; !hasGlobalHead && len(members) == 0 {
		return headKey.namespace, headKey.local
	}
	bestNS := headKey.namespace
	bestLocal := headKey.local
	bestDistance := int(^uint(0) >> 1)
	try := func(el schema.Element, ns string, local string) {
		if el.TypeName.Local == "" {
			return
		}
		candidate := typeKey{namespace: el.TypeName.Namespace, local: el.TypeName.Local}
		if dist, ok := typeDistance(ctx, concrete, candidate); ok && dist < bestDistance {
			bestDistance = dist
			bestNS = ns
			bestLocal = local
		}
	}
	if head, ok := ctx.elements[headKey]; ok {
		try(head.element, headKey.namespace, headKey.local)
	}
	for _, member := range members {
		try(member.element, member.elementKey.namespace, member.element.Name)
	}
	return bestNS, bestLocal
}

func typeDistance(ctx context, concrete typeKey, ancestor typeKey) (int, bool) {
	cur := concrete
	for dist := 0; ; dist++ {
		if cur == ancestor {
			return dist, true
		}
		base, ok := ctx.typeBase[cur]
		if !ok {
			return 0, false
		}
		cur = base
	}
}

func appendPolymorphicImpl(t *MetaType, impl PolymorphicImpl) {
	for _, existing := range t.PolymorphicImpls {
		if existing.MarkerMethod == impl.MarkerMethod && existing.ElementNS == impl.ElementNS && existing.ElementLocal == impl.ElementLocal {
			return
		}
	}
	t.PolymorphicImpls = append(t.PolymorphicImpls, impl)
}

func sortedKeys(m map[typeKey]bool) []typeKey {
	keys := make([]typeKey, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keyLess(keys[j], keys[j-1]); j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}

func keyLess(a, b typeKey) bool {
	if a.namespace != b.namespace {
		return a.namespace < b.namespace
	}
	return a.local < b.local
}

func exportGoName(local string) string {
	if local == "" {
		return local
	}
	r := []rune(local)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func isXSDNamespace(namespace string) bool {
	return namespace == "" || namespace == "http://www.w3.org/2001/XMLSchema"
}

func registryVar(local string) string {
	if local == "" {
		local = "poly"
	}
	r := []rune(local)
	r[0] = unicode.ToUpper(r[0])
	return string(r) + "Registry"
}
