---
name: xsd-schema-triage
description: Analyze failing or complex XSD schemas for xsd-parser-go. Use when reducing real-world schemas, diagnosing unsupported constructs, or creating minimal fixtures.
---

# XSD Schema Triage Skill

Use this skill when a schema fails to parse, interpret, generate, compile, marshal, or unmarshal correctly.

## Goals

- Reduce real-world schemas to minimal fixtures.
- Identify which XSD feature is missing.
- Determine whether the failure belongs to parser, interpreter, optimizer, generator, renderer, or runtime XML code.
- Convert failures into regression tests.

## Triage process

1. Reproduce the failure with the original schema.
2. Run schema inspection:

```bash
go run ./cmd/xsd-parser-go inspect path/to/schema.xsd
```

3. If inspection fails, investigate the native Go parser.
4. If inspection succeeds, inspect interpreter/generator output.
5. Reduce the schema while preserving the failing behavior.
6. Store the reduced case under `tests/fixtures/<feature>/`.
7. Add XML examples when runtime behavior matters.

## Common Java/JAXB-style constructs to look for

- `abstract="true"`
- `xsi:type`
- `substitutionGroup`
- `complexContent` / `extension`
- deeply nested `choice` and `sequence`
- namespace-qualified references
- imported schemas with duplicate local names

## Fixture quality checklist

A fixture should be:

- as small as possible
- named after the feature it covers
- documented with the expected behavior
- paired with XML input/output samples when relevant
