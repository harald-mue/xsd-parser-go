package interpreter

import "github.com/harald-mue/xsd-parser-go/internal/schema"

// MetaTypes is the language-neutral semantic model produced from parsed schemas.
//
// This layer keeps schema semantics that are needed before Go-specific lowering:
// resolved XML names, field cardinality, attributes, and simple aliases.
type MetaTypes struct {
	Types        []MetaType
	Interfaces   []MetaInterface
	Declarations DeclarationIndex
}

// MetaType describes one interpreted schema type or top-level element.
type MetaType struct {
	Kind          string
	Name          string
	Documentation []string
	XMLName       string
	TypeRef       string
	TypeName      schema.QName
	Namespace     string
	Nillable      bool

	Fields []MetaField
	Alias  *MetaAlias
	List   *MetaList
	Union  *MetaUnion
	Mixed  bool

	IsSubstitutionHead bool
	SkipElementAlias   bool
	DuplicateOf        schema.QName
	PolymorphicImpls   []PolymorphicImpl
	CustomXML          *CustomXML
}

// MetaInterface describes a generated polymorphic interface (substitution
// group head or dynamic type root).
type MetaInterface struct {
	Name         string
	MarkerMethod string
	RegistryVar  string
	HeadNS       string
	HeadLocal    string
	Members      []MetaInterfaceMember
}

// MetaInterfaceMember is one candidate element/type for an interface registry.
type MetaInterfaceMember struct {
	DispatchNS        string
	DispatchLocal     string
	ElementNS         string
	ElementLocal      string
	ConcreteTypeNS    string
	ConcreteTypeLocal string
	XSITypeNS         string
	XSITypeLocal      string
	UseXSIType        bool
	Head              bool
}

// PolymorphicImpl marks a complex type as a concrete implementation of a
// polymorphic interface, so the generator emits a marker method and an XMLName
// field bound to the member element name.
type PolymorphicImpl struct {
	InterfaceName string
	MarkerMethod  string
	ElementNS     string
	ElementLocal  string
}

// CustomXML describes a struct that needs generated MarshalXML/UnmarshalXML
// because it contains a polymorphic field that encoding/xml cannot dispatch.
type CustomXML struct {
	ElementNS    string
	ElementLocal string
}

// MetaField describes one property of a semantic complex type.
type MetaField struct {
	Name          string
	Documentation []string
	XMLName       string
	Namespace     string
	TypeRef       string
	TypeName      schema.QName

	Attribute          bool
	AnyAttribute       bool
	AnyElement         bool
	AnyNamespace       string
	AnyProcessContents string
	AnyTargetNamespace string
	Chardata           bool
	Choice             bool
	Nillable           bool
	Required           bool
	MinOccurs          int
	MaxOccurs          int
	Default            string
	Fixed              string

	Polymorphic *FieldPolymorphic
}

// FieldPolymorphic marks a field as a polymorphic reference to a substitution
// group head, rendered as an interface (or slice of interface).
type FieldPolymorphic struct {
	InterfaceName string
	RegistryVar   string
	Repeated      bool
}

// MetaAlias describes the base type of a semantic simple type.
type MetaAlias struct {
	BaseRef  string
	BaseName schema.QName
	Facets   []schema.Facet
}

// MetaList describes an xs:list simple type.
type MetaList struct {
	ItemRef  string
	ItemName schema.QName
}

// MetaUnion describes an xs:union simple type.
type MetaUnion struct {
	MemberRefs  string
	MemberNames []schema.QName
}

type complexEntry struct {
	file  schema.File
	type_ schema.ComplexType
}

type elementEntry struct {
	file    schema.File
	element schema.Element
}

type attributeEntry struct {
	file      schema.File
	attribute schema.Attribute
}

type typeKey struct {
	namespace string
	local     string
}

