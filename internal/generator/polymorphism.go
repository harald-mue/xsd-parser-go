package generator

import (
	"strings"

	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

// applyImpl decorates a struct that is a concrete implementation of a
// polymorphic interface: it gets an XMLName field and marker methods.
func applyImpl(decl *TypeDecl, registry nameRegistry, impl *interpreter.PolymorphicImpl) {
	decl.ImplInterfaces = appendUnique(decl.ImplInterfaces, impl.InterfaceName)
	decl.MarkerMethods = appendUnique(decl.MarkerMethods, impl.MarkerMethod)
	if decl.XMLName == "" {
		decl.XMLName = impl.ElementNS + " " + impl.ElementLocal
	}
}

// applyCustomXML attaches custom Marshal/Unmarshal metadata to a container.
func applyCustomXML(decl *TypeDecl, custom *interpreter.CustomXML) {
	customDecl := &CustomXMLDecl{
		ElementNS:    custom.ElementNS,
		ElementLocal: custom.ElementLocal,
	}
	for _, field := range decl.Fields {
		if field.Polymorphic {
			customDecl.Fields = append(customDecl.Fields, CustomXMLFieldDecl{
				FieldName:   field.Name,
				RegistryVar: field.RegistryVar,
				Repeated:    field.Repeated,
			})
			continue
		}
		// Regular element fields (skip attributes, any-element, chardata, empty XMLName)
		if field.Attribute || field.AnyAttribute || field.AnyElement || field.Chardata || field.XMLName == "" {
			continue
		}
		customDecl.RegularFields = append(customDecl.RegularFields, CustomXMLRegularField{
			FieldName:    field.Name,
			XMLName:      field.XMLName,
			XMLNamespace: field.XMLNamespace,
			BaseType:     regularBaseType(field.Type),
			Optional:     field.Optional,
			Repeated:     field.Repeated,
		})
	}
	if len(customDecl.Fields) > 0 || len(customDecl.RegularFields) > 0 {
		decl.CustomXML = customDecl
	}
}

// regularBaseType strips leading * or [] from a Go type string.
func regularBaseType(typ string) string {
	if strings.HasPrefix(typ, "[]") {
		return typ[2:]
	}
	if strings.HasPrefix(typ, "*") {
		return typ[1:]
	}
	return typ
}

// applyInterfaces emits polymorphic interfaces and registries for a file.
func applyInterfaces(file *File, registry nameRegistry, meta interpreter.MetaTypes, options Options, imports map[string]ImportDecl) {
	if len(meta.Interfaces) == 0 {
		return
	}
	for _, iface := range meta.Interfaces {
		if packageForNamespace(iface.HeadNS, options) != file.Package {
			continue
		}
		file.NeedsQName = true
		file.Interfaces = append(file.Interfaces, InterfaceDecl{
			Name:         iface.Name,
			MarkerMethod: iface.MarkerMethod,
		})
		entries := make([]RegistryEntry, 0, len(iface.Members))
		for _, member := range iface.Members {
			concrete := registry.typeNameForPackage(schema.QName{
				Namespace: member.ConcreteTypeNS,
				Local:     member.ConcreteTypeLocal,
			}, file.Package, options, imports)
			if concrete == "" {
				continue
			}
			entries = append(entries, RegistryEntry{
				DispatchNS:    member.DispatchNS,
				DispatchLocal: member.DispatchLocal,
				ElementNS:     member.ElementNS,
				ElementLocal:  member.ElementLocal,
				ConcreteType:  concrete,
				XSITypeNS:     member.XSITypeNS,
				XSITypeLocal:  member.XSITypeLocal,
				UseXSIType:    member.UseXSIType,
			})
		}
		file.Registries = append(file.Registries, RegistryDecl{
			VarName:       iface.RegistryVar,
			InterfaceName: iface.Name,
			Entries:       entries,
		})
	}
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
