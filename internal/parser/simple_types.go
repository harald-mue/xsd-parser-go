package parser

import (
	"encoding/xml"
	"io"
	"strings"

	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func (p *parser) parseSimpleType(start xml.StartElement) (schema.SimpleType, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	name := attr(start, "name")
	simpleType := schema.SimpleType{Name: name, QName: p.componentQName(name)}

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return simpleType, nil
			}
			return schema.SimpleType{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space != xsdNamespace {
				if err := p.skip(token); err != nil {
					return schema.SimpleType{}, err
				}
				continue
			}
			switch token.Name.Local {
			case "annotation":
				annotation, err := p.parseAnnotation(token)
				if err != nil {
					return schema.SimpleType{}, err
				}
				simpleType.Annotation = annotation
			case "restriction":
				restriction, err := p.parseSimpleRestriction(token)
				if err != nil {
					return schema.SimpleType{}, err
				}
				simpleType.Restriction = &restriction
			case "list":
				list, err := p.parseSimpleList(token)
				if err != nil {
					return schema.SimpleType{}, err
				}
				simpleType.List = &list
			case "union":
				union, err := p.parseSimpleUnion(token)
				if err != nil {
					return schema.SimpleType{}, err
				}
				simpleType.Union = &union
			default:
				if err := p.skip(token); err != nil {
					return schema.SimpleType{}, err
				}
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return simpleType, nil
			}
		}
	}
}

func (p *parser) parseSimpleRestriction(start xml.StartElement) (schema.SimpleRestriction, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	base := attr(start, "base")
	restriction := schema.SimpleRestriction{Base: base, BaseName: p.parseQName(base)}

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return restriction, nil
			}
			return schema.SimpleRestriction{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space == xsdNamespace && token.Name.Local != "annotation" {
				restriction.Facets = append(restriction.Facets, schema.Facet{Name: token.Name.Local, Value: attr(token, "value")})
			}
			if err := p.skip(token); err != nil {
				return schema.SimpleRestriction{}, err
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return restriction, nil
			}
		}
	}
}

func (p *parser) parseSimpleList(start xml.StartElement) (schema.SimpleList, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	itemType := attr(start, "itemType")
	list := schema.SimpleList{ItemType: itemType, ItemTypeName: p.parseQName(itemType)}

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return list, nil
			}
			return schema.SimpleList{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space == xsdNamespace && token.Name.Local == "simpleType" {
				simpleType, err := p.parseSimpleType(token)
				if err != nil {
					return schema.SimpleList{}, err
				}
				list.SimpleType = &simpleType
				continue
			}
			if err := p.skip(token); err != nil {
				return schema.SimpleList{}, err
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return list, nil
			}
		}
	}
}

func (p *parser) parseSimpleUnion(start xml.StartElement) (schema.SimpleUnion, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	memberTypes := attr(start, "memberTypes")
	union := schema.SimpleUnion{MemberTypes: memberTypes}
	for _, memberType := range strings.Fields(memberTypes) {
		union.MemberTypeNames = append(union.MemberTypeNames, p.parseQName(memberType))
	}

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return union, nil
			}
			return schema.SimpleUnion{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space == xsdNamespace && token.Name.Local == "simpleType" {
				simpleType, err := p.parseSimpleType(token)
				if err != nil {
					return schema.SimpleUnion{}, err
				}
				union.SimpleTypes = append(union.SimpleTypes, simpleType)
				continue
			}
			if err := p.skip(token); err != nil {
				return schema.SimpleUnion{}, err
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return union, nil
			}
		}
	}
}