type context struct {
	complexTypes     map[typeKey]complexEntry
	elements         map[typeKey]elementEntry
	elementsByType   map[typeKey][]typeKey
	attributes       map[typeKey]attributeEntry
	attributeGroups  map[typeKey]schema.AttributeGroup
	groups           map[typeKey]schema.Group
	substs           map[typeKey][]substMember
	derivedTypes     map[typeKey][]typeKey
	typeBase         map[typeKey]typeKey
	usedHeads        map[typeKey]bool
	usedDynamicHeads map[typeKey]typeKey
}

type substMember struct {
	elementKey typeKey
	element    schema.Element
}

// Interpret converts parsed schemas into a semantic model.
func Interpret(schemas schema.Schemas) (MetaTypes, error) {
	ctx := buildContext(schemas)

	var meta MetaTypes
	for _, file := range schemas.Files {
		for _, element := range file.Elements {
			typeName := elementTypeName(file, element)
			meta.Types = append(meta.Types, MetaType{
				Kind:          "element",
				Name:          element.Name,
				Documentation: append([]string(nil), element.Annotation.Documentation...),
				XMLName:       element.Name,
				TypeRef:       element.Type,
				TypeName:      typeName,
				Namespace:     file.TargetNamespace,
				Nillable:      element.Nillable,
			})
			if element.AnonymousComplexType != nil {
				anonymous := *element.AnonymousComplexType
				anonymous.Name = anonymousTypeName(element.Name)
				anonymous.QName = schema.QName{Raw: anonymous.Name, Local: anonymous.Name, Namespace: file.TargetNamespace}
				meta.Types = append(meta.Types, MetaType{
					Kind:          "complexType",
					Name:          anonymous.Name,
					Documentation: append([]string(nil), anonymous.Annotation.Documentation...),
					XMLName:       element.Name,
					Namespace:     file.TargetNamespace,
					Mixed:         anonymous.Mixed,
					Fields:        interpretComplexFields(ctx, complexEntry{file: file, type_: anonymous}, nil),
				})
				if enumMeta := simpleContentRestrictionEnumMeta(ctx, file, anonymous.Name, anonymous); enumMeta != nil {
					meta.Types = append(meta.Types, *enumMeta)
				}
			}
		}
		for _, complexType := range file.ComplexTypes {
			meta.Types = append(meta.Types, MetaType{
				Kind:          "complexType",
				Name:          complexType.Name,
				Documentation: append([]string(nil), complexType.Annotation.Documentation...),
				XMLName:       complexType.Name,
				Namespace:     file.TargetNamespace,
				Mixed:         complexType.Mixed,
				Fields:        interpretComplexFields(ctx, complexEntry{file: file, type_: complexType}, nil),
			})
			if enumMeta := simpleContentRestrictionEnumMeta(ctx, file, complexType.Name, complexType); enumMeta != nil {
				meta.Types = append(meta.Types, *enumMeta)
			}
		}
		for _, simpleType := range file.SimpleTypes {
			metaType := MetaType{
				Kind:          "simpleType",
				Name:          simpleType.Name,
				Documentation: append([]string(nil), simpleType.Annotation.Documentation...),
				XMLName:       simpleType.Name,
				Namespace:     file.TargetNamespace,
			}
			if simpleType.Restriction != nil {
				metaType.Alias = &MetaAlias{
					BaseRef:  simpleType.Restriction.Base,
					BaseName: simpleType.Restriction.BaseName,
					Facets:   append([]schema.Facet(nil), simpleType.Restriction.Facets...),
				}
			}
			if simpleType.List != nil {
				metaType.List = &MetaList{ItemRef: simpleType.List.ItemType, ItemName: simpleType.List.ItemTypeName}
				if simpleType.List.SimpleType != nil && simpleType.List.SimpleType.Restriction != nil {
					metaType.List.ItemName = simpleType.List.SimpleType.Restriction.BaseName
				}
			}
			if simpleType.Union != nil {
				metaType.Union = &MetaUnion{MemberRefs: simpleType.Union.MemberTypes, MemberNames: append([]schema.QName(nil), simpleType.Union.MemberTypeNames...)}
			}
			meta.Types = append(meta.Types, metaType)
		}
	}

	enrichPolymorphism(&meta, ctx)
	meta.Declarations = buildDeclarationIndex(schemas)
	return meta, nil
}

