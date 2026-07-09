# Agent Instructions for xsd-parser-go

## Project purpose

Build a native Go implementation of an XSD parser, interpreter, optimizer, and Go code generator.

The project is inspired by [`xsd-parser`](https://github.com/Bergmann89/xsd-parser) but is a fully independent Go implementation.

## Language and communication

- Write repository documentation, comments, plans, and skill files in English.
- Keep user-facing summaries concise.
- Prefer explicit file paths in responses.

## Development strategy

1. Implement the complete pipeline in Go:
   - parser
   - resolver/include/import handling
   - interpreter
   - optimizer
   - Go lowering
   - renderer
2. Keep generated Go code deterministic and gofmt-compatible.
3. Add a Go-owned intermediate schema model and semantic meta model.
4. Treat polymorphism as a first-class feature, not as a post-processing hack.
5. Add fixtures for every schema construct before implementing support.

## Commands

Use these commands from the repository root:

```bash
go test ./...
go run ./cmd/xsd-parser-go inspect examples/minimal/schema.xsd
go run ./cmd/xsd-parser-go generate --package model --out generated/model examples/minimal/schema.xsd
go run ./cmd/xsd-parser-go convert examples/minimal/schema.xsd github.com/example/model generated/model
```

## Coding conventions

- Keep CLI code in `cmd/xsd-parser-go/`.
- Keep implementation packages under `internal/`.
- Prefer small packages matching pipeline stages:
  - `internal/parser` (core, `complex_types.go`, `elements.go`, `simple_types.go`, `attributes.go`, `annotations.go`, `catalog.go`, `patches.go`)
  - `internal/schema`
  - `internal/interpreter`
  - `internal/optimizer`
  - `internal/generator` (`model.go`, `naming.go`, `fields.go`, `scalars.go`)
  - `internal/renderer` (`render.go`, `runtime_helpers.go`, `struct_render.go`, `simple_render.go`, `polymorphic_render.go`)
  - `internal/binding` (JAXB `.xjb` binding subset)
  - `internal/pipeline`
- Use standard library packages first.
- Avoid global mutable state except generated registries where appropriate.
- Preserve XML namespace information at every pipeline stage.

## Testing expectations

For each supported XSD feature, add:

- a minimal XSD fixture
- expected Go output or behavior
- at least one XML sample when serialization/deserialization is relevant

Prioritize real failing schemas after reducing them to minimal reproducible fixtures.

## Important design constraints

- Go's `encoding/xml` struct tags are not enough for Java/JAXB-style polymorphic schemas. For these cases, generate explicit decoder/encoder code with QName and `xsi:type` dispatch.
- Enterprise schemas can be resolved via OASIS XML catalogs (`--catalog`), customized via JAXB `.xjb` binding files (`--binding`), or remapped with `--type-map`.
- XSD scalar types (date/time, decimal, binary, ID/IDREF) have dedicated runtime helper types with validation and conversion methods.
