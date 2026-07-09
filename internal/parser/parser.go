package parser

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

const xsdNamespace = "http://www.w3.org/2001/XMLSchema"

// ParseFiles parses one or more XSD files into the project's raw schema model.
// Local xs:include and xs:import schemaLocation values are resolved relative to
// the file that declares them and parsed once.
func ParseFiles(paths []string) (schema.Schemas, error) {
	return ParseFilesWithOptions(paths, Options{})
}

// ParseFilesWithOptions parses XSD files with resolver options.
func ParseFilesWithOptions(paths []string, options Options) (schema.Schemas, error) {
	catalogs, err := loadCatalogs(options.Catalogs)
	if err != nil {
		return schema.Schemas{}, err
	}
	var schemas schema.Schemas
	seen := make(map[string]struct{})
	queue := append([]string(nil), paths...)

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]

		key, err := filepath.Abs(path)
		if err != nil {
			return schema.Schemas{}, fmt.Errorf("resolve %s: %w", path, err)
		}
		key = filepath.Clean(key)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		file, err := ParseFile(path)
		if err != nil {
			return schema.Schemas{}, err
		}
		schemas.Files = append(schemas.Files, file)

		baseDir := filepath.Dir(path)
		for _, include := range file.Includes {
			if include.SchemaLocation == "" {
				continue
			}
			queue = append(queue, resolveSchemaLocationWithCatalog(baseDir, include.SchemaLocation, catalogs))
		}
		for _, imp := range file.Imports {
			if imp.SchemaLocation == "" {
				continue
			}
			queue = append(queue, resolveSchemaLocationWithCatalog(baseDir, imp.SchemaLocation, catalogs))
		}
		for _, override := range file.Overrides {
			if override.SchemaLocation == "" {
				continue
			}
			queue = append(queue, resolveSchemaLocationWithCatalog(baseDir, override.SchemaLocation, catalogs))
		}
		for _, redefine := range file.Redefines {
			if redefine.SchemaLocation == "" {
				continue
			}
			queue = append(queue, resolveSchemaLocationWithCatalog(baseDir, redefine.SchemaLocation, catalogs))
		}
	}

	applySchemaPatches(&schemas, catalogs)
	return schemas, nil
}