func buildContext(schemas schema.Schemas) context {
	ctx := context{
		complexTypes:     make(map[typeKey]complexEntry),
		elements:         make(map[typeKey]elementEntry),
		elementsByType:   make(map[typeKey][]typeKey),
		attributes:       make(map[typeKey]attributeEntry),
		attributeGroups:  make(map[typeKey]schema.AttributeGroup),
		groups:           make(map[typeKey]schema.Group),
		substs:           buildSubstitutionGroups(schemas),
		derivedTypes:     buildDerivedTypes(schemas),
		typeBase:         buildTypeBase(schemas),
		usedHeads:        make(map[typeKey]bool),
		usedDynamicHeads: make(map[typeKey]typeKey),
	}
	for _, file := range schemas.Files {
		for _, complexType := range file.ComplexTypes {
			if complexType.Name == "" {
				continue
			}
			ctx.complexTypes[typeKey{namespace: file.TargetNamespace, local: complexType.Name}] = complexEntry{file: file, type_: complexType}
		}
		for _, element := range file.Elements {
			if element.Name == "" {
				continue
			}
			elKey := typeKey{namespace: file.TargetNamespace, local: element.Name}
			ctx.elements[elKey] = elementEntry{file: file, element: element}
			if typeName := elementTypeName(file, element); typeName.Local != "" {
				tk := typeKey{namespace: typeName.Namespace, local: typeName.Local}
				ctx.elementsByType[tk] = append(ctx.elementsByType[tk], elKey)
			}
			if element.AnonymousComplexType != nil {
				anonymous := *element.AnonymousComplexType
				anonymous.Name = anonymousTypeName(element.Name)
				anonymous.QName = schema.QName{Raw: anonymous.Name, Local: anonymous.Name, Namespace: file.TargetNamespace}
				ctx.complexTypes[typeKey{namespace: file.TargetNamespace, local: anonymous.Name}] = complexEntry{file: file, type_: anonymous}
			}
		}
		for _, attribute := range file.Attributes {
			if attribute.Name == "" {
				continue
			}
			ctx.attributes[typeKey{namespace: file.TargetNamespace, local: attribute.Name}] = attributeEntry{file: file, attribute: attribute}
		}
		for _, group := range file.AttributeGroups {
			if group.Name == "" {
				continue
			}
			ctx.attributeGroups[typeKey{namespace: file.TargetNamespace, local: group.Name}] = group
		}
		for _, group := range file.Groups {
			if group.Name == "" {
				continue
			}
			ctx.groups[typeKey{namespace: file.TargetNamespace, local: group.Name}] = group
		}
	}
	return ctx
}

func simpleContentRestrictionEnumMeta(ctx context, file schema.File, ownerName string, complexType schema.ComplexType) *MetaType {
	if ownerName == "" || complexType.SimpleContent == nil || complexType.SimpleContent.Restriction == nil {
		return nil
	}
	restriction := complexType.SimpleContent.Restriction
	if !hasEnumerationFacets(restriction.Facets) {
		return nil
	}

	baseRef := restriction.Base
	baseName := restriction.BaseName
	baseKey := typeKey{namespace: restriction.BaseName.Namespace, local: restriction.BaseName.Local}
	if base, ok := ctx.complexTypes[baseKey]; ok {
		_, content := splitChardataField(interpretComplexFields(ctx, base, nil))
		if content != nil && content.TypeName.Local != "" {
			baseRef = content.TypeRef
			baseName = content.TypeName
		}
	}

	return &MetaType{
		Kind:      "simpleType",
		Name:      contentTypeName(ownerName),
		XMLName:   contentTypeName(ownerName),
		Namespace: file.TargetNamespace,
		Alias: &MetaAlias{
			BaseRef:  baseRef,
			BaseName: baseName,
			Facets:   append([]schema.Facet(nil), restriction.Facets...),
		},
	}
}

