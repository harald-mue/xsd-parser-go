# Implementation Plan

## Current status

The native Go pipeline now covers parser, resolver, interpreter, optimizer,
Go lowering, renderer, validation, polymorphic XML runtime support, namespace
package mapping, optional shared runtime helpers, strict wildcard declaration
indexes, presence tracking, attribute groups, default/fixed values, XSD scalar
helper types, ID/IDREF helpers, enum helpers, XML catalog resolution, custom
type mappings, a small JAXB binding subset, generated Go doc comments, common
facets, legacy-friendly `convert` generation, automatic namespace package detection,
configurable generated file names, additive legacy compatibility aliases, and
real-schema compile/XML smoke regressions.

Remaining work is mostly deeper XSD conformance and additional real-world XML
regression coverage: identity constraints (`xs:key`, `xs:keyref`, `xs:unique`),
full XSD 1.1 override/redefine rule validation, complete `xs:all` semantic
validation, wildcard slot-order preservation, and more schema-specific XML
roundtrips.

## Guiding principle

Implement the full XSD processing pipeline in Go. The project is inspired by [`xsd-parser`](https://github.com/Bergmann89/xsd-parser) but is a fully independent implementation.

```text
XSD files
  -> Go parser / resolver
  -> Go raw schema model
  -> Go semantic interpreter
  -> Go optimizer
  -> Go lowering
  -> Go renderer
  -> generated .go files
```

## Milestone 1: Project scaffold

Status: done.

Deliverables:

- Go module
- CLI with `inspect` and `generate`
- minimal native parser for top-level declarations
- placeholder pipeline stages
- documentation and pi skills
- minimal XSD fixture

Acceptance criteria:

```bash
go test ./...
go run ./cmd/xsd-parser-go inspect examples/minimal/schema.xsd
```

## Milestone 2: Raw XSD parser

Status: done.

Deliverables:

- robust XML decoder wrapper
- namespace declaration capture
- QName parsing and resolution primitives
- schema location resolution
- `xs:include`
- `xs:import`
- top-level declarations
- nested complex type content
- nested simple type content
- occurrence constraints
- annotations/documentation capture when useful

Parser packages:

- `internal/parser` for XML decoding and resolver orchestration
- `internal/schema` for raw XSD syntax model

Acceptance criteria:

- parse minimal fixtures without losing namespace information
- parse feature fixtures under `tests/fixtures/`

## Milestone 3: Semantic meta model

Status: done.

Deliverables:

- Go-owned semantic type model
- stable identifiers for schema/types/nodes/properties
- QName/type reference resolver
- built-in XSD type table
- representation for:
  - simple types
  - complex types
  - enumerations
  - unions
  - lists
  - references
  - groups
  - dynamic/polymorphic types

## Milestone 4: Interpreter

Status: done.

Deliverables:

- raw schema to meta model conversion
- namespace-aware reference resolution
- include/import handling semantics
- complex content extension/restriction
- simple restrictions/facets
- substitution group collection
- abstract type metadata

Acceptance criteria:

- fixtures can be interpreted without relying on generated Go output
- model dumps are deterministic enough for debugging

## Milestone 5: Optimizer

Status: done.

Deliverables:

- pass framework
- alias/typedef resolution
- duplicate removal
- group flattening where safe
- union simplification where safe
- dynamic type metadata normalization
- cardinality normalization

## Milestone 6: Go intermediate model

Status: done.

Deliverables:

- `Package`
- `File`
- `ImportSet`
- `TypeDecl`
- `Struct`
- `Field`
- `Interface`
- `Method`
- enum-like constants
- XML metadata on fields/types
- polymorphic registry declarations

Design constraints:

- The model must be independent of text rendering.
- Field metadata must preserve XML name, namespace, cardinality, and attributes.
- Polymorphic fields must be representable before rendering.

## Milestone 7: Basic Go renderer

Status: done.

Deliverables:

- deterministic `.go` output
- gofmt-compatible rendering
- import block rendering
- struct rendering
- scalar type aliases
- enum-like constants
- XML struct tags where sufficient

Acceptance criteria:

- generated code compiles for simple fixtures

## Milestone 8: Basic Go lowering

Status: done.

Deliverables:

- built-in XSD scalar mapping
- complex type to struct lowering
- simple type to alias lowering
- attributes to Go fields
- sequence elements to Go fields
- optional/repeated cardinality mapping

## Milestone 9: XML decoding strategy

Status: done.

Deliverables:

- document when `encoding/xml` tags are enough
- implement generated helper functions for QName and `xsi:type`
- generate custom `UnmarshalXML` for choices and polymorphic fields
- generate custom `MarshalXML` where necessary

## Milestone 10: Polymorphism

Status: done.

Deliverables:

- abstract/base type detection
- derived type registration
- substitution group registry
- `xsi:type` dispatch
- generated interface marker methods

Example target shape:

```go
type Animal interface { isAnimal() }

type Dog struct { Name string }
func (*Dog) isAnimal() {}

var animalTypeRegistry = map[QName]func() Animal{
    {Namespace: "urn:test", Local: "Dog"}: func() Animal { return &Dog{} },
}
```

## Milestone 11: Real schema hardening

Status: in progress (optional compile + XML smoke for large real-world schemas under `tests/real_schema/schemas/`).

Deliverables:

- anonymized real-world fixture set
- regression tests for every fixed edge case
- documented limitations

## Milestone 12: Packaging

Status: done (CLI stable, `go:generate` ready, optional runtime package, legacy-friendly `convert` command).

Deliverables:

- stable CLI name
- release build instructions
- `go:generate` usage documentation
- optional generated runtime package strategy
