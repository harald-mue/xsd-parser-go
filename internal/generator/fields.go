package generator

import (
	"sort"
	"strings"

	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func buildMixedContentDecl(typeName string, fields []Field) *MixedContentDecl {
	elements := make([]MixedElement, 0, len(fields))
	seenXML := make(map[string]bool)
	for _, field := range fields {
		if field.Attribute || field.AnyAttribute || field.Chardata || field.Polymorphic || field.ChoiceModel {
			continue
		}
		key := field.XMLNamespace + "\x00" + field.XMLName
		if seenXML[key] {
			continue
		}
		seenXML[key] = true
		fieldType := strings.TrimPrefix(field.Type, "*")
		fieldType = strings.TrimPrefix(fieldType, "[]")
		elements = append(elements, MixedElement{
			StructName:   typeName + "Content" + field.Name,
			FieldType:    fieldType,
			XMLName:      field.XMLName,
			XMLNamespace: field.XMLNamespace,
		})
	}
	tokenType := typeName + "Content"
	iface := tokenType + "Element"
	marker := strings.ToLower(tokenType[:1]) + tokenType[1:] + "Element"
	return &MixedContentDecl{
		TokenTypeName: tokenType,
		InterfaceName: iface,
		MarkerMethod:  marker,
		Elements:      elements,
		HasText:       true,
	}
}

func buildChoiceDecl(typeName string, fields []Field) *ChoiceDecl {
	if len(fields) < 2 {
		return nil
	}
	seenXML := make(map[string]bool)
	repeated := fields[0].Repeated
	for _, field := range fields {
		if !field.Choice || field.Attribute || field.AnyAttribute || field.Chardata || field.Polymorphic || field.Repeated != repeated {
			return nil
		}
		key := field.XMLNamespace + "\x00" + field.XMLName
		if seenXML[key] {
			return nil
		}
		seenXML[key] = true
	}
	iface := typeName + "Content"
	marker := "is" + iface
	minOccurs := fields[0].MinOccurs
	maxOccurs := fields[0].MaxOccurs
	if !repeated {
		maxOccurs = 1
	}
	choice := &ChoiceDecl{InterfaceName: iface, MarkerMethod: marker, Repeated: repeated, MinOccurs: minOccurs, MaxOccurs: maxOccurs}
	for _, field := range fields {
		fieldType := strings.TrimPrefix(field.Type, "*")
		if repeated {
			fieldType = strings.TrimPrefix(fieldType, "[]")
		}
		choice.Variants = append(choice.Variants, ChoiceVariant{
			Name:         field.Name,
			StructName:   iface + field.Name,
			MarkerMethod: marker,
			FieldType:    fieldType,
			XMLName:      field.XMLName,
			XMLNamespace: field.XMLNamespace,
		})
	}
	return choice
}

func generateFields(registry nameRegistry, metaFields []interpreter.MetaField, currentPackage string, options Options, imports map[string]ImportDecl) []Field {
	fields := make([]Field, 0, len(metaFields))
	fieldNames := fieldNameRegistry(metaFields)
	for i, metaField := range metaFields {
		if metaField.Name == "" {
			continue
		}
		fieldName := fieldNames[i]
		if override := fieldNameOverride(metaField.Namespace, metaField.XMLName, options); override != metaField.XMLName && override != "" {
			fieldName = goName(override)
		}

		if metaField.Polymorphic != nil {
			fieldType := qualifyIdentifier(metaField.Polymorphic.InterfaceName, metaField.Namespace, currentPackage, options, imports)
			registryVar := qualifyIdentifier(metaField.Polymorphic.RegistryVar, metaField.Namespace, currentPackage, options, imports)
			if metaField.Polymorphic.Repeated {
				fieldType = "[]" + fieldType
			}
			fields = append(fields, Field{
				Name:          fieldName,
				Documentation: append([]string(nil), metaField.Documentation...),
				Type:          fieldType,
				Polymorphic:   true,
				RegistryVar:   registryVar,
				Repeated:      metaField.Polymorphic.Repeated,
				Required:      metaField.MinOccurs > 0,
				MinOccurs:     metaField.MinOccurs,
				MaxOccurs:     metaField.MaxOccurs,
			})
			continue
		}

		fieldType := registry.typeNameForPackage(metaField.TypeName, currentPackage, options, imports)
		if metaField.AnyElement || (isXSDNamespace(metaField.TypeName.Namespace) && metaField.TypeName.Local == "anyType") {
			fieldType = runtimeType("AnyElement", currentPackage, options, imports)
		} else if metaField.AnyAttribute {
			fieldType = runtimeType("AnyAttributes", currentPackage, options, imports)
		} else if fieldType == "" {
			fieldType = "string"
		}
		repeated := metaField.MaxOccurs == schema.OccursUnbounded || metaField.MaxOccurs > 1
		optional := !metaField.Attribute && !metaField.Chardata && !metaField.AnyAttribute && (metaField.MinOccurs == 0 || metaField.Choice)
		if metaField.Attribute {
			optional = !metaField.Required
		}
		if metaField.Nillable {
			fieldType = runtimeType("Nillable", currentPackage, options, imports) + "[" + fieldType + "]"
		}
		if metaField.AnyAttribute {
			repeated = false
		} else if repeated {
			fieldType = "[]" + fieldType
		} else if optional {
			fieldType = "*" + fieldType
		}

		fields = append(fields, Field{
			Name:               fieldName,
			Documentation:      append([]string(nil), metaField.Documentation...),
			Type:               fieldType,
			XMLName:            metaField.XMLName,
			XMLNamespace:       metaField.Namespace,
			Attribute:          metaField.Attribute,
			AnyAttribute:       metaField.AnyAttribute,
			AnyElement:         metaField.AnyElement,
			AnyNamespace:       metaField.AnyNamespace,
			AnyProcessContents: metaField.AnyProcessContents,
			AnyTargetNamespace: metaField.AnyTargetNamespace,
			Chardata:           metaField.Chardata,
			Choice:             metaField.Choice,
			Nillable:           metaField.Nillable,
			Optional:           optional,
			Required:           (metaField.Attribute && metaField.Required) || (!metaField.Attribute && metaField.MinOccurs > 0 && !metaField.Choice),
			Repeated:           repeated,
			MinOccurs:          metaField.MinOccurs,
			MaxOccurs:          metaField.MaxOccurs,
			Default:            metaField.Default,
			Fixed:              metaField.Fixed,
		})
	}
	return fields
}

func fieldNameRegistry(metaFields []interpreter.MetaField) map[int]string {
	bases := make(map[string][]int)
	for i, metaField := range metaFields {
		base := goName(metaField.Name)
		if base == "" {
			continue
		}
		bases[base] = append(bases[base], i)
	}

	used := make(map[string]int)
	result := make(map[int]string, len(metaFields))
	baseNames := make([]string, 0, len(bases))
	for base := range bases {
		baseNames = append(baseNames, base)
	}
	sort.Strings(baseNames)

	for _, base := range baseNames {
		indexes := bases[base]
		multipleNamespaces := false
		for i := 1; i < len(indexes); i++ {
			if metaFields[indexes[i]].Namespace != metaFields[indexes[0]].Namespace {
				multipleNamespaces = true
				break
			}
		}
		for i, index := range indexes {
			candidate := base
			if len(indexes) > 1 {
				if multipleNamespaces {
					candidate = namespacePrefix(metaFields[index].Namespace) + base
				} else if i > 0 {
					candidate = base + strconvSuffix(i+1)
				}
			}
			result[index] = uniqueName(candidate, used)
		}
	}
	return result
}

func generateFacetDecls(alias string, metaType interpreter.MetaType) []FacetDecl {
	if metaType.Alias == nil {
		return nil
	}
	var facets []FacetDecl
	for _, facet := range metaType.Alias.Facets {
		switch facet.Name {
		case "enumeration":
			if !isNumericAlias(alias) {
				facets = append(facets, FacetDecl{Name: facet.Name, Value: facet.Value})
			}
		case "length", "minLength", "maxLength", "pattern", "whiteSpace":
			if isStringAlias(alias) || !isNumericAlias(alias) {
				facets = append(facets, FacetDecl{Name: facet.Name, Value: facet.Value})
			}
		case "totalDigits", "fractionDigits":
			facets = append(facets, FacetDecl{Name: facet.Name, Value: facet.Value})
		case "minInclusive", "maxInclusive", "minExclusive", "maxExclusive":
			if isNumericAlias(alias) {
				facets = append(facets, FacetDecl{Name: facet.Name, Value: facet.Value})
			}
		}
	}
	return facets
}

func isStringAlias(alias string) bool {
	return alias == "" || alias == "string"
}

func isNumericAlias(alias string) bool {
	switch alias {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return true
	default:
		return false
	}
}

func generateEnumConsts(typeName string, metaType interpreter.MetaType) []ConstDecl {
	if metaType.Alias == nil {
		return nil
	}
	seenValues := make(map[string]struct{})
	usedNames := make(map[string]int)
	var consts []ConstDecl
	for _, facet := range metaType.Alias.Facets {
		if facet.Name != "enumeration" {
			continue
		}
		if _, ok := seenValues[facet.Value]; ok {
			continue
		}
		seenValues[facet.Value] = struct{}{}
		base := goName(facet.Value)
		if base == "" {
			base = "Value" + strconvSuffix(len(consts)+1)
		}
		name := uniqueName(typeName+base, usedNames)
		consts = append(consts, ConstDecl{Name: name, Type: typeName, Value: facet.Value})
		legacyBase := legacyEnumConstPart(facet.Value)
		if legacyBase != "" && legacyBase != base {
			legacyName := typeName + legacyBase
			if usedNames[legacyName] == 0 {
				usedNames[legacyName] = 1
				consts = append(consts, ConstDecl{Name: legacyName, Type: typeName, Value: facet.Value, Legacy: true})
			}
		}
	}
	return consts
}