func hasEnumerationFacets(facets []schema.Facet) bool {
	for _, facet := range facets {
		if facet.Name == "enumeration" {
			return true
		}
	}
	return false
}

func contentTypeName(ownerName string) string {
	if ownerName == "" {
		return "ContentType"
	}
	return ownerName + "ContentType"
}

func interpretComplexFields(ctx context, entry complexEntry, visiting map[typeKey]struct{}) []MetaField {
	key := typeKey{namespace: entry.file.TargetNamespace, local: entry.type_.Name}
	if visiting == nil {
		visiting = make(map[typeKey]struct{})
	}
	if key.local != "" {
		if _, ok := visiting[key]; ok {
			return nil
		}
		visiting[key] = struct{}{}
		defer delete(visiting, key)
	}

	if simpleContent := entry.type_.SimpleContent; simpleContent != nil {
		return interpretSimpleContent(ctx, entry.file, entry.type_.Name, simpleContent, visiting)
	}

	var fields []MetaField
	if extension := entry.type_.Extension; extension != nil {
		baseKey := typeKey{namespace: extension.BaseName.Namespace, local: extension.BaseName.Local}
		if base, ok := ctx.complexTypes[baseKey]; ok {
			fields = append(fields, interpretComplexFields(ctx, base, visiting)...)
		}
		fields = append(fields, interpretParticles(ctx, entry.file, extension.Content, false)...)
		fields = append(fields, interpretAttributes(ctx, entry.file, extension.Attributes)...)
		fields = append(fields, interpretAttributeGroupRefs(ctx, entry.file, extension.AttributeGroups)...)
		fields = append(fields, interpretAnyAttributes(entry.file, extension.AnyAttributes)...)
		return fields
	}
	if restriction := entry.type_.Restriction; restriction != nil {
		fields = append(fields, interpretParticles(ctx, entry.file, restriction.Content, false)...)
		fields = append(fields, interpretAttributes(ctx, entry.file, restriction.Attributes)...)
		fields = append(fields, interpretAttributeGroupRefs(ctx, entry.file, restriction.AttributeGroups)...)
		fields = append(fields, interpretAnyAttributes(entry.file, restriction.AnyAttributes)...)
		return fields
	}

	fields = append(fields, interpretParticles(ctx, entry.file, entry.type_.Content, false)...)
	fields = append(fields, interpretAttributes(ctx, entry.file, entry.type_.Attributes)...)
	fields = append(fields, interpretAttributeGroupRefs(ctx, entry.file, entry.type_.AttributeGroups)...)
	fields = append(fields, interpretAnyAttributes(entry.file, entry.type_.AnyAttributes)...)
	return fields
}

func interpretSimpleContent(ctx context, file schema.File, ownerName string, simpleContent *schema.SimpleContent, visiting map[typeKey]struct{}) []MetaField {
	if simpleContent == nil {
		return nil
	}
	if simpleContent.Extension != nil {
		return interpretSimpleContentExtension(ctx, file, simpleContent.Extension, visiting)
	}
	if simpleContent.Restriction != nil {
		return interpretSimpleContentRestriction(ctx, file, ownerName, simpleContent.Restriction, visiting)
	}
	return nil
}

func interpretSimpleContentExtension(ctx context, file schema.File, extension *schema.SimpleContentExtension, visiting map[typeKey]struct{}) []MetaField {
	baseKey := typeKey{namespace: extension.BaseName.Namespace, local: extension.BaseName.Local}
	if base, ok := ctx.complexTypes[baseKey]; ok {
		baseFields := interpretComplexFields(ctx, base, visiting)
		baseAttrs, content := splitChardataField(baseFields)
		fields := append([]MetaField{}, baseAttrs...)
		fields = append(fields, interpretAttributes(ctx, file, extension.Attributes)...)
		fields = append(fields, interpretAttributeGroupRefs(ctx, file, extension.AttributeGroups)...)
		fields = append(fields, interpretAnyAttributes(file, extension.AnyAttributes)...)
		if content != nil {
			fields = append(fields, *content)
		}
		return fields
	}

	fields := interpretAttributes(ctx, file, extension.Attributes)
	fields = append(fields, interpretAttributeGroupRefs(ctx, file, extension.AttributeGroups)...)
	fields = append(fields, interpretAnyAttributes(file, extension.AnyAttributes)...)
	fields = append(fields, simpleContentField(extension.Base, extension.BaseName))
	return fields
}