// ParseFile parses a single XSD file.
func ParseFile(path string) (schema.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return schema.File{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	file, err := Parse(path, f)
	if err != nil {
		return schema.File{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return file, nil
}

// Parse parses a single XSD document from r.
func Parse(path string, r io.Reader) (schema.File, error) {
	p := &parser{
		decoder: xml.NewDecoder(r),
		file: schema.File{
			Path:       path,
			Namespaces: map[string]string{"xml": "http://www.w3.org/XML/1998/namespace"},
		},
		nsStack: []map[string]string{{"xml": "http://www.w3.org/XML/1998/namespace"}},
	}

	for {
		tok, err := p.decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return schema.File{}, err
		}

		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Space == xsdNamespace && start.Name.Local == "schema" {
			if err := p.parseSchema(start); err != nil {
				return schema.File{}, err
			}
			return p.file, nil
		}
		if err := p.skip(start); err != nil {
			return schema.File{}, err
		}
	}

	return p.file, nil
}

type parser struct {
	decoder *xml.Decoder
	file    schema.File
	nsStack []map[string]string
}

func (p *parser) parseSchema(start xml.StartElement) error {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	p.file.TargetNamespace = attr(start, "targetNamespace")
	p.file.ElementFormDefault = attr(start, "elementFormDefault")
	p.file.AttributeFormDefault = attr(start, "attributeFormDefault")

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Space != xsdNamespace {
				if err := p.skip(token); err != nil {
					return err
				}
				continue
			}

			switch token.Name.Local {
			case "include":
				p.file.Includes = append(p.file.Includes, schema.Include{SchemaLocation: attr(token, "schemaLocation")})
				if err := p.skip(token); err != nil {
					return err
				}
			case "import":
				p.file.Imports = append(p.file.Imports, schema.Import{
					Namespace:      attr(token, "namespace"),
					SchemaLocation: attr(token, "schemaLocation"),
				})
				if err := p.skip(token); err != nil {
					return err
				}
			case "override":
				override, err := p.parseOverride(token)
				if err != nil {
					return err
				}
				p.file.Overrides = append(p.file.Overrides, override)
				p.file.Elements = append(p.file.Elements, override.Elements...)
				p.file.Attributes = append(p.file.Attributes, override.Attributes...)
				p.file.AttributeGroups = append(p.file.AttributeGroups, override.AttributeGroups...)
				p.file.Groups = append(p.file.Groups, override.Groups...)
				p.file.ComplexTypes = append(p.file.ComplexTypes, override.ComplexTypes...)
				p.file.SimpleTypes = append(p.file.SimpleTypes, override.SimpleTypes...)
			case "redefine":
				redefine, err := p.parseRedefine(token)
				if err != nil {
					return err
				}
				p.file.Redefines = append(p.file.Redefines, redefine)
				p.file.Elements = append(p.file.Elements, redefine.Elements...)
				p.file.Attributes = append(p.file.Attributes, redefine.Attributes...)
				p.file.AttributeGroups = append(p.file.AttributeGroups, redefine.AttributeGroups...)
				p.file.Groups = append(p.file.Groups, redefine.Groups...)
				p.file.ComplexTypes = append(p.file.ComplexTypes, redefine.ComplexTypes...)
				p.file.SimpleTypes = append(p.file.SimpleTypes, redefine.SimpleTypes...)
			case "element":
				element, err := p.parseElement(token)
				if err != nil {
					return err
				}
				p.file.Elements = append(p.file.Elements, element)
			case "attribute":
				attribute, err := p.parseAttribute(token)
				if err != nil {
					return err
				}
				p.file.Attributes = append(p.file.Attributes, attribute)
			case "attributeGroup":
				attributeGroup, err := p.parseAttributeGroup(token)
				if err != nil {
					return err
				}
				p.file.AttributeGroups = append(p.file.AttributeGroups, attributeGroup)
			case "group":
				group, err := p.parseGroup(token)
				if err != nil {
					return err
				}
				p.file.Groups = append(p.file.Groups, group)
			case "complexType":
				complexType, err := p.parseComplexType(token)
				if err != nil {
					return err
				}
				p.file.ComplexTypes = append(p.file.ComplexTypes, complexType)
			case "simpleType":
				simpleType, err := p.parseSimpleType(token)
				if err != nil {
					return err
				}
				p.file.SimpleTypes = append(p.file.SimpleTypes, simpleType)
			default:
				if err := p.skip(token); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

func (p *parser) skip(start xml.StartElement) error {
	p.pushNamespaceScope(start)
	defer p.popNamespaceScope()

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch token := tok.(type) {
		case xml.StartElement:
			if err := p.skip(token); err != nil {
				return err
			}
		case xml.EndElement:
			if token.Name.Space == start.Name.Space && token.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

func (p *parser) pushNamespaceScope(start xml.StartElement) {
	scope := make(map[string]string)
	for _, attr := range start.Attr {
		switch {
		case attr.Name.Space == "xmlns":
			scope[attr.Name.Local] = attr.Value
			p.file.Namespaces[attr.Name.Local] = attr.Value
		case attr.Name.Space == "" && attr.Name.Local == "xmlns":
			scope[""] = attr.Value
			p.file.Namespaces[""] = attr.Value
		}
	}
	p.nsStack = append(p.nsStack, scope)
}

func (p *parser) popNamespaceScope() {
	if len(p.nsStack) == 0 {
		return
	}
	p.nsStack = p.nsStack[:len(p.nsStack)-1]
}

func (p *parser) parseQName(raw string) schema.QName {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return schema.QName{}
	}

	qname := schema.QName{Raw: raw}
	if prefix, local, ok := strings.Cut(raw, ":"); ok {
		qname.Prefix = prefix
		qname.Local = local
		qname.Namespace = p.lookupNamespace(prefix)
		return qname
	}

	qname.Local = raw
	qname.Namespace = p.lookupNamespace("")
	if qname.Namespace == "" {
		qname.Namespace = p.file.TargetNamespace
	}
	return qname
}

func (p *parser) componentQName(local string) schema.QName {
	if local == "" {
		return schema.QName{}
	}
	return schema.QName{Raw: local, Local: local, Namespace: p.file.TargetNamespace}
}

func (p *parser) lookupNamespace(prefix string) string {
	for i := len(p.nsStack) - 1; i >= 0; i-- {
		if namespace, ok := p.nsStack[i][prefix]; ok {
			return namespace
		}
	}
	return ""
}

func attr(start xml.StartElement, local string) string {
	for _, attr := range start.Attr {
		if attr.Name.Space == "" && attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

func parseMinOccurs(raw string) int {
	if raw == "" {
		return schema.OccursDefault
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return schema.OccursDefault
	}
	return value
}

func parseMaxOccurs(raw string) int {
	if raw == "" {
		return schema.OccursDefault
	}
	if raw == "unbounded" {
		return schema.OccursUnbounded
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return schema.OccursDefault
	}
	return value
}

func parseBool(raw string) bool {
	switch strings.TrimSpace(raw) {
	case "true", "1":
		return true
	default:
		return false
	}
}

func resolveSchemaLocationWithCatalog(baseDir string, location string, catalogs catalogResolver) string {
	if resolved, ok := catalogs.resolve(location); ok {
		return resolved
	}
	return resolveSchemaLocation(baseDir, location)
}

func resolveSchemaLocation(baseDir string, location string) string {
	if filepath.IsAbs(location) {
		return location
	}
	return filepath.Clean(filepath.Join(baseDir, location))
}
