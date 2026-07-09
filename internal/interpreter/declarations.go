package interpreter

import "github.com/harald-mue/xsd-parser-go/internal/schema"

// DeclarationIndex lists element and attribute declarations discovered in the
// parsed schema set. It is used for xs:any / xs:anyAttribute
// processContents="strict" validation. The index includes global declarations
// and local declarations with the namespace they use in instance XML.
type DeclarationIndex struct {
	Elements   []schema.QName
	Attributes []schema.QName
}

func buildDeclarationIndex(schemas schema.Schemas) DeclarationIndex {
	attributeGroups := make(map[typeKey]schema.AttributeGroup)
	for _, file := range schemas.Files {
		for _, group := range file.AttributeGroups {
			if group.Name != "" {
				attributeGroups[typeKey{namespace: file.TargetNamespace, local: group.Name}] = group
			}
		}
	}

	seenElements := make(map[typeKey]struct{})
	seenAttributes := make(map[typeKey]struct{})
	var index DeclarationIndex
	addElement := func(q schema.QName) {
		if q.Local == "" {
			return
		}
		key := typeKey{namespace: q.Namespace, local: q.Local}
		if _, ok := seenElements[key]; ok {
			return
		}
		seenElements[key] = struct{}{}
		index.Elements = append(index.Elements, q)
	}
	addAttribute := func(q schema.QName) {
		if q.Local == "" {
			return
		}
		key := typeKey{namespace: q.Namespace, local: q.Local}
		if _, ok := seenAttributes[key]; ok {
			return
		}
		seenAttributes[key] = struct{}{}
		index.Attributes = append(index.Attributes, q)
	}

	for _, file := range schemas.Files {
		for _, element := range file.Elements {
			if element.Name == "" || element.Ref != "" {
				continue
			}
			addElement(schema.QName{Namespace: file.TargetNamespace, Local: element.Name})
			if element.AnonymousComplexType != nil {
				collectComplexTypeDeclarations(file, *element.AnonymousComplexType, attributeGroups, addElement, addAttribute)
			}
		}
		for _, attribute := range file.Attributes {
			if attribute.Name == "" || attribute.Ref != "" {
				continue
			}
			addAttribute(schema.QName{Namespace: file.TargetNamespace, Local: attribute.Name})
		}
		for _, complexType := range file.ComplexTypes {
			collectComplexTypeDeclarations(file, complexType, attributeGroups, addElement, addAttribute)
		}
		for _, group := range file.AttributeGroups {
			collectAttributeGroupDeclarations(file, group, attributeGroups, addAttribute, nil)
		}
		for _, group := range file.Groups {
			collectParticleGroupDeclarations(file, group.Content, attributeGroups, addElement, addAttribute)
		}
	}
	return index
}

func collectComplexTypeDeclarations(file schema.File, complexType schema.ComplexType, attributeGroups map[typeKey]schema.AttributeGroup, addElement func(schema.QName), addAttribute func(schema.QName)) {
	for _, particle := range complexType.Content {
		collectParticleDeclarations(file, particle, attributeGroups, addElement, addAttribute)
	}
	if complexType.Extension != nil {
		for _, particle := range complexType.Extension.Content {
			collectParticleDeclarations(file, particle, attributeGroups, addElement, addAttribute)
		}
		collectAttributeDeclarations(file, complexType.Extension.Attributes, complexType.Extension.AttributeGroups, attributeGroups, addAttribute)
	}
	if complexType.Restriction != nil {
		for _, particle := range complexType.Restriction.Content {
			collectParticleDeclarations(file, particle, attributeGroups, addElement, addAttribute)
		}
		collectAttributeDeclarations(file, complexType.Restriction.Attributes, complexType.Restriction.AttributeGroups, attributeGroups, addAttribute)
	}
	if complexType.SimpleContent != nil {
		if complexType.SimpleContent.Extension != nil {
			collectAttributeDeclarations(file, complexType.SimpleContent.Extension.Attributes, complexType.SimpleContent.Extension.AttributeGroups, attributeGroups, addAttribute)
		}
		if complexType.SimpleContent.Restriction != nil {
			collectAttributeDeclarations(file, complexType.SimpleContent.Restriction.Attributes, complexType.SimpleContent.Restriction.AttributeGroups, attributeGroups, addAttribute)
		}
	}
	collectAttributeDeclarations(file, complexType.Attributes, complexType.AttributeGroups, attributeGroups, addAttribute)
}

