# xsd-parser-go

Native Go implementation of an XML Schema Definition (XSD) parser, semantic interpreter, optimizer, and Go code generator.

Inspired by [`xsd-parser`](https://github.com/Bergmann89/xsd-parser).

## Goal

Generate useful, maintainable Go models and XML marshal/unmarshal code for complex XSD files, including schemas that were originally designed for Java/JAXB-style polymorphism.

Important target features:

- XSD parsing in Go
- schema include/import resolution
- semantic interpretation into a language-neutral Go-owned model
- optimization/simplification passes
- Go type lowering
- Go code rendering
- `xsi:type` polymorphism
- `abstract="true"` base types
- substitution groups
- complex type extension/restriction
- namespace-aware XML dispatch

## Current status

The generator supports a growing native Go XSD pipeline, including:

- include/import resolution
- complex/simple content extension and restriction
- groups, attribute groups, choices, all, wildcards, nillable values, list/union types, facets
- substitution groups and `xsi:type` polymorphism
- namespace package mapping and optional shared runtime package generation
- deterministic namespace-prefix-controlled marshaling
- generated validation with optional validate-on-unmarshal, fixed values, and common facets including digit/whitespace facets
- XSD scalar helper types for date/time/duration, exact decimals, and binary values
- required scalar presence tracking on generated unmarshal paths
- wildcard namespace/processContents validation with generated declaration indexes for `xs:any` / `xs:anyAttribute`
- default/fixed value handling on generated unmarshal/validation paths
- Go doc comments generated from `xs:documentation`
- typed `xs:ID` / `xs:IDREF` / `xs:IDREFS` helpers
- enum helper functions (`Values`, `Parse`, `IsValid`)
- OASIS XML catalog resolution, custom type mappings, and a small JAXB `.xjb` binding subset
- basic `xs:override` / `xs:redefine` support for element, attribute, attributeGroup, group, complexType, and simpleType declarations

## Quick start

```bash
go test ./...
go run ./cmd/xsd-parser-go inspect examples/minimal/schema.xsd
go run ./cmd/xsd-parser-go generate --package model --out generated/model examples/minimal/schema.xsd
```

Install the CLI with:

```bash
go install github.com/harald-mue/xsd-parser-go/cmd/xsd-parser-go@latest
```

## CLI usage

The CLI has three commands:

```bash
xsd-parser-go inspect <schema.xsd> [...]
xsd-parser-go generate [options] <schema.xsd> [...]
xsd-parser-go convert [options] <schema.xsd> [...] <module/path> <output/dir>
```

During local development you can replace `xsd-parser-go` with:

```bash
go run ./cmd/xsd-parser-go
```

### Inspect schemas

Use `inspect` for a quick parser/resolver smoke test and schema summary:

```bash
xsd-parser-go inspect examples/minimal/schema.xsd
```

Output includes the number of loaded schema files, namespaces, elements, complex types, and simple types.

### Recommended migration command: `convert`

`convert` is the legacy-friendly generation mode. It writes `models.go` by default, auto-detects Go package names from each schema `targetNamespace`, and emits additive compatibility aliases for older gocomply-style exported names when they differ from the new canonical names.

```bash
xsd-parser-go convert \
  schema.xsd \
  example.com/app/generated/model \
  generated/model
```

Arguments:

1. one or more XSD files
2. the Go import path of the generated package root
3. the output directory

For related schemas, pass all entry XSD files in one command so imports, duplicate names, and package mappings are resolved consistently:

```bash
xsd-parser-go convert \
  Command.xsd Reply.xsd Notification.xsd \
  example.com/app/pkg/common/messaging/v1/model \
  pkg/common/messaging/v1/model
```

When automatic namespace package names are not compatible with an existing public import path, override only the affected namespace:

```bash
xsd-parser-go convert \
  DeviceCommonTypes.xsd DeviceControlMessages.xsd DeviceMonitorMessages.xsd \
  --namespace-package=https://example.com/device/v7/monitor=notification \
  example.com/app/pkg/device/v7/model \
  pkg/device/v7/model
```

In this example, all namespaces are auto-detected except the monitor namespace, which is written to package `notification` for backwards compatibility.

### Explicit generation command: `generate`

`generate` is the explicit/modern mode. Use it when you want full control over package naming instead of auto-package detection.

Single-package output:

```bash
xsd-parser-go generate \
  --package model \
  --out generated/model \
  schema.xsd
```

Multi-package output by XML namespace:

```bash
xsd-parser-go generate \
  --module example.com/app/generated/model \
  --out generated/model \
  --namespace-package=urn:invoice=invoice \
  --namespace-package=urn:common=common \
  invoice.xsd common.xsd
```

This writes packages such as:

```text
generated/model/invoice/models.go
generated/model/common/models.go
```

Generated cross-package imports are based on `--module`, so it should match the import path of `--out` in your Go module.

## Generator options

### `--package name`

Default Go package name for schemas that are not mapped to a namespace package.

Default for `generate`:

```text
model
```

Example:

```bash
xsd-parser-go generate --package invoice --out generated/invoice schema.xsd
```

With `convert`, package names are usually derived from `targetNamespace`; use `--package` only as a fallback for schemas without a target namespace.

### `--out dir`

Output directory for generated files.

Default for `generate`:

```text
generated/model
```

Example:

```bash
xsd-parser-go generate --out internal/model schema.xsd
```

For `convert`, prefer the positional output directory:

```bash
xsd-parser-go convert schema.xsd example.com/app/internal/model internal/model
```

### `--module path`

Go import path for the generated package root. This is required when generated packages import each other, for example when using `--namespace-package`, `--runtime-package`, or `convert` auto-packages.

Example:

```bash
xsd-parser-go generate \
  --module example.com/app/generated/model \
  --namespace-package=urn:invoice=invoice \
  --namespace-package=urn:common=common \
  invoice.xsd common.xsd
```

For `convert`, prefer the positional module path:

```bash
xsd-parser-go convert invoice.xsd common.xsd example.com/app/generated/model generated/model
```

### `--namespace-package URI=pkg`

Maps an XML namespace URI to a Go package name. The flag is repeatable.

Example:

```bash
xsd-parser-go generate \
  --module example.com/app/model \
  --namespace-package=https://example.com/invoice=invoice \
  --namespace-package=https://example.com/common=common \
  schema.xsd
```

Use this flag when:

- using `generate` with multiple target namespaces
- preserving legacy package names
- overriding one auto-detected `convert` package name

`convert` auto-detects namespace packages by default, so most migrations do not need this flag.

### `--namespace-prefix URI=prefix`

Controls deterministic XML prefixes emitted by generated `MarshalXML` methods. This changes the preferred prefix only; XML namespace URIs remain the wire-compatible identity.

Example:

```bash
xsd-parser-go generate \
  --namespace-prefix=https://example.com/invoice=inv \
  --namespace-prefix=https://example.com/common=com \
  schema.xsd
```

### `--runtime-package pkg`

Emits shared XML runtime helpers into a generated package instead of duplicating helper code in each namespace package.

Example:

```bash
xsd-parser-go generate \
  --module example.com/app/model \
  --runtime-package=xsdxml \
  --namespace-package=https://example.com/invoice=invoice \
  --namespace-package=https://example.com/common=common \
  schema.xsd
```

This creates an additional package such as:

```text
generated/model/xsdxml/models.go
```

Use this for larger multi-package schemas to reduce duplicate helper code.

### `--output-file file`

Name of the generated Go file in each output package.

Default:

```text
models.go
```

Example:

```bash
xsd-parser-go generate \
  --output-file model_gen.go \
  --out generated/model \
  schema.xsd
```

### `--validate-on-unmarshal`

Calls generated `Validate` methods from generated `UnmarshalXML` methods where custom unmarshal code is generated.

Example:

```bash
xsd-parser-go generate \
  --validate-on-unmarshal \
  schema.xsd
```

This is useful for catching missing required values, invalid fixed values, invalid enums, and supported facets during XML decoding. Required scalar elements and attributes are presence-tracked, so missing values can be distinguished from Go zero-values.

For `convert`, boolean flags can be disabled explicitly if needed:

```bash
xsd-parser-go convert \
  --validate-on-unmarshal=false \
  schema.xsd example.com/app/model model
```

### `--optimizer-serde`

Enables serde-oriented optimizer passes, currently including union flattening and enum-union merging.

Example:

```bash
xsd-parser-go generate \
  --optimizer-serde \
  schema.xsd
```

Use this when you prefer simpler generated serialization shapes for supported union patterns.

### `--catalog file`

Adds an OASIS XML catalog for resolving `xs:include`, `xs:import`, `xs:override`, and `xs:redefine` schema locations. The flag is repeatable.

Example:

```bash
xsd-parser-go generate \
  --catalog catalog.xml \
  --catalog vendor/catalog.xml \
  schema.xsd
```

### `--binding file`

Applies the supported subset of JAXB `.xjb` bindings for class/property naming customizations. The flag is repeatable.

Example:

```bash
xsd-parser-go generate \
  --binding bindings.xjb \
  schema.xsd
```

### `--type-map QName=GoType`

Maps an XSD type to an existing Go type. The flag is repeatable.

Examples:

```bash
# Prefix form for built-in XML Schema types.
xsd-parser-go generate \
  --type-map=xs:dateTime=time.Time \
  schema.xsd

# Namespace-qualified form for schema-specific types.
xsd-parser-go generate \
  --type-map=https://example.com/common#Money=example.com/app/types.Money \
  schema.xsd
```

Use custom mappings when a domain type already exists and should be referenced instead of generated.

## Common workflows

### Generate one package from one schema

```bash
xsd-parser-go generate \
  --package model \
  --out generated/model \
  schema.xsd
```

### Generate multiple namespace packages explicitly

```bash
xsd-parser-go generate \
  --module example.com/app/generated/model \
  --out generated/model \
  --namespace-package=https://example.com/order=order \
  --namespace-package=https://example.com/common=common \
  order.xsd common.xsd
```

### Generate multiple namespace packages automatically

```bash
xsd-parser-go convert \
  order.xsd common.xsd \
  example.com/app/generated/model \
  generated/model
```

### Preserve one legacy package name while using auto-packages

```bash
xsd-parser-go convert \
  common.xsd control.xsd monitor.xsd \
  --namespace-package=https://example.com/device/v7/monitor=notification \
  example.com/app/pkg/device/v7/model \
  pkg/device/v7/model
```

### Generate with catalog, bindings, validation, prefixes, and custom type mappings

```bash
xsd-parser-go convert \
  --catalog catalog.xml \
  --binding bindings.xjb \
  --validate-on-unmarshal \
  --namespace-prefix=https://example.com/order=ord \
  --type-map=xs:dateTime=time.Time \
  order.xsd common.xsd \
  example.com/app/generated/model \
  generated/model
```

## Documentation

- [Workflow](docs/WORKFLOW.md)
- [Implementation plan](docs/IMPLEMENTATION_PLAN.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Design decisions](docs/DESIGN_DECISIONS.md)
- [Inspiration](docs/XSD_PARSER_REFERENCE.md)