func interpretSimpleContentRestriction(ctx context, file schema.File, ownerName string, restriction *schema.SimpleContentRestriction, visiting map[typeKey]struct{}) []MetaField {
	baseKey := typeKey{namespace: restriction.BaseName.Namespace, local: restriction.BaseName.Local}
	if base, ok := ctx.complexTypes[baseKey]; ok {
		baseFields := interpretComplexFields(ctx, base, visiting)
		baseAttrs, content := splitChardataField(baseFields)
		fields := append([]MetaField{}, baseAttrs...)
		fields = append(fields, interpretAttributes(ctx, file, restriction.Attributes)...)
		fields = append(fields, interpretAttributeGroupRefs(ctx, file, restriction.AttributeGroups)...)
		fields = append(fields, interpretAnyAttributes(file, restriction.AnyAttributes)...)
		if content != nil {
			if hasEnumerationFacets(restriction.Facets) && ownerName != "" {
				content.TypeName = schema.QName{Raw: contentTypeName(ownerName), Local: contentTypeName(ownerName), Namespace: file.TargetNamespace}
			}
			fields = append(fields, *content)
		} else {
			fields = append(fields, restrictedSimpleContentField(file, ownerName, restriction))
		}
		return fields
	}

	fields := interpretAttributes(ctx, file, restriction.Attributes)
	fields = append(fields, interpretAttributeGroupRefs(ctx, file, restriction.AttributeGroups)...)
	fields = append(fields, interpretAnyAttributes(file, restriction.AnyAttributes)...)
	fields = append(fields, restrictedSimpleContentField(file, ownerName, restriction))
	return fields
}

func restrictedSimpleContentField(file schema.File, ownerName string, restriction *schema.SimpleContentRestriction) MetaField {
	field := simpleContentField(restriction.Base, restriction.BaseName)
	if hasEnumerationFacets(restriction.Facets) && ownerName != "" {
		name := contentTypeName(ownerName)
		field.TypeName = schema.QName{Raw: name, Local: name, Namespace: file.TargetNamespace}
	}
	return field
}

func simpleContentField(typeRef string, typeName schema.QName) MetaField {
	return MetaField{
		Name:      "Content",
		XMLName:   "",
		TypeRef:   typeRef,
		TypeName:  typeName,
		Chardata:  true,
		MinOccurs: schema.OccursDefault,
		MaxOccurs: schema.OccursDefault,
	}
}

func splitChardataField(fields []MetaField) ([]MetaField, *MetaField) {
	attrs := make([]MetaField, 0, len(fields))
	var content *MetaField
	for _, field := range fields {
		if field.Chardata && content == nil {
			copy := field
			content = &copy
			continue
		}
		attrs = append(attrs, field)
	}
	return attrs, content
}

func interpretParticles(ctx context, file schema.File, particles []schema.Particle, inChoice bool) []MetaField {
	var fields []MetaField
	for _, particle := range particles {
		fields = append(fields, interpretParticle(ctx, file, particle, inChoice)...)
	}
	return fields
}

func interpretAttributes(ctx context, file schema.File, attributes []schema.Attribute) []MetaField {
	var fields []MetaField
	for _, attribute := range attributes {
		fields = append(fields, interpretAttribute(ctx, file, attribute))
	}
	return fields
}

