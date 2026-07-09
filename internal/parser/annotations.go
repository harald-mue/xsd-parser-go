package parser

import (
	"encoding/xml"
	"io"
	"strings"

	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func (p *parser) parseAnnotation(start xml.StartElement) (schema.Annotation, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	var annotation schema.Annotation
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return annotation, nil
			}
			return schema.Annotation{}, err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space == xsdNamespace && token.Name.Local == "documentation" {
				text, err := p.readTextElement(token)
				if err != nil {
					return schema.Annotation{}, err
				}
				text = strings.TrimSpace(text)
				if text != "" {
					annotation.Documentation = append(annotation.Documentation, text)
				}
				continue
			}
			if err := p.skip(token); err != nil {
				return schema.Annotation{}, err
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return annotation, nil
			}
		}
	}
}

func (p *parser) readTextElement(start xml.StartElement) (string, error) {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	var b strings.Builder
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return b.String(), nil
			}
			return "", err
		}
		switch token := tok.(type) {
		case xml.CharData:
			b.Write([]byte(token))
		case xml.StartElement:
			if err := p.skip(token); err != nil {
				return "", err
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return b.String(), nil
			}
		}
	}
}
