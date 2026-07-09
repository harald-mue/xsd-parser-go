package parser

import (
	"encoding/xml"
	"io"

	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func (p *parser) parseElement(start xml.StartElement) (schema.Element, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	name := attr(start, "name")
	typeRef := attr(start, "type")
	ref := attr(start, "ref")
	subGroup := attr(start, "substitutionGroup")
	element := schema.Element{
		Name:                  name,
		Ref:                   ref,
		Type:                  typeRef,
		Form:                  attr(start, "form"),
		Default:               attr(start, "default"),
		Fixed:                 attr(start, "fixed"),
		QName:                 p.componentQName(name),
		RefName:               p.parseQName(ref),
		TypeName:              p.parseQName(typeRef),
		SubstitutionGroup:     subGroup,
		SubstitutionGroupName: p.parseQName(subGroup),
		Abstract:              parseBool(attr(start, "abstract")),
		Nillable:              parseBool(attr(start, "nillable")),
		MinOccurs:             parseMinOccurs(attr(start, "minOccurs")),
		MaxOccurs:             parseMaxOccurs(attr(start, "maxOccurs")),
	}

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return element, nil
			}
			return schema.Element{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space != xsdNamespace {
				if err := p.skip(token); err != nil {
					return schema.Element{}, err
				}
				continue
			}
			switch token.Name.Local {
			case "annotation":
				annotation, err := p.parseAnnotation(token)
				if err != nil {
					return schema.Element{}, err
				}
				element.Annotation = annotation
			case "complexType":
				complexType, err := p.parseComplexType(token)
				if err != nil {
					return schema.Element{}, err
				}
				element.AnonymousComplexType = &complexType
			case "simpleType":
				simpleType, err := p.parseSimpleType(token)
				if err != nil {
					return schema.Element{}, err
				}
				element.AnonymousSimpleType = &simpleType
			default:
				if err := p.skip(token); err != nil {
					return schema.Element{}, err
				}
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return element, nil
			}
		}
	}
}

func (p *parser) parseParticleGroup(start xml.StartElement) (schema.ParticleGroup, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	group := schema.ParticleGroup{
		Kind:      start.Name.Local,
		MinOccurs: parseMinOccurs(attr(start, "minOccurs")),
		MaxOccurs: parseMaxOccurs(attr(start, "maxOccurs")),
	}

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return group, nil
			}
			return schema.ParticleGroup{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space != xsdNamespace {
				if err := p.skip(token); err != nil {
					return schema.ParticleGroup{}, err
				}
				continue
			}
			switch token.Name.Local {
			case "element":
				element, err := p.parseElement(token)
				if err != nil {
					return schema.ParticleGroup{}, err
				}
				group.Particles = append(group.Particles, schema.Particle{Kind: schema.ParticleKindElement, Element: &element})
			case "sequence", "choice", "all":
				nested, err := p.parseParticleGroup(token)
				if err != nil {
					return schema.ParticleGroup{}, err
				}
				group.Particles = append(group.Particles, schema.Particle{Kind: schema.ParticleKindGroup, Group: &nested})
			case "group":
				groupRef, err := p.parseGroupRef(token)
				if err != nil {
					return schema.ParticleGroup{}, err
				}
				group.Particles = append(group.Particles, schema.Particle{Kind: schema.ParticleKindGroup, Group: &groupRef})
			case "any":
				any := parseAny(token)
				if err := p.skip(token); err != nil {
					return schema.ParticleGroup{}, err
				}
				group.Particles = append(group.Particles, schema.Particle{Kind: schema.ParticleKindAny, Any: &any})
			default:
				if err := p.skip(token); err != nil {
					return schema.ParticleGroup{}, err
				}
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return group, nil
			}
		}
	}
}

func (p *parser) parseGroup(start xml.StartElement) (schema.Group, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	name := attr(start, "name")
	group := schema.Group{Name: name, QName: p.componentQName(name)}

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return group, nil
			}
			return schema.Group{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space != xsdNamespace {
				if err := p.skip(token); err != nil {
					return schema.Group{}, err
				}
				continue
			}
			switch token.Name.Local {
			case "sequence", "choice", "all":
				content, err := p.parseParticleGroup(token)
				if err != nil {
					return schema.Group{}, err
				}
				group.Content = content
			default:
				if err := p.skip(token); err != nil {
					return schema.Group{}, err
				}
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return group, nil
			}
		}
	}
}

func (p *parser) parseGroupRef(start xml.StartElement) (schema.ParticleGroup, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	ref := attr(start, "ref")
	group := schema.ParticleGroup{
		Kind:      "group",
		MinOccurs: parseMinOccurs(attr(start, "minOccurs")),
		MaxOccurs: parseMaxOccurs(attr(start, "maxOccurs")),
		Ref:       ref,
		RefName:   p.parseQName(ref),
	}
	if err := p.skip(start); err != nil {
		return schema.ParticleGroup{}, err
	}
	return group, nil
}

func parseAny(start xml.StartElement) schema.Any {
	return schema.Any{
		Namespace:       attr(start, "namespace"),
		ProcessContents: attr(start, "processContents"),
		MinOccurs:       parseMinOccurs(attr(start, "minOccurs")),
		MaxOccurs:       parseMaxOccurs(attr(start, "maxOccurs")),
	}
}

func parseAnyAttribute(start xml.StartElement) schema.AnyAttribute {
	return schema.AnyAttribute{
		Namespace:       attr(start, "namespace"),
		ProcessContents: attr(start, "processContents"),
	}
}