func interpretAttributeGroupRefs(ctx context, file schema.File, refs []schema.AttributeGroupRef) []MetaField {
	var fields []MetaField
	visiting := make(map[typeKey]struct{})
	for _, ref := range refs {
		fields = append(fields, interpretAttributeGroupRef(ctx, file, ref, visiting)...)
	}
	return fields
}

func interpretAttributeGroupRef(ctx context, file schema.File, ref schema.AttributeGroupRef, visiting map[typeKey]struct{}) []MetaField {
	key := typeKey{namespace: ref.RefName.Namespace, local: ref.RefName.Local}
	if key.local == "" {
		return nil
	}
	if _, ok := visiting[key]; ok {
		return nil
	}
	group, ok := ctx.attributeGroups[key]
	if !ok {
		return nil
	}
	visiting[key] = struct{}{}
	defer delete(visiting, key)
	fields := interpretAttributes(ctx, file, group.Attributes)
	for _, nested := range group.AttributeGroups {
		fields = append(fields, interpretAttributeGroupRef(ctx, file, nested, visiting)...)
	}
	fields = append(fields, interpretAnyAttributes(file, group.AnyAttributes)...)
	return fields
}

func interpretAnyAttributes(file schema.File, anyAttributes []schema.AnyAttribute) []MetaField {
	fields := make([]MetaField, 0, len(anyAttributes))
	for _, anyAttribute := range anyAttributes {
		fields = append(fields, MetaField{
			Name:               "AnyAttribute",
			AnyAttribute:       true,
			AnyNamespace:       anyAttribute.Namespace,
			AnyProcessContents: anyAttribute.ProcessContents,
			AnyTargetNamespace: file.TargetNamespace,
			Attribute:          true,
			MinOccurs:          0,
			MaxOccurs:          schema.OccursUnbounded,
		})
	}
	return fields
}

func interpretParticle(ctx context, file schema.File, particle schema.Particle, inChoice bool) []MetaField {
	switch particle.Kind {
	case schema.ParticleKindElement:
		if particle.Element == nil {
			return nil
		}
		element := *particle.Element
		if element.RefName.Local != "" {
			refKey := typeKey{namespace: element.RefName.Namespace, local: element.RefName.Local}
			if ref, ok := ctx.elements[refKey]; ok {
				element.Name = ref.element.Name
				element.Type = ref.element.Type
				element.TypeName = elementTypeName(ref.file, ref.element)
				if element.Default == "" {
					element.Default = ref.element.Default
				}
				if element.Fixed == "" {
					element.Fixed = ref.element.Fixed
				}
				if element.MinOccurs == 0 && ref.element.MinOccurs != 0 {
					element.MinOccurs = ref.element.MinOccurs
				}
				if element.MaxOccurs == schema.OccursDefault && ref.element.MaxOccurs != schema.OccursDefault {
					element.MaxOccurs = ref.element.MaxOccurs
				}
				if poly := fieldPolymorphic(ctx, refKey, element); poly != nil {
					return []MetaField{{
						Name:          ref.element.Name,
						Documentation: append([]string(nil), ref.element.Annotation.Documentation...),
						XMLName:       ref.element.Name,
						Namespace:     elementNamespace(file, element),
						Choice:        inChoice,
						MinOccurs:     element.MinOccurs,
						MaxOccurs:     element.MaxOccurs,
						Polymorphic:   poly,
					}}
				}
			}
		}
		name := element.Name
		if name == "" {
			name = element.RefName.Local
		}
		xmlName := name
		namespace := elementNamespace(file, element)
		if wireNS, wireLocal, ok := canonicalGlobalElementWire(ctx, file, element, xmlName, namespace); ok {
			xmlName = wireLocal
			namespace = wireNS
		}
		if poly := localFieldPolymorphic(ctx, typeKey{namespace: namespace, local: xmlName}, element); poly != nil {
			return []MetaField{{
				Name:          name,
				Documentation: append([]string(nil), element.Annotation.Documentation...),
				XMLName:       xmlName,
				Namespace:     namespace,
				Choice:        inChoice,
				MinOccurs:     element.MinOccurs,
				MaxOccurs:     element.MaxOccurs,
				Polymorphic:   poly,
			}}
		}
		return []MetaField{{
			Name:          name,
			Documentation: append([]string(nil), element.Annotation.Documentation...),
			XMLName:       xmlName,
			Namespace:     namespace,
			TypeRef:       element.Type,
			TypeName:      element.TypeName,
			Choice:        inChoice,
			Nillable:      element.Nillable,
			MinOccurs:     element.MinOccurs,
			MaxOccurs:     element.MaxOccurs,
			Default:       element.Default,
			Fixed:         element.Fixed,
		}}
	case schema.ParticleKindAny:
		if particle.Any == nil {
			return nil
		}
		return []MetaField{{
			Name:               "Any",
			AnyElement:         true,
			AnyNamespace:       particle.Any.Namespace,
			AnyProcessContents: particle.Any.ProcessContents,
			AnyTargetNamespace: file.TargetNamespace,
			Choice:             inChoice,
			MinOccurs:          particle.Any.MinOccurs,
			MaxOccurs:          particle.Any.MaxOccurs,
		}}
	case schema.ParticleKindGroup:
		if particle.Group == nil {
			return nil
		}
		group := resolveGroupRef(ctx, *particle.Group)
		nestedInChoice := inChoice || group.Kind == schema.GroupChoice
		var fields []MetaField
		for _, nested := range group.Particles {
			nestedFields := interpretParticle(ctx, file, nested, nestedInChoice)
			if group.Kind == schema.GroupChoice {
				for i := range nestedFields {
					nestedFields[i].MinOccurs = group.MinOccurs
					nestedFields[i].MaxOccurs = group.MaxOccurs
				}
			}
			fields = append(fields, nestedFields...)
		}
		return fields
	default:
		return nil
	}
}

