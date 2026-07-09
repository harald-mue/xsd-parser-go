package generator

import (
	"sort"
	"strings"

	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func buildNameRegistry(meta interpreter.MetaTypes, options Options) nameRegistry {
	registry := nameRegistry{
		declarations:        make(map[declarationKey]string),
		types:               make(map[qnameKey]string),
		implementationTypes: make(map[qnameKey]string),
		validatableTypes:    make(map[qnameKey]bool),
	}
	for _, metaType := range meta.Types {
		if metaType.Kind == "complexType" && len(metaType.Fields) > 0 {
			registry.validatableTypes[qnameKey{namespace: metaType.Namespace, local: metaType.Name}] = true
		}
	}

	declarations := collectDeclarations(meta, options)
	byBase := make(map[string][]declarationName)
	for _, declaration := range declarations {
		byBase[declaration.base] = append(byBase[declaration.base], declaration)
	}

	bases := make([]string, 0, len(byBase))
	for base := range byBase {
		bases = append(bases, base)
	}
	sort.Strings(bases)

	used := make(map[string]int)
	// Reserve polymorphic interface names so concrete types never steal them;
	// a type whose base collides with an interface is suffixed with "Type".
	interfaceNames := make(map[string]bool)
	for _, iface := range meta.Interfaces {
		interfaceNames[iface.Name] = true
		used[iface.Name] = 1
	}

	for _, base := range bases {
		group := byBase[base]
		sort.Slice(group, func(i, j int) bool {
			return declarationLess(group[i].key, group[j].key)
		})

		allSameNamespace := true
		for i := 1; i < len(group); i++ {
			if group[i].key.namespace != group[0].key.namespace {
				allSameNamespace = false
				break
			}
		}

		hasElement := false
		for _, declaration := range group {
			if declaration.key.kind == "element" {
				hasElement = true
				break
			}
		}

		assigned := make(map[declarationKey]string, len(group))
		for i, declaration := range group {
			name := base
			if len(group) > 1 {
				if allSameNamespace {
					name = base
					// When a global element and its named type collapse to the same
					// Go identifier, keep the wire element on the legacy/public name.
					// The reusable schema type gets a deterministic Type suffix instead
					// of forcing callers to use a suffixed element wrapper for root XML
					// marshaling.
					if hasElement {
						if declaration.key.kind != "element" {
							name += "Type"
						}
					} else if i > 0 {
						name += strconvSuffix(i + 1)
					}
				} else {
					name = namespacePrefix(declaration.key.namespace) + base
				}
			}
			if interfaceNames[name] {
				name += "Type"
			}
			name = uniqueName(name, used)
			assigned[declaration.key] = name
			registry.declarations[declaration.key] = name
			if declaration.key.kind == "complexType" || declaration.key.kind == "simpleType" {
				registry.implementationTypes[qnameKey{namespace: declaration.key.namespace, local: declaration.key.local}] = name
			}
		}
		for _, declaration := range group {
			if declaration.key.kind != "complexType" && declaration.key.kind != "simpleType" {
				continue
			}
			qkey := qnameKey{namespace: declaration.key.namespace, local: declaration.key.local}
			publicName := assigned[declaration.key]
			if hasElement {
				for _, candidate := range group {
					if candidate.key.kind == "element" && candidate.key.namespace == declaration.key.namespace && candidate.typeName.Namespace == declaration.key.namespace && candidate.typeName.Local == declaration.key.local {
						if name := assigned[candidate.key]; name != "" {
							publicName = name
							break
						}
					}
				}
			}
			registry.types[qkey] = publicName
		}
	}

	return registry
}

func collectDeclarations(meta interpreter.MetaTypes, options Options) []declarationName {
	declarations := make([]declarationName, 0, len(meta.Types))
	for _, metaType := range meta.Types {
		if metaType.Name == "" || (metaType.Kind == "element" && metaType.SkipElementAlias) {
			continue
		}
		base := goName(typeNameOverride(metaType.Namespace, metaType.Name, options))
		if base == "" {
			continue
		}
		declarations = append(declarations, declarationName{
			key: declarationKey{
				kind:      metaType.Kind,
				namespace: metaType.Namespace,
				local:     metaType.Name,
			},
			base:     base,
			typeName: metaType.TypeName,
		})
	}
	return declarations
}

func typeNameOverride(namespace, local string, options Options) string {
	if options.TypeNameOverrides != nil {
		for _, key := range []string{namespace + "#" + local, namespace + " " + local, local} {
			if value := options.TypeNameOverrides[key]; value != "" {
				return value
			}
		}
	}
	return local
}

func fieldNameOverride(namespace, local string, options Options) string {
	if options.FieldNameOverrides != nil {
		for _, key := range []string{namespace + "#" + local, namespace + " " + local, local} {
			if value := options.FieldNameOverrides[key]; value != "" {
				return value
			}
		}
	}
	return local
}

func declarationLess(a, b declarationKey) bool {
	if a.namespace != b.namespace {
		return a.namespace < b.namespace
	}
	if a.local != b.local {
		return a.local < b.local
	}
	return kindRank(a.kind) < kindRank(b.kind)
}

func kindRank(kind string) int {
	switch kind {
	case "complexType":
		return 0
	case "simpleType":
		return 1
	case "element":
		return 2
	default:
		return 10
	}
}

func uniqueName(candidate string, used map[string]int) string {
	if candidate == "" {
		candidate = "Value"
	}
	used[candidate]++
	if used[candidate] == 1 {
		return candidate
	}
	for n := used[candidate]; ; n++ {
		name := candidate + strconvSuffix(n)
		if used[name] == 0 {
			used[name] = 1
			return name
		}
	}
}

func (r nameRegistry) declarationName(kind string, namespace string, local string) string {
	key := declarationKey{kind: kind, namespace: namespace, local: local}
	if name, ok := r.declarations[key]; ok {
		return name
	}
	return goName(local)
}

func (r nameRegistry) typeName(qname schema.QName) string {
	if qname.Local == "" {
		return ""
	}
	if name, ok := r.types[qnameKey{namespace: qname.Namespace, local: qname.Local}]; ok {
		return name
	}
	if isXSDNamespace(qname.Namespace) {
		return scalarType(qname)
	}
	return goName(qname.Local)
}

func (r nameRegistry) typeNameForPackage(qname schema.QName, currentPackage string, options Options, imports map[string]ImportDecl) string {
	if mapped := mappedTypeName(qname, currentPackage, options, imports); mapped != "" {
		return mapped
	}
	name := r.typeName(qname)
	return qualifyIdentifier(name, qname.Namespace, currentPackage, options, imports)
}

func (r nameRegistry) implementationTypeName(qname schema.QName) string {
	if qname.Local == "" {
		return ""
	}
	if name, ok := r.implementationTypes[qnameKey{namespace: qname.Namespace, local: qname.Local}]; ok {
		return name
	}
	if isXSDNamespace(qname.Namespace) {
		return scalarType(qname)
	}
	return goName(qname.Local)
}

func (r nameRegistry) hasImplementationType(qname schema.QName) bool {
	_, ok := r.implementationTypes[qnameKey{namespace: qname.Namespace, local: qname.Local}]
	return ok
}

func (r nameRegistry) typeHasValidate(qname schema.QName) bool {
	return r.validatableTypes[qnameKey{namespace: qname.Namespace, local: qname.Local}]
}

func mappedTypeName(qname schema.QName, currentPackage string, options Options, imports map[string]ImportDecl) string {
	if len(options.TypeMappings) == 0 || qname.Local == "" {
		return ""
	}
	keys := []string{qname.Namespace + "#" + qname.Local, qname.Namespace + " " + qname.Local, qname.Raw}
	if isXSDNamespace(qname.Namespace) {
		keys = append(keys, "xs:"+qname.Local, "xsd:"+qname.Local, qname.Local)
	}
	for _, key := range keys {
		if value := options.TypeMappings[key]; value != "" {
			return mappedGoType(value, currentPackage, imports)
		}
	}
	return ""
}

func mappedGoType(value string, currentPackage string, imports map[string]ImportDecl) string {
	if idx := strings.LastIndex(value, "."); idx > 0 {
		prefix := value[:idx]
		typeName := value[idx+1:]
		if strings.Contains(prefix, "/") {
			path := prefix
			alias := importAlias(path)
			if alias == currentPackage {
				return typeName
			}
			imports[alias] = ImportDecl{Alias: alias, Path: path}
			return alias + "." + typeName
		}
		if prefix != currentPackage {
			imports[prefix] = ImportDecl{Path: prefix}
		}
		return prefix + "." + typeName
	}
	return value
}

func importAlias(path string) string {
	path = strings.TrimRight(path, "/")
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return goName(path[idx+1:])
	}
	return goName(path)
}

