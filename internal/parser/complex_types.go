package parser

import (
	"encoding/xml"
	"io"

	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func (p *parser) parseComplexType(start xml.StartElement) (schema.ComplexType, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	name := attr(start, "name")
	complexType := schema.ComplexType{
		Name:     name,
		QName:    p.componentQName(name),
		Abstract: parseBool(attr(start, "abstract")),
		Mixed:    parseBool(attr(start, "mixed")),
	}

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return complexType, nil
			}
			return schema.ComplexType{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space != xsdNamespace {
				if err := p.skip(token); err != nil {
					return schema.ComplexType{}, err
				}
				continue
			}
			switch token.Name.Local {
			case "annotation":
				annotation, err := p.parseAnnotation(token)
				if err != nil {
					return schema.ComplexType{}, err
				}
				complexType.Annotation = annotation
			case "complexContent":
				extension, restriction, err := p.parseComplexContent(token)
				if err != nil {
					return schema.ComplexType{}, err
				}
				complexType.Extension = extension
				complexType.Restriction = restriction
			case "simpleContent":
				simpleContent, err := p.parseSimpleContent(token)
				if err != nil {
					return schema.ComplexType{}, err
				}
				complexType.SimpleContent = simpleContent
			case "sequence", "choice", "all":
				group, err := p.parseParticleGroup(token)
				if err != nil {
					return schema.ComplexType{}, err
				}
				complexType.Content = append(complexType.Content, schema.Particle{Kind: schema.ParticleKindGroup, Group: &group})
			case "group":
				groupRef, err := p.parseGroupRef(token)
				if err != nil {
					return schema.ComplexType{}, err
				}
				complexType.Content = append(complexType.Content, schema.Particle{Kind: schema.ParticleKindGroup, Group: &groupRef})
			case "attribute":
				attribute, err := p.parseAttribute(token)
				if err != nil {
					return schema.ComplexType{}, err
				}
				complexType.Attributes = append(complexType.Attributes, attribute)
			case "attributeGroup":
				group, err := p.parseAttributeGroupRef(token)
				if err != nil {
					return schema.ComplexType{}, err
				}
				complexType.AttributeGroups = append(complexType.AttributeGroups, group)
			case "anyAttribute":
				complexType.AnyAttributes = append(complexType.AnyAttributes, parseAnyAttribute(token))
				if err := p.skip(token); err != nil {
					return schema.ComplexType{}, err
				}
			default:
				if err := p.skip(token); err != nil {
					return schema.ComplexType{}, err
				}
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return complexType, nil
			}
		}
	}
}

func (p *parser) parseComplexContent(start xml.StartElement) (*schema.ComplexExtension, *schema.ComplexRestriction, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	var extension *schema.ComplexExtension
	var restriction *schema.ComplexRestriction
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return extension, restriction, nil
			}
			return nil, nil, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space == xsdNamespace && token.Name.Local == "extension" {
				parsed, err := p.parseComplexExtension(token)
				if err != nil {
					return nil, nil, err
				}
				extension = &parsed
				continue
			}
			if token.Name.Space == xsdNamespace && token.Name.Local == "restriction" {
				parsed, err := p.parseComplexRestriction(token)
				if err != nil {
					return nil, nil, err
				}
				restriction = &parsed
				continue
			}
			if err := p.skip(token); err != nil {
				return nil, nil, err
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return extension, restriction, nil
			}
		}
	}
}

func (p *parser) parseComplexRestriction(start xml.StartElement) (schema.ComplexRestriction, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	base := attr(start, "base")
	restriction := schema.ComplexRestriction{Base: base, BaseName: p.parseQName(base)}
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return restriction, nil
			}
			return schema.ComplexRestriction{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space != xsdNamespace {
				if err := p.skip(token); err != nil {
					return schema.ComplexRestriction{}, err
				}
				continue
			}
			switch token.Name.Local {
			case "sequence", "choice", "all":
				group, err := p.parseParticleGroup(token)
				if err != nil {
					return schema.ComplexRestriction{}, err
				}
				restriction.Content = append(restriction.Content, schema.Particle{Kind: schema.ParticleKindGroup, Group: &group})
			case "group":
				groupRef, err := p.parseGroupRef(token)
				if err != nil {
					return schema.ComplexRestriction{}, err
				}
				restriction.Content = append(restriction.Content, schema.Particle{Kind: schema.ParticleKindGroup, Group: &groupRef})
			case "attribute":
				attribute, err := p.parseAttribute(token)
				if err != nil {
					return schema.ComplexRestriction{}, err
				}
				restriction.Attributes = append(restriction.Attributes, attribute)
			case "attributeGroup":
				group, err := p.parseAttributeGroupRef(token)
				if err != nil {
					return schema.ComplexRestriction{}, err
				}
				restriction.AttributeGroups = append(restriction.AttributeGroups, group)
			case "anyAttribute":
				restriction.AnyAttributes = append(restriction.AnyAttributes, parseAnyAttribute(token))
				if err := p.skip(token); err != nil {
					return schema.ComplexRestriction{}, err
				}
			default:
				if err := p.skip(token); err != nil {
					return schema.ComplexRestriction{}, err
				}
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return restriction, nil
			}
		}
	}
}