func resolveGroupRef(ctx context, group schema.ParticleGroup) schema.ParticleGroup {
	if group.Ref == "" {
		return group
	}
	refKey := typeKey{namespace: group.RefName.Namespace, local: group.RefName.Local}
	if ref, ok := ctx.groups[refKey]; ok && ref.Content.Kind != "" {
		resolved := ref.Content
		if group.MinOccurs != schema.OccursDefault {
			resolved.MinOccurs = group.MinOccurs
		}
		if group.MaxOccurs != schema.OccursDefault {
			resolved.MaxOccurs = group.MaxOccurs
		}
		return resolved
	}
	return group
}

func interpretAttribute(ctx context, file schema.File, attribute schema.Attribute) MetaField {
	if attribute.RefName.Local != "" {
		if ref, ok := ctx.attributes[typeKey{namespace: attribute.RefName.Namespace, local: attribute.RefName.Local}]; ok {
			attribute.Name = ref.attribute.Name
			attribute.Type = ref.attribute.Type
			attribute.TypeName = attributeTypeName(ref.attribute)
			if attribute.Default == "" {
				attribute.Default = ref.attribute.Default
			}
			if attribute.Fixed == "" {
				attribute.Fixed = ref.attribute.Fixed
			}
			if attribute.Use == "" {
				attribute.Use = ref.attribute.Use
			}
		}
	}
	name := attribute.Name
	if name == "" {
		name = attribute.RefName.Local
	}
	return MetaField{
		Name:          name,
		Documentation: append([]string(nil), attribute.Annotation.Documentation...),
		XMLName:       name,
		Namespace:     attributeNamespace(file, attribute),
		TypeRef:       attribute.Type,
		TypeName:      attributeTypeName(attribute),
		Attribute:     true,
		Required:      attribute.Use == "required",
		MinOccurs:     attributeMinOccurs(attribute),
		MaxOccurs:     schema.OccursDefault,
		Default:       attribute.Default,
		Fixed:         attribute.Fixed,
	}
}

