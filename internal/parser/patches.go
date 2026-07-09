package parser

import (
	"encoding/xml"
	"io"
	"path/filepath"

	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func (p *parser) parseOverride(start xml.StartElement) (schema.Override, error) {
	patch, err := p.parseSchemaPatch(start)
	if err != nil {
		return schema.Override{}, err
	}
	return schema.Override{
		SchemaLocation:  patch.SchemaLocation,
		Elements:        patch.Elements,
		Attributes:      patch.Attributes,
		AttributeGroups: patch.AttributeGroups,
		Groups:          patch.Groups,
		ComplexTypes:    patch.ComplexTypes,
		SimpleTypes:     patch.SimpleTypes,
	}, nil
}

func (p *parser) parseRedefine(start xml.StartElement) (schema.Redefine, error) {
	patch, err := p.parseSchemaPatch(start)
	if err != nil {
		return schema.Redefine{}, err
	}
	return schema.Redefine{
		SchemaLocation:  patch.SchemaLocation,
		Elements:        patch.Elements,
		Attributes:      patch.Attributes,
		AttributeGroups: patch.AttributeGroups,
		Groups:          patch.Groups,
		ComplexTypes:    patch.ComplexTypes,
		SimpleTypes:     patch.SimpleTypes,
	}, nil
}

type parsedSchemaPatch struct {
	SchemaLocation  string
	Elements        []schema.Element
	Attributes      []schema.Attribute
	AttributeGroups []schema.AttributeGroup
	Groups          []schema.Group
	ComplexTypes    []schema.ComplexType
	SimpleTypes     []schema.SimpleType
}

func (p *parser) parseSchemaPatch(start xml.StartElement) (parsedSchemaPatch, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	patch := parsedSchemaPatch{SchemaLocation: attr(start, "schemaLocation")}
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return patch, nil
			}
			return parsedSchemaPatch{}, err
		}
		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space != xsdNamespace {
				if err := p.skip(token); err != nil {
					return parsedSchemaPatch{}, err
				}
				continue
			}
			switch token.Name.Local {
			case "annotation":
				if err := p.skip(token); err != nil {
					return parsedSchemaPatch{}, err
				}
			case "element":
				element, err := p.parseElement(token)
				if err != nil {
					return parsedSchemaPatch{}, err
				}
				patch.Elements = append(patch.Elements, element)
			case "attribute":
				attribute, err := p.parseAttribute(token)
				if err != nil {
					return parsedSchemaPatch{}, err
				}
				patch.Attributes = append(patch.Attributes, attribute)
			case "attributeGroup":
				attributeGroup, err := p.parseAttributeGroup(token)
				if err != nil {
					return parsedSchemaPatch{}, err
				}
				patch.AttributeGroups = append(patch.AttributeGroups, attributeGroup)
			case "group":
				group, err := p.parseGroup(token)
				if err != nil {
					return parsedSchemaPatch{}, err
				}
				patch.Groups = append(patch.Groups, group)
			case "complexType":
				complexType, err := p.parseComplexType(token)
				if err != nil {
					return parsedSchemaPatch{}, err
				}
				patch.ComplexTypes = append(patch.ComplexTypes, complexType)
			case "simpleType":
				simpleType, err := p.parseSimpleType(token)
				if err != nil {
					return parsedSchemaPatch{}, err
				}
				patch.SimpleTypes = append(patch.SimpleTypes, simpleType)
			default:
				if err := p.skip(token); err != nil {
					return parsedSchemaPatch{}, err
				}
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return patch, nil
			}
		}
	}
}

func applySchemaPatches(schemas *schema.Schemas, catalogs catalogResolver) {
	pathToIndex := make(map[string]int, len(schemas.Files))
	for i := range schemas.Files {
		pathToIndex[cleanAbs(schemas.Files[i].Path)] = i
	}
	for i := range schemas.Files {
		file := &schemas.Files[i]
		baseDir := filepath.Dir(file.Path)
		for _, override := range file.Overrides {
			idx, ok := pathToIndex[cleanAbs(resolveSchemaLocationWithCatalog(baseDir, override.SchemaLocation, catalogs))]
			if !ok {
				continue
			}
			removeOverriddenDeclarations(&schemas.Files[idx], override.Elements, override.Attributes, override.AttributeGroups, override.Groups, override.ComplexTypes, override.SimpleTypes)
		}
		for _, redefine := range file.Redefines {
			idx, ok := pathToIndex[cleanAbs(resolveSchemaLocationWithCatalog(baseDir, redefine.SchemaLocation, catalogs))]
			if !ok {
				continue
			}
			applyRedefine(file, &schemas.Files[idx], redefine)
		}
	}
}

func cleanAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}

func removeOverriddenDeclarations(target *schema.File, elements []schema.Element, attributes []schema.Attribute, attributeGroups []schema.AttributeGroup, groups []schema.Group, complexTypes []schema.ComplexType, simpleTypes []schema.SimpleType) {
	for _, element := range elements {
		target.Elements = removeElement(target.Elements, element.Name)
	}
	for _, attribute := range attributes {
		target.Attributes = removeAttribute(target.Attributes, attribute.Name)
	}
	for _, group := range attributeGroups {
		target.AttributeGroups = removeAttributeGroup(target.AttributeGroups, group.Name)
	}
	for _, group := range groups {
		target.Groups = removeGroup(target.Groups, group.Name)
	}
	for _, typ := range complexTypes {
		target.ComplexTypes = removeComplexType(target.ComplexTypes, typ.Name)
	}
	for _, typ := range simpleTypes {
		target.SimpleTypes = removeSimpleType(target.SimpleTypes, typ.Name)
	}
}

func applyRedefine(current *schema.File, target *schema.File, redefine schema.Redefine) {
	for _, typ := range redefine.SimpleTypes {
		old, ok := takeSimpleType(target, typ.Name)
		if !ok {
			continue
		}
		renamed := typ.Name + "RedefinedBase"
		old.Name = renamed
		old.QName.Local = renamed
		old.QName.Raw = renamed
		target.SimpleTypes = append(target.SimpleTypes, old)
		rewriteSimpleTypeSelfBase(current, typ.Name, renamed)
	}
	for _, typ := range redefine.ComplexTypes {
		old, ok := takeComplexType(target, typ.Name)
		if !ok {
			continue
		}
		renamed := typ.Name + "RedefinedBase"
		old.Name = renamed
		old.QName.Local = renamed
		old.QName.Raw = renamed
		target.ComplexTypes = append(target.ComplexTypes, old)
		rewriteComplexTypeSelfBase(current, typ.Name, renamed)
	}
	for _, group := range redefine.AttributeGroups {
		old, ok := takeAttributeGroup(target, group.Name)
		if !ok {
			continue
		}
		renamed := group.Name + "RedefinedBase"
		old.Name = renamed
		old.QName.Local = renamed
		old.QName.Raw = renamed
		target.AttributeGroups = append(target.AttributeGroups, old)
		rewriteAttributeGroupSelfRefs(current, group.Name, renamed)
	}
	for _, group := range redefine.Groups {
		old, ok := takeGroup(target, group.Name)
		if !ok {
			continue
		}
		renamed := group.Name + "RedefinedBase"
		old.Name = renamed
		old.QName.Local = renamed
		old.QName.Raw = renamed
		target.Groups = append(target.Groups, old)
		rewriteGroupSelfRefs(current, group.Name, renamed)
	}
	removeOverriddenDeclarations(target, redefine.Elements, redefine.Attributes, redefine.AttributeGroups, nil, nil, nil)
}

func rewriteSimpleTypeSelfBase(file *schema.File, oldName, newName string) {
	old := schema.QName{Namespace: file.TargetNamespace, Local: oldName}
	for i := range file.SimpleTypes {
		if file.SimpleTypes[i].Name != oldName || file.SimpleTypes[i].Restriction == nil {
			continue
		}
		if sameQName(file.SimpleTypes[i].Restriction.BaseName, old) {
			file.SimpleTypes[i].Restriction.BaseName.Local = newName
			file.SimpleTypes[i].Restriction.BaseName.Raw = newName
			file.SimpleTypes[i].Restriction.Base = newName
		}
	}
}

func rewriteComplexTypeSelfBase(file *schema.File, oldName, newName string) {
	old := schema.QName{Namespace: file.TargetNamespace, Local: oldName}
	for i := range file.ComplexTypes {
		if file.ComplexTypes[i].Name != oldName {
			continue
		}
		if file.ComplexTypes[i].Extension != nil && sameQName(file.ComplexTypes[i].Extension.BaseName, old) {
			file.ComplexTypes[i].Extension.BaseName.Local = newName
			file.ComplexTypes[i].Extension.BaseName.Raw = newName
			file.ComplexTypes[i].Extension.Base = newName
		}
		if file.ComplexTypes[i].Restriction != nil && sameQName(file.ComplexTypes[i].Restriction.BaseName, old) {
			file.ComplexTypes[i].Restriction.BaseName.Local = newName
			file.ComplexTypes[i].Restriction.BaseName.Raw = newName
			file.ComplexTypes[i].Restriction.Base = newName
		}
	}
}

