package generator

import (
	"sort"
	"strings"

	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

// Package is the Go-specific code generation model.
type Package struct {
	Name  string
	Files []File
}

// Options controls Go lowering behavior.
type Options struct {
	Package               string
	ModulePath            string
	NamespacePackages     map[string]string
	AutoNamespacePackages bool
	NamespacePrefixes     map[string]string
	RuntimePackage        string
	OutputFile            string
	ValidateOnUnmarshal   bool
	OptimizerSerde        bool
	Catalogs              []string
	TypeMappings          map[string]string
	TypeNameOverrides     map[string]string
	FieldNameOverrides    map[string]string
}

// XMLNameDecl is one global element or attribute declaration QName.
type XMLNameDecl struct {
	Namespace string
	Local     string
}

// File is one generated Go source file.
type File struct {
	Name    string
	Package string
	Imports []ImportDecl
	Types   []TypeDecl

	IsRuntime              bool
	RuntimePackage         string
	ValidateOnUnmarshal    bool
	Interfaces             []InterfaceDecl
	Registries             []RegistryDecl
	NeedsQName             bool
	NeedsNillable          bool
	NeedsAny               bool
	NeedsValidate          bool
	NeedsValidationHelper  bool
	NeedsRegexp            bool
	NeedsStrings           bool
	NeedsNamespacePrefixes bool
	NamespacePrefixes      []NamespacePrefixDecl
	ElementDeclarations    []XMLNameDecl
	AttributeDeclarations  []XMLNameDecl
}

// ImportDecl is one generated Go import.
type ImportDecl struct {
	Alias string
	Path  string
}

// NamespacePrefixDecl is one deterministic namespace URI -> XML prefix mapping.
type NamespacePrefixDecl struct {
	Namespace string
	Prefix    string
}

// InterfaceDecl is a generated polymorphic interface.
type InterfaceDecl struct {
	Name         string
	MarkerMethod string
}

// RegistryDecl is a generated QName -> constructor registry for an interface.
type RegistryDecl struct {
	VarName       string
	InterfaceName string
	Entries       []RegistryEntry
}

// RegistryEntry is one candidate in a polymorphic registry.
type RegistryEntry struct {
	DispatchNS    string
	DispatchLocal string
	ElementNS     string
	ElementLocal  string
	ConcreteType  string
	XSITypeNS     string
	XSITypeLocal  string
	UseXSIType    bool
}

// TypeDecl is a generated Go type declaration.
type TypeDecl struct {
	Name           string
	Kind           string
	Documentation  []string
	Alias          string
	AliasEqual     bool
	AliasGenerated bool
	AliasValidate  bool
	Fields         []Field
	Choice         *ChoiceDecl
	Mixed          *MixedContentDecl
	Consts         []ConstDecl
	Facets         []FacetDecl
	List           *ListDecl
	Union          *UnionDecl
	Validate       bool

	XMLName        string
	MarkerMethods  []string
	ImplInterfaces []string
	CustomXML      *CustomXMLDecl
}

// ChoiceDecl describes a generated wrapper model for xs:choice.
type ChoiceDecl struct {
	InterfaceName string
	MarkerMethod  string
	Repeated      bool
	MinOccurs     int
	MaxOccurs     int
	Variants      []ChoiceVariant
}

// MixedContentDecl describes a generated order-preserving model for mixed content.
type MixedContentDecl struct {
	TokenTypeName string
	InterfaceName string
	MarkerMethod  string
	Elements      []MixedElement
	HasText       bool
}

// MixedElement is one known element variant in a mixed content model.
type MixedElement struct {
	StructName   string
	FieldType    string
	XMLName      string
	XMLNamespace string
}

// ChoiceVariant is one possible branch of an xs:choice.
type ChoiceVariant struct {
	Name         string
	StructName   string
	MarkerMethod string
	FieldType    string
	XMLName      string
	XMLNamespace string
}

// CustomXMLDecl describes generated MarshalXML/UnmarshalXML for a container.
type CustomXMLDecl struct {
	ElementNS    string
	ElementLocal string
	Fields       []CustomXMLFieldDecl
}

// CustomXMLFieldDecl is one polymorphic field handled by custom XML methods.
type CustomXMLFieldDecl struct {
	FieldName   string
	RegistryVar string
	Repeated    bool
}

// ListDecl is one generated xs:list simple type.
type ListDecl struct {
	ItemType string
}

// UnionDecl is one generated xs:union simple type.
type UnionDecl struct {
	MemberTypes []string
}

// FacetDecl is one generated validation rule from XSD facets.
type FacetDecl struct {
	Name  string
	Value string
}

// ConstDecl is a generated Go constant declaration.
type ConstDecl struct {
	Name   string
	Type   string
	Value  string
	Legacy bool
}

// Field is a generated Go struct field.
type Field struct {
	Name               string
	Documentation      []string
	Type               string
	XMLName            string
	XMLNamespace       string
	Attribute          bool
	AnyAttribute       bool
	AnyElement         bool
	AnyNamespace       string
	AnyProcessContents string
	AnyTargetNamespace string
	Chardata           bool
	Choice             bool
	ChoiceModel        bool
	Polymorphic        bool
	RegistryVar        string
	Nillable           bool
	Optional           bool
	Required           bool
	Repeated           bool
	MinOccurs          int
	MaxOccurs          int
	Default            string
	Fixed              string
}

type qnameKey struct {
	namespace string
	local     string
}

type declarationKey struct {
	kind      string
	namespace string
	local     string
}

type declarationName struct {
	key      declarationKey
	base     string
	typeName schema.QName
}

type nameRegistry struct {
	declarations        map[declarationKey]string
	types               map[qnameKey]string
	implementationTypes map[qnameKey]string
	validatableTypes    map[qnameKey]bool
}

// Generate lowers semantic meta types into a Go-specific model.
func Generate(meta interpreter.MetaTypes, packageName string) (Package, error) {
	return GenerateWithOptions(meta, Options{Package: packageName})
}

// GenerateWithOptions lowers semantic meta types into a Go-specific model using options.
func GenerateWithOptions(meta interpreter.MetaTypes, options Options) (Package, error) {
	if options.Package == "" {
		options.Package = "model"
	}
	registry := buildNameRegistry(meta, options)
	outputFile := generatedOutputFile(options)
	files := make(map[string]*File)
	importSets := make(map[string]map[string]ImportDecl)
	legacyAliases := make(map[string][]TypeDecl)
	getFile := func(namespace string) *File {
		pkgName := packageForNamespace(namespace, options)
		name := outputFile
		if pkgName != options.Package || len(options.NamespacePackages) > 0 {
			name = pkgName + "/" + outputFile
		}
		if file := files[pkgName]; file != nil {
			return file
		}
		file := &File{Name: name, Package: pkgName, RuntimePackage: options.RuntimePackage, ValidateOnUnmarshal: options.ValidateOnUnmarshal}
		files[pkgName] = file
		importSets[pkgName] = make(map[string]ImportDecl)
		return file
	}

	for _, metaType := range meta.Types {
		if metaType.Name == "" {
			continue
		}
		file := getFile(metaType.Namespace)
		imports := importSets[file.Package]
		decl := TypeDecl{
			Name:          registry.declarationName(metaType.Kind, metaType.Namespace, metaType.Name),
			Kind:          metaType.Kind,
			Documentation: append([]string(nil), metaType.Documentation...),
		}
		if metaType.DuplicateOf.Local != "" {
			decl.Alias = registry.typeNameForPackage(metaType.DuplicateOf, file.Package, options, imports)
			decl.AliasEqual = true
			file.Types = append(file.Types, decl)
			addLegacyAlias(legacyAliases, file.Package, metaType.Name, decl)
			continue
		}
		switch metaType.Kind {
		case "complexType":
			decl.Fields = generateFields(registry, metaType.Fields, file.Package, options, imports)
			decl.Validate = len(decl.Fields) > 0
			if metaType.Mixed {
				if mixed := buildMixedContentDecl(decl.Name, decl.Fields); mixed != nil {
					decl.Mixed = mixed
					decl.Fields = []Field{{Name: "Content", Type: "[]" + mixed.TokenTypeName, ChoiceModel: true, Repeated: true}}
				} else {
					decl.Mixed = &MixedContentDecl{TokenTypeName: decl.Name + "Content", HasText: true}
					decl.Fields = []Field{{Name: "Content", Type: "[]" + decl.Name + "Content", ChoiceModel: true, Repeated: true}}
				}
				decl.Validate = true
			}
			if choice := buildChoiceDecl(decl.Name, decl.Fields); choice != nil {
				decl.Choice = choice
				contentType := choice.InterfaceName
				if choice.Repeated {
					contentType = "[]" + contentType
				}
				decl.Fields = []Field{{Name: "Content", Type: contentType, ChoiceModel: true, Repeated: choice.Repeated, Required: choice.MinOccurs > 0, MinOccurs: choice.MinOccurs, MaxOccurs: choice.MaxOccurs}}
			}
			for i := range metaType.PolymorphicImpls {
				applyImpl(&decl, registry, &metaType.PolymorphicImpls[i])
			}
			if metaType.CustomXML != nil {
				applyCustomXML(&decl, metaType.CustomXML)
			}
		case "simpleType":
			if metaType.List != nil {
				decl.List = &ListDecl{ItemType: registry.typeNameForPackage(metaType.List.ItemName, file.Package, options, imports)}
				if decl.List.ItemType == "" {
					decl.List.ItemType = "string"
				}
			} else if metaType.Union != nil {
				decl.Alias = "string"
				decl.Union = &UnionDecl{}
				for _, member := range metaType.Union.MemberNames {
					memberType := registry.typeNameForPackage(member, file.Package, options, imports)
					if memberType != "" {
						decl.Union.MemberTypes = append(decl.Union.MemberTypes, memberType)
					}
				}
			} else {
				decl.Alias = registry.typeNameForPackage(aliasBaseName(metaType), file.Package, options, imports)
			}
			decl.Consts = generateEnumConsts(decl.Name, metaType)
			decl.Facets = generateFacetDecls(decl.Alias, metaType)
			for i := range metaType.PolymorphicImpls {
				applyImpl(&decl, registry, &metaType.PolymorphicImpls[i])
			}
		case "element":
			decl.XMLName = metaType.Namespace + " " + metaType.XMLName
			if metaType.SkipElementAlias {
				continue
			}
			if metaType.TypeName.Local != "" {
				if mapped := mappedTypeName(metaType.TypeName, file.Package, options, imports); mapped != "" {
					decl.Alias = mapped
				} else {
					implementationName := registry.implementationTypeName(metaType.TypeName)
					decl.Alias = qualifyIdentifier(implementationName, metaType.TypeName.Namespace, file.Package, options, imports)
					publicName := registry.typeName(metaType.TypeName)
					if publicName == decl.Name && implementationName != publicName {
						decl.AliasGenerated = registry.hasImplementationType(metaType.TypeName)
						decl.AliasValidate = registry.typeHasValidate(metaType.TypeName)
					}
				}
			} else if metaType.TypeRef != "" {
				decl.Alias = scalarType(schema.QName{Raw: metaType.TypeRef, Local: metaType.TypeRef})
			}
			if metaType.Nillable && decl.Alias != "" {
				decl.Alias = runtimeType("Nillable", file.Package, options, imports) + "[" + decl.Alias + "]"
				decl.AliasEqual = true
				decl.AliasGenerated = false
				decl.AliasValidate = false
			}
		}
		file.Types = append(file.Types, decl)
		addLegacyAlias(legacyAliases, file.Package, metaType.Name, decl)
	}

	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	declarations := declarationDecls(meta.Declarations)
	result := make([]File, 0, len(keys)+1)
	if options.RuntimePackage != "" {
		result = append(result, File{
			Name:                  options.RuntimePackage + "/" + outputFile,
			Package:               options.RuntimePackage,
			IsRuntime:             true,
			ElementDeclarations:   declarations.elements,
			AttributeDeclarations: declarations.attributes,
		})
	}
	for _, key := range keys {
		file := *files[key]
		applyInterfaces(&file, registry, meta, options, importSets[key])
		if needsCustomXML(file) {
			file.NeedsQName = true
			if options.RuntimePackage != "" {
				importSets[key][options.RuntimePackage] = ImportDecl{Alias: options.RuntimePackage, Path: importPathForPackage(options.RuntimePackage, options)}
			}
		}
		if options.RuntimePackage != "" && (len(file.Registries) > 0 || needsValidationHelper(file) || len(collectNamespacePrefixes(file, options)) > 0) {
			importSets[key][options.RuntimePackage] = ImportDecl{Alias: options.RuntimePackage, Path: importPathForPackage(options.RuntimePackage, options)}
		}
		appendLegacyAliases(&file, legacyAliases[key])
		file.Imports = sortedImports(importSets[key])
		file.NeedsNillable = needsNillable(file)
		file.NeedsAny = needsAny(file)
		if file.NeedsAny {
			file.ElementDeclarations = declarations.elements
			file.AttributeDeclarations = declarations.attributes
		}
		file.NeedsValidate, file.NeedsRegexp = needsValidation(file)
		file.NeedsValidationHelper = needsValidationHelper(file)
		file.NeedsStrings = needsStrings(file)
		file.NamespacePrefixes = collectNamespacePrefixes(file, options)
		file.NeedsNamespacePrefixes = len(file.NamespacePrefixes) > 0
		result = append(result, file)
	}
	return Package{Name: options.Package, Files: result}, nil
}

func generatedOutputFile(options Options) string {
	if options.OutputFile != "" {
		return options.OutputFile
	}
	return "models.go"
}

func addLegacyAlias(aliases map[string][]TypeDecl, packageName string, localName string, decl TypeDecl) {
	legacy := legacyGoName(localName)
	if legacy == "" || legacy == decl.Name || decl.Name == "" {
		return
	}
	aliases[packageName] = append(aliases[packageName], TypeDecl{
		Name:       legacy,
		Kind:       decl.Kind,
		Alias:      decl.Name,
		AliasEqual: true,
	})
}

func appendLegacyAliases(file *File, aliases []TypeDecl) {
	if len(aliases) == 0 {
		return
	}
	used := make(map[string]bool)
	for _, typ := range file.Types {
		if typ.Name != "" {
			used[typ.Name] = true
		}
	}
	for _, iface := range file.Interfaces {
		used[iface.Name] = true
	}
	for _, alias := range aliases {
		if used[alias.Name] || alias.Name == alias.Alias {
			continue
		}
		used[alias.Name] = true
		file.Types = append(file.Types, alias)
	}
}

func legacyGoName(name string) string {
	words := legacyNameWords(name)
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}
		for i := 0; i < len(word); i++ {
			ch := word[i]
			if i == 0 {
				if ch >= 'a' && ch <= 'z' {
					ch -= 'a' - 'A'
				}
			} else if ch >= 'A' && ch <= 'Z' {
				ch += 'a' - 'A'
			}
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_' || (ch >= '0' && ch <= '9' && b.Len() > 0) {
				b.WriteByte(ch)
			}
		}
	}
	out := b.String()
	if out == "" {
		return ""
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "Value" + out
	}
	return out
}

func legacyEnumConstPart(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		for i := 0; i < len(part); i++ {
			ch := part[i]
			if ch >= 'A' && ch <= 'Z' {
				ch += 'a' - 'A'
			}
			if i == 0 && ch >= 'a' && ch <= 'z' {
				ch -= 'a' - 'A'
			}
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_' || (ch >= '0' && ch <= '9' && b.Len() > 0) {
				b.WriteByte(ch)
			}
		}
	}
	out := b.String()
	if out == "" {
		return ""
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "Value" + out
	}
	return out
}

func legacyNameWords(name string) []string {
	var words []string
	start := -1
	var prev byte
	for i := 0; i < len(name); i++ {
		ch := name[i]
		valid := (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_'
		if !valid {
			if start >= 0 {
				words = append(words, name[start:i])
				start = -1
			}
			prev = 0
			continue
		}
		if start < 0 {
			start = i
		} else if isLegacyWordBoundary(prev, ch) {
			words = append(words, name[start:i])
			start = i
		}
		prev = ch
	}
	if start >= 0 {
		words = append(words, name[start:])
	}
	return words
}

func isLegacyWordBoundary(prev byte, cur byte) bool {
	return ((prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9')) && cur >= 'A' && cur <= 'Z'
}