func elementTypeName(file schema.File, element schema.Element) schema.QName {
	if element.TypeName.Local != "" {
		return element.TypeName
	}
	if element.AnonymousComplexType != nil {
		name := anonymousTypeName(element.Name)
		return schema.QName{Raw: name, Local: name, Namespace: file.TargetNamespace}
	}
	if element.AnonymousSimpleType != nil && element.AnonymousSimpleType.Restriction != nil {
		return element.AnonymousSimpleType.Restriction.BaseName
	}
	return schema.QName{}
}

func attributeTypeName(attribute schema.Attribute) schema.QName {
	if attribute.TypeName.Local != "" {
		return attribute.TypeName
	}
	if attribute.AnonymousSimpleType != nil && attribute.AnonymousSimpleType.Restriction != nil {
		return attribute.AnonymousSimpleType.Restriction.BaseName
	}
	return schema.QName{}
}

func anonymousTypeName(elementName string) string {
	if elementName == "" {
		return "AnonymousType"
	}
	return elementName + "Type"
}

func elementNamespace(file schema.File, element schema.Element) string {
	if element.RefName.Namespace != "" {
		return element.RefName.Namespace
	}
	if element.Form == "qualified" || (element.Form == "" && file.ElementFormDefault == "qualified") {
		return file.TargetNamespace
	}
	return ""
}

// canonicalGlobalElementWire resolves a JAXB-style on-the-wire element QName
// for a local particle typed with a QName from another namespace.
//
// This is deliberately conservative. Standard XSD semantics say the local
// element name is the wire name, even when its type is imported. Some legacy
// schemas instead use a local generic slot name (for example "exception") but
// send the canonical global element for the imported type (for example
// "webRTCException" of type "WebRTCException"). Apply the rewrite only when
// the type has exactly one global element, that element is the lower-camel type
// name, and the local slot name is the lower-camel last word of the type name.
func canonicalGlobalElementWire(ctx context, file schema.File, element schema.Element, localName, localNamespace string) (namespace, xmlName string, ok bool) {
	if element.RefName.Local != "" {
		return "", "", false
	}
	typeName := elementTypeName(file, element)
	if typeName.Local == "" || typeName.Namespace == "" || typeName.Namespace == file.TargetNamespace {
		return "", "", false
	}
	candidates := ctx.elementsByType[typeKey{namespace: typeName.Namespace, local: typeName.Local}]
	if len(candidates) != 1 {
		return "", "", false
	}
	globalKey := candidates[0]
	if globalKey.local == localName || (globalKey.namespace == localNamespace && globalKey.local == localName) {
		return "", "", false
	}
	if globalKey.local != lowerFirstIdentifier(typeName.Local) {
		return "", "", false
	}
	if localName != lowerFirstIdentifier(lastCamelWord(typeName.Local)) {
		return "", "", false
	}
	return globalKey.namespace, globalKey.local, true
}

func lowerFirstIdentifier(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	if b[0] >= 'A' && b[0] <= 'Z' {
		b[0] += 'a' - 'A'
	}
	return string(b)
}

func lastCamelWord(s string) string {
	if s == "" {
		return s
	}
	start := 0
	for i := len(s) - 2; i > 0; i-- {
		if isASCIIUpper(s[i]) && isASCIILower(s[i+1]) {
			start = i
			break
		}
	}
	return s[start:]
}

func isASCIIUpper(b byte) bool { return b >= 'A' && b <= 'Z' }

func isASCIILower(b byte) bool { return b >= 'a' && b <= 'z' }

func attributeNamespace(file schema.File, attribute schema.Attribute) string {
	if attribute.RefName.Namespace != "" {
		return attribute.RefName.Namespace
	}
	if attribute.Form == "qualified" || (attribute.Form == "" && file.AttributeFormDefault == "qualified") {
		return file.TargetNamespace
	}
	return ""
}

func attributeMinOccurs(attribute schema.Attribute) int {
	if attribute.Use == "required" {
		return schema.OccursDefault
	}
	return 0
}