func isRuntimeHelperType(name string) bool {
	switch name {
	case "AnyElement", "AnyAttributes", "Nillable", "XSDID", "XSDIDREF", "XSDIDREFS", "XSDDateTime", "XSDDate", "XSDTime", "XSDDuration", "XSDGYear", "XSDGYearMonth", "XSDGMonth", "XSDGMonthDay", "XSDGDay", "XSDDecimal", "XSDHexBinary", "XSDBase64Binary":
		return true
	default:
		return false
	}
}

func runtimeType(name string, currentPackage string, options Options, imports map[string]ImportDecl) string {
	if options.RuntimePackage == "" || currentPackage == options.RuntimePackage {
		return name
	}
	imports[options.RuntimePackage] = ImportDecl{Alias: options.RuntimePackage, Path: importPathForPackage(options.RuntimePackage, options)}
	return options.RuntimePackage + "." + name
}

func qualifyIdentifier(name string, namespace string, currentPackage string, options Options, imports map[string]ImportDecl) string {
	if name == "" {
		return name
	}
	if isXSDNamespace(namespace) {
		if isRuntimeHelperType(name) {
			return runtimeType(name, currentPackage, options, imports)
		}
		return name
	}
	targetPackage := packageForNamespace(namespace, options)
	if targetPackage == "" || targetPackage == currentPackage {
		return name
	}
	imports[targetPackage] = ImportDecl{Alias: targetPackage, Path: importPathForPackage(targetPackage, options)}
	return targetPackage + "." + name
}

func packageForNamespace(namespace string, options Options) string {
	if pkg := options.NamespacePackages[namespace]; pkg != "" {
		return pkg
	}
	return options.Package
}

func importPathForPackage(packageName string, options Options) string {
	if options.ModulePath == "" {
		return packageName
	}
	return strings.TrimRight(options.ModulePath, "/") + "/" + packageName
}

func sortedImports(imports map[string]ImportDecl) []ImportDecl {
	keys := make([]string, 0, len(imports))
	for key := range imports {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]ImportDecl, 0, len(keys))
	for _, key := range keys {
		result = append(result, imports[key])
	}
	return result
}