func (p *parser) parseComplexExtension(start xml.StartElement) (schema.ComplexExtension, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	base := attr(start, "base")
	extension := schema.ComplexExtension{Base: base, BaseName: p.parseQName(base)}
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return extension, nil
			}
			return schema.ComplexExtension{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space != xsdNamespace {
				if err := p.skip(token); err != nil {
					return schema.ComplexExtension{}, err
				}
				continue
			}
			switch token.Name.Local {
			case "sequence", "choice", "all":
				group, err := p.parseParticleGroup(token)
				if err != nil {
					return schema.ComplexExtension{}, err
				}
				extension.Content = append(extension.Content, schema.Particle{Kind: schema.ParticleKindGroup, Group: &group})
			case "group":
				groupRef, err := p.parseGroupRef(token)
				if err != nil {
					return schema.ComplexExtension{}, err
				}
				extension.Content = append(extension.Content, schema.Particle{Kind: schema.ParticleKindGroup, Group: &groupRef})
			case "attribute":
				attribute, err := p.parseAttribute(token)
				if err != nil {
					return schema.ComplexExtension{}, err
				}
				extension.Attributes = append(extension.Attributes, attribute)
			case "attributeGroup":
				group, err := p.parseAttributeGroupRef(token)
				if err != nil {
					return schema.ComplexExtension{}, err
				}
				extension.AttributeGroups = append(extension.AttributeGroups, group)
			case "anyAttribute":
				extension.AnyAttributes = append(extension.AnyAttributes, parseAnyAttribute(token))
				if err := p.skip(token); err != nil {
					return schema.ComplexExtension{}, err
				}
			default:
				if err := p.skip(token); err != nil {
					return schema.ComplexExtension{}, err
				}
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return extension, nil
			}
		}
	}
}

func (p *parser) parseSimpleContent(start xml.StartElement) (*schema.SimpleContent, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	simpleContent := &schema.SimpleContent{}
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return simpleContent, nil
			}
			return nil, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space == xsdNamespace && token.Name.Local == "extension" {
				extension, err := p.parseSimpleContentExtension(token)
				if err != nil {
					return nil, err
				}
				simpleContent.Extension = &extension
				continue
			}
			if token.Name.Space == xsdNamespace && token.Name.Local == "restriction" {
				restriction, err := p.parseSimpleContentRestriction(token)
				if err != nil {
					return nil, err
				}
				simpleContent.Restriction = &restriction
				continue
			}
			if err := p.skip(token); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return simpleContent, nil
			}
		}
	}
}

func (p *parser) parseSimpleContentExtension(start xml.StartElement) (schema.SimpleContentExtension, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	base := attr(start, "base")
	extension := schema.SimpleContentExtension{Base: base, BaseName: p.parseQName(base)}
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return extension, nil
			}
			return schema.SimpleContentExtension{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space == xsdNamespace && token.Name.Local == "attribute" {
				attribute, err := p.parseAttribute(token)
				if err != nil {
					return schema.SimpleContentExtension{}, err
				}
				extension.Attributes = append(extension.Attributes, attribute)
				continue
			}
			if token.Name.Space == xsdNamespace && token.Name.Local == "attributeGroup" {
				group, err := p.parseAttributeGroupRef(token)
				if err != nil {
					return schema.SimpleContentExtension{}, err
				}
				extension.AttributeGroups = append(extension.AttributeGroups, group)
				continue
			}
			if token.Name.Space == xsdNamespace && token.Name.Local == "anyAttribute" {
				extension.AnyAttributes = append(extension.AnyAttributes, parseAnyAttribute(token))
				if err := p.skip(token); err != nil {
					return schema.SimpleContentExtension{}, err
				}
				continue
			}
			if err := p.skip(token); err != nil {
				return schema.SimpleContentExtension{}, err
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return extension, nil
			}
		}
	}
}

func (p *parser) parseSimpleContentRestriction(start xml.StartElement) (schema.SimpleContentRestriction, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	base := attr(start, "base")
	restriction := schema.SimpleContentRestriction{Base: base, BaseName: p.parseQName(base)}
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return restriction, nil
			}
			return schema.SimpleContentRestriction{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space != xsdNamespace {
				if err := p.skip(token); err != nil {
					return schema.SimpleContentRestriction{}, err
				}
				continue
			}
			switch token.Name.Local {
			case "attribute":
				attribute, err := p.parseAttribute(token)
				if err != nil {
					return schema.SimpleContentRestriction{}, err
				}
				restriction.Attributes = append(restriction.Attributes, attribute)
			case "attributeGroup":
				group, err := p.parseAttributeGroupRef(token)
				if err != nil {
					return schema.SimpleContentRestriction{}, err
				}
				restriction.AttributeGroups = append(restriction.AttributeGroups, group)
			case "anyAttribute":
				restriction.AnyAttributes = append(restriction.AnyAttributes, parseAnyAttribute(token))
				if err := p.skip(token); err != nil {
					return schema.SimpleContentRestriction{}, err
				}
			case "annotation":
				if err := p.skip(token); err != nil {
					return schema.SimpleContentRestriction{}, err
				}
			default:
				restriction.Facets = append(restriction.Facets, schema.Facet{Name: token.Name.Local, Value: attr(token, "value")})
				if err := p.skip(token); err != nil {
					return schema.SimpleContentRestriction{}, err
				}
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return restriction, nil
			}
		}
	}
}
