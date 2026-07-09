package parser

import (
	"encoding/xml"
	"io"

	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func (p *parser) parseAttributeGroup(start xml.StartElement) (schema.AttributeGroup, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	name := attr(start, "name")
	group := schema.AttributeGroup{Name: name, QName: p.componentQName(name)}
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return group, nil
			}
			return schema.AttributeGroup{}, err
		}
		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space != xsdNamespace {
				if err := p.skip(token); err != nil {
					return schema.AttributeGroup{}, err
				}
				continue
			}
			switch token.Name.Local {
			case "annotation":
				if err := p.skip(token); err != nil {
					return schema.AttributeGroup{}, err
				}
			case "attribute":
				attribute, err := p.parseAttribute(token)
				if err != nil {
					return schema.AttributeGroup{}, err
				}
				group.Attributes = append(group.Attributes, attribute)
			case "attributeGroup":
				ref, err := p.parseAttributeGroupRef(token)
				if err != nil {
					return schema.AttributeGroup{}, err
				}
				group.AttributeGroups = append(group.AttributeGroups, ref)
			case "anyAttribute":
				group.AnyAttributes = append(group.AnyAttributes, parseAnyAttribute(token))
				if err := p.skip(token); err != nil {
					return schema.AttributeGroup{}, err
				}
			default:
				if err := p.skip(token); err != nil {
					return schema.AttributeGroup{}, err
				}
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return group, nil
			}
		}
	}
}

func (p *parser) parseAttributeGroupRef(start xml.StartElement) (schema.AttributeGroupRef, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()
	ref := attr(start, "ref")
	group := schema.AttributeGroupRef{Ref: ref, RefName: p.parseQName(ref)}
	if err := p.skip(start); err != nil {
		return schema.AttributeGroupRef{}, err
	}
	return group, nil
}

func (p *parser) parseAttribute(start xml.StartElement) (schema.Attribute, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	name := attr(start, "name")
	typeRef := attr(start, "type")
	ref := attr(start, "ref")
	attribute := schema.Attribute{
		Name:     name,
		Ref:      ref,
		Type:     typeRef,
		Use:      attr(start, "use"),
		Form:     attr(start, "form"),
		Default:  attr(start, "default"),
		Fixed:    attr(start, "fixed"),
		QName:    p.componentQName(name),
		RefName:  p.parseQName(ref),
		TypeName: p.parseQName(typeRef),
	}

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return attribute, nil
			}
			return schema.Attribute{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space != xsdNamespace {
				if err := p.skip(token); err != nil {
					return schema.Attribute{}, err
				}
				continue
			}
			switch token.Name.Local {
			case "annotation":
				annotation, err := p.parseAnnotation(token)
				if err != nil {
					return schema.Attribute{}, err
				}
				attribute.Annotation = annotation
			case "simpleType":
				simpleType, err := p.parseSimpleType(token)
				if err != nil {
					return schema.Attribute{}, err
				}
				attribute.AnonymousSimpleType = &simpleType
			default:
				if err := p.skip(token); err != nil {
					return schema.Attribute{}, err
				}
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return attribute, nil
			}
		}
	}
}
