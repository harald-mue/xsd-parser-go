# Workflow

This document is the operational plan for working on `xsd-parser-go`.

## Phase 0: Orientation

1. Read `README.md` and `AGENTS.md` in this repository.
2. Keep in mind: this project implements parsing, interpretation, optimization, and rendering in Go. It is inspired by [`xsd-parser`](https://github.com/Bergmann89/xsd-parser) but does not depend on it.

## Phase 1: Native Go parser scaffold

Goal: parse enough XSD in Go to inspect top-level declarations.

Commands:

```bash
go test ./...
go run ./cmd/xsd-parser-go inspect examples/minimal/schema.xsd
```

Expected result: schema statistics are printed without errors.

## Phase 2: Raw schema model expansion

Goal: parse core XSD structures into `internal/schema`.

Initial scope:

- `xs:schema`
- `targetNamespace`
- namespace declarations
- `xs:include`
- `xs:import`
- top-level `xs:element`
- top-level `xs:complexType`
- top-level `xs:simpleType`
- attributes
- sequences
- choices
- all groups
- occurrence constraints

## Phase 3: Semantic interpreter

Goal: convert raw schema syntax into a language-neutral Go-owned meta model.

The interpreter should resolve:

- QName references
- namespaces
- element/type identity
- built-in XSD types
- simple restrictions
- complex content extension
- substitution groups
- abstract types

## Phase 4: Optimizer

Goal: simplify and normalize the semantic model before Go lowering.

Candidate passes:

- resolve aliases/typedefs
- flatten trivial groups
- deduplicate structurally identical generated helper types
- normalize occurrence/cardinality
- simplify unions where safe
- prepare polymorphic dispatch metadata

## Phase 5: Go model generation

Goal: lower semantic meta types into Go-specific declarations.

Initial scope:

- structs for complex types
- type aliases for simple types
- constants for enumerations
- fields for attributes and elements
- optional/repeated field mapping
- import management

## Phase 6: Go XML runtime strategy

Goal: define generated helper code for cases `encoding/xml` cannot handle.

Runtime concepts:

- `QName`
- namespace stack/scope
- `xsi:type` extraction
- type registry
- unknown element preservation for `xs:any`

## Phase 7: Polymorphism support

Goal: support Java/JAXB-style schema designs.

Schema constructs:

- abstract complex types
- derived complex types via extension
- substitution groups
- `xsi:type` dispatch
- namespace-qualified element dispatch

Generated Go shape:

- interfaces for abstract/base concepts
- concrete structs for derived types
- generated registration tables
- custom `UnmarshalXML` for polymorphic fields
- custom `MarshalXML` where needed

## Daily development loop

```bash
go test ./...
go run ./cmd/xsd-parser-go inspect examples/minimal/schema.xsd
go run ./cmd/xsd-parser-go generate --package model --out generated/model examples/minimal/schema.xsd
go run ./cmd/xsd-parser-go convert examples/minimal/schema.xsd github.com/example/model generated/model
```