func collectAttributeDeclarations(file schema.File, attributes []schema.Attribute, refs []schema.AttributeGroupRef, attributeGroups map[typeKey]schema.AttributeGroup, addAttribute func(schema.QName)) {
	for _, attribute := range attributes {
		collectLocalAttributeDeclaration(file, attribute, addAttribute)
	}
	for _, ref := range refs {
		collectAttributeGroupRefDeclaration(file, ref, attributeGroups, addAttribute, nil)
	}
}

func collectAttributeGroupRefDeclaration(file schema.File, ref schema.AttributeGroupRef, attributeGroups map[typeKey]schema.AttributeGroup, addAttribute func(schema.QName), visiting map[typeKey]struct{}) {
	key := typeKey{namespace: ref.RefName.Namespace, local: ref.RefName.Local}
	group, ok := attributeGroups[key]
	if !ok {
		return
	}
	collectAttributeGroupDeclarations(file, group, attributeGroups, addAttribute, visiting)
}

func collectAttributeGroupDeclarations(file schema.File, group schema.AttributeGroup, attributeGroups map[typeKey]schema.AttributeGroup, addAttribute func(schema.QName), visiting map[typeKey]struct{}) {
	key := typeKey{namespace: file.TargetNamespace, local: group.Name}
	if visiting == nil {
		visiting = make(map[typeKey]struct{})
	}
	if key.local != "" {
		if _, ok := visiting[key]; ok {
			return
		}
		visiting[key] = struct{}{}
		defer delete(visiting, key)
	}
	for _, attribute := range group.Attributes {
		collectLocalAttributeDeclaration(file, attribute, addAttribute)
	}
	for _, ref := range group.AttributeGroups {
		collectAttributeGroupRefDeclaration(file, ref, attributeGroups, addAttribute, visiting)
	}
}

func collectParticleDeclarations(file schema.File, particle schema.Particle, attributeGroups map[typeKey]schema.AttributeGroup, addElement func(schema.QName), addAttribute func(schema.QName)) {
	switch particle.Kind {
	case schema.ParticleKindElement:
		if particle.Element == nil {
			return
		}
		collectLocalElementDeclaration(file, *particle.Element, attributeGroups, addElement, addAttribute)
	case schema.ParticleKindGroup:
		if particle.Group != nil {
			collectParticleGroupDeclarations(file, *particle.Group, attributeGroups, addElement, addAttribute)
		}
	}
}

func collectParticleGroupDeclarations(file schema.File, group schema.ParticleGroup, attributeGroups map[typeKey]schema.AttributeGroup, addElement func(schema.QName), addAttribute func(schema.QName)) {
	for _, particle := range group.Particles {
		collectParticleDeclarations(file, particle, attributeGroups, addElement, addAttribute)
	}
}

func collectLocalElementDeclaration(file schema.File, element schema.Element, attributeGroups map[typeKey]schema.AttributeGroup, addElement func(schema.QName), addAttribute func(schema.QName)) {
	if element.RefName.Local != "" {
		addElement(element.RefName)
		return
	}
	if element.Name == "" {
		return
	}
	ns := ""
	if element.Form == "qualified" || (element.Form == "" && file.ElementFormDefault == "qualified") {
		ns = file.TargetNamespace
	}
	addElement(schema.QName{Namespace: ns, Local: element.Name})
	if element.AnonymousComplexType != nil {
		collectComplexTypeDeclarations(file, *element.AnonymousComplexType, attributeGroups, addElement, addAttribute)
	}
}

func collectLocalAttributeDeclaration(file schema.File, attribute schema.Attribute, addAttribute func(schema.QName)) {
	if attribute.RefName.Local != "" {
		addAttribute(attribute.RefName)
		return
	}
	if attribute.Name == "" {
		return
	}
	ns := ""
	if attribute.Form == "qualified" || (attribute.Form == "" && file.AttributeFormDefault == "qualified") {
		ns = file.TargetNamespace
	}
	addAttribute(schema.QName{Namespace: ns, Local: attribute.Name})
}