func rewriteAttributeGroupSelfRefs(file *schema.File, oldName, newName string) {
	old := schema.QName{Namespace: file.TargetNamespace, Local: oldName}
	for i := range file.AttributeGroups {
		if file.AttributeGroups[i].Name != oldName {
			continue
		}
		rewriteAttributeGroupRefs(file.AttributeGroups[i].AttributeGroups, old, newName)
	}
}

func rewriteAttributeGroupRefs(groups []schema.AttributeGroupRef, old schema.QName, newName string) {
	for i := range groups {
		if sameQName(groups[i].RefName, old) {
			groups[i].RefName.Local = newName
			groups[i].RefName.Raw = newName
			groups[i].Ref = newName
		}
	}
}

func rewriteGroupSelfRefs(file *schema.File, oldName, newName string) {
	old := schema.QName{Namespace: file.TargetNamespace, Local: oldName}
	for i := range file.Groups {
		if file.Groups[i].Name != oldName {
			continue
		}
		rewriteParticleGroupRefs(&file.Groups[i].Content, old, newName)
	}
}

func rewriteParticleGroupRefs(group *schema.ParticleGroup, old schema.QName, newName string) {
	if sameQName(group.RefName, old) {
		group.RefName.Local = newName
		group.RefName.Raw = newName
		group.Ref = newName
	}
	for i := range group.Particles {
		if group.Particles[i].Group != nil {
			rewriteParticleGroupRefs(group.Particles[i].Group, old, newName)
		}
	}
}

func sameQName(a, b schema.QName) bool { return a.Namespace == b.Namespace && a.Local == b.Local }

func removeElement(items []schema.Element, name string) []schema.Element {
	out := items[:0]
	for _, item := range items {
		if item.Name != name {
			out = append(out, item)
		}
	}
	return out
}

func removeAttribute(items []schema.Attribute, name string) []schema.Attribute {
	out := items[:0]
	for _, item := range items {
		if item.Name != name {
			out = append(out, item)
		}
	}
	return out
}

func removeAttributeGroup(items []schema.AttributeGroup, name string) []schema.AttributeGroup {
	out := items[:0]
	for _, item := range items {
		if item.Name != name {
			out = append(out, item)
		}
	}
	return out
}

func removeGroup(items []schema.Group, name string) []schema.Group {
	out := items[:0]
	for _, item := range items {
		if item.Name != name {
			out = append(out, item)
		}
	}
	return out
}

func removeComplexType(items []schema.ComplexType, name string) []schema.ComplexType {
	out := items[:0]
	for _, item := range items {
		if item.Name != name {
			out = append(out, item)
		}
	}
	return out
}

func removeSimpleType(items []schema.SimpleType, name string) []schema.SimpleType {
	out := items[:0]
	for _, item := range items {
		if item.Name != name {
			out = append(out, item)
		}
	}
	return out
}

func takeSimpleType(file *schema.File, name string) (schema.SimpleType, bool) {
	for i, item := range file.SimpleTypes {
		if item.Name == name {
			file.SimpleTypes = append(file.SimpleTypes[:i], file.SimpleTypes[i+1:]...)
			return item, true
		}
	}
	return schema.SimpleType{}, false
}

func takeComplexType(file *schema.File, name string) (schema.ComplexType, bool) {
	for i, item := range file.ComplexTypes {
		if item.Name == name {
			file.ComplexTypes = append(file.ComplexTypes[:i], file.ComplexTypes[i+1:]...)
			return item, true
		}
	}
	return schema.ComplexType{}, false
}

func takeAttributeGroup(file *schema.File, name string) (schema.AttributeGroup, bool) {
	for i, item := range file.AttributeGroups {
		if item.Name == name {
			file.AttributeGroups = append(file.AttributeGroups[:i], file.AttributeGroups[i+1:]...)
			return item, true
		}
	}
	return schema.AttributeGroup{}, false
}

func takeGroup(file *schema.File, name string) (schema.Group, bool) {
	for i, item := range file.Groups {
		if item.Name == name {
			file.Groups = append(file.Groups[:i], file.Groups[i+1:]...)
			return item, true
		}
	}
	return schema.Group{}, false
}
