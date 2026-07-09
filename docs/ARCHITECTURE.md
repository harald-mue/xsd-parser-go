# Architecture

## Overview

`xsd-parser-go` is a native Go implementation of an XML Schema parser and Go code generator.

```text
XSD files + catalogs + bindings
  -> parser/resolver
  -> raw schema model
  -> semantic interpreter
  -> optimizer
  -> Go lowering
  -> Go source renderer
  -> generated .go files
```

Inspired by [`xsd-parser`](https://github.com/Bergmann89/xsd-parser). Fully independent Go implementation.

## Pipeline stages

### Parser

Package: `internal/parser`

Files: `parser.go` (core + utility), `elements.go`, `complex_types.go`, `simple_types.go`, `attributes.go`, `annotations.go`, `catalog.go`, `patches.go`

Responsibilities:

- XML tokenization/decoding
- namespace declaration capture
- QName parsing
- include/import resolution
- OASIS XML catalog resolution (`uri`, `system`, `rewriteURI`, `rewriteSystem`)
- `xs:override` / `xs:redefine` patching
- raw XSD syntax extraction (elements, attributes, attribute groups, complex/simple types, groups, wildcards, facets, default/fixed values, annotations)

Output model package: `internal/schema`

### Raw schema model

Package: `internal/schema`

Represents the XSD syntax tree including:

- schemas, includes, imports, overrides, redefines
- elements and attributes (with default/fixed/form)
- attribute groups and group refs
- complex/simple types
- complex/simple content extension and restriction
- sequences/choices/all
- wildcards (`xs:any`, `xs:anyAttribute`)
- facets and occurrence constraints
- annotations/documentation

### Interpreter

Package: `internal/interpreter`

Files: `model.go` (core interpretation), `polymorphism.go`, `declarations.go`

Responsibilities:

- resolve QName references
- assign stable identifiers
- interpret schema syntax into semantic type descriptions
- collect substitution groups and detect abstract/dynamic types
- interpret restrictions/extensions
- resolve attribute groups (including nested refs)
- build declaration index for strict wildcard validation
- carry `xs:documentation` text through the pipeline

### Optimizer

Package: `internal/optimizer`

Files: `optimizer.go`, `config.go`, `helpers.go`, `resolve_typedefs.go`, `remove_duplicates.go`, `passes_simple.go`, `passes_union.go`

Responsibilities:

- configurable optimizer flag system (`Flags`, `DefaultFlags`, `FlagSerde`)
- alias/typedef resolution
- duplicate type removal
- empty enum/variant removal
- union flattening and enum-union merging
- unrestricted base type simplification
- `xs:anyType` replacement

### Binding

Package: `internal/binding`

Parses a JAXB `.xjb` subset for class/property name overrides.

### Generator / Go lowering

Package: `internal/generator`

Files: `model.go` (types + `Generate`), `naming.go` (name registry, type qualification, imports), `fields.go` (field building, facets, enums), `scalars.go` (scalar mapping, type mapping, needs analysis)

Responsibilities:

- map semantic types to Go declarations (structs, aliases, interfaces)
- XSD scalar mapping (`xs:dateTime` → `XSDDateTime`, `xs:decimal` → `XSDDecimal`, `xs:hexBinary` → `XSDHexBinary`, etc.)
- custom type mapping via `--type-map`
- map fields and XML metadata
- prepare imports and polymorphic registries
- generate enum constants and facet declarations

### Renderer

Package: `internal/renderer`

Files: `render.go` (core orchestration), `runtime_helpers.go` (XSD scalars, QName, Nillable, Any, wildcard, prefix, registry), `struct_render.go` (struct/marshal/unmarshal/validation), `simple_render.go` (list/union/consts/enum/facets), `polymorphic_render.go` (mixed content/choice/custom XML)

Responsibilities:

- render deterministic, gofmt-compatible Go source
- emit runtime helper types (date/time/duration, decimal, binary, ID/IDREF, Nillable, AnyElement, QName)
- emit enum helpers (`Values`, `Parse`, `IsValid`)
- emit validation code (facets, fixed values, presence tracking, wildcard strict)
- emit prefix-controlled marshaling
- keep rendering separate from schema semantics

### Pipeline

Package: `internal/pipeline`

Orchestrates the full pipeline: parse → interpret → optimize → generate → render → write.

## Generated runtime types

When a schema uses XSD scalar types, the renderer emits helper types with `UnmarshalText`/`MarshalText` and conversion methods:

| XSD type | Go type | Methods |
|----------|---------|---------|
| `xs:dateTime` | `XSDDateTime` | `.Time() (time.Time, error)` |
| `xs:date` | `XSDDate` | `.Time()` |
| `xs:time` | `XSDTime` | `.Time()` |
| `xs:duration` | `XSDDuration` | regex validation |
| `xs:decimal` | `XSDDecimal` | `.Rat() (*big.Rat, error)` |
| `xs:hexBinary` | `XSDHexBinary` | `MustHexBinary()` |
| `xs:base64Binary` | `XSDBase64Binary` | `MustBase64Binary()` |
| `xs:ID` | `XSDID` | NCName validation |
| `xs:IDREF` | `XSDIDREF` | `.Resolve(map[XSDID]any)` |
| `xs:IDREFS` | `XSDIDREFS` | list of `XSDIDREF` |
| `xs:gYear` etc. | `XSDGYear` etc. | regex validation |

## CLI flags

```
--package              default Go package name
--out                  output directory
--module               Go module path for cross-package imports
--namespace-package    map XML namespace URI to Go package
--namespace-prefix     map XML namespace URI to XML prefix
--runtime-package      emit shared runtime helpers into a package
--output-file          generated Go file name (default: models.go)
--validate-on-unmarshal  call Validate from generated UnmarshalXML
--optimizer-serde      enable serde-oriented optimizer passes
--catalog              OASIS XML catalog file (repeatable)
--binding              JAXB .xjb binding subset (repeatable)
--type-map             XSD type to Go type mapping (repeatable)
```

The `convert` command accepts legacy argument order, writes `models.go` by default, and auto-detects namespace packages from schema `targetNamespace` values:

```bash
xsd-parser-go convert schema-a.xsd schema-b.xsd module/path output/dir
```

Use `--namespace-package=URI=pkg` only when the automatically derived package name must be overridden. Use a module path that points to the generated package root. For example, if the output directory is `pkg/webrtc/v1/model`, the module path should usually be `example.com/project/pkg/webrtc/v1/model`.

For migration compatibility the generator also emits additive legacy type aliases when an older gocomply-style exported name differs from the new Go name, for example `type Rgbcolor = RGBColor`. The improved names remain primary; aliases only preserve old public API references.

## Testing architecture

Fixture-driven tests:

```text
tests/fixtures/<feature>/
  schema.xsd               entry schema
  input.xml                optional XML for round-trip tests
  expected/model.go.golden committed snapshot
  catalog.xml              optional OASIS catalog
  bindings.xjb             optional JAXB binding file
  optimizer_serde          optional marker to enable serde optimizer
```

Test suites:

- `tests/snapshot` — golden file comparison for every fixture
- `tests/roundtrip` — Unmarshal → Marshal → Unmarshal DeepEqual
- `tests/validation` — field validation, facets, defaults, presence tracking
- `tests/enterprise` — ID/IDREF, catalog, type mapping, binding
- `tests/legacy_stubs` — legacy migration patterns and compile checks for ignored local stub copies
- `tests/real_schema` — optional compile + XML smoke for large schemas under `tests/real_schema/schemas/`
