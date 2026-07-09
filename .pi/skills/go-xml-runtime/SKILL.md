---
name: go-xml-runtime
description: Design or implement generated Go XML marshal/unmarshal support. Use for QName handling, namespace scopes, xsi:type dispatch, substitution groups, custom UnmarshalXML, custom MarshalXML, or runtime helper code.
---

# Go XML Runtime Skill

Use this skill when `encoding/xml` struct tags are not enough.

## Required context

Read:

- `docs/ARCHITECTURE.md`
- `docs/DESIGN_DECISIONS.md`
- `docs/IMPLEMENTATION_PLAN.md`

## When custom XML code is required

Generate explicit `MarshalXML` / `UnmarshalXML` methods for:

- `xsi:type` polymorphism
- abstract base types
- substitution groups
- repeated choices
- order-sensitive content models
- mixed content
- `xs:any` / `xs:anyType`
- namespace-dependent enum or QName values

## Core runtime concepts

The generated code will likely need these concepts:

```go
type QName struct {
    Namespace string
    Local     string
}
```

Potential helpers:

- namespace prefix scope tracking
- QName resolution from lexical values
- extraction of `xsi:type`
- registry lookup by QName
- unknown element capture

## Dispatch strategy

For polymorphic fields:

1. inspect the element QName
2. inspect `xsi:type` if present
3. resolve namespace prefixes
4. select a constructor from a generated registry
5. decode into the concrete type
6. assign it to an interface field or choice wrapper

## Design checklist

- Do not compare namespace prefixes as semantic identifiers.
- Compare expanded names: namespace URI plus local name.
- Preserve unknown content only when the schema permits it.
- Keep generated runtime code deterministic.
- Prefer self-contained generated helpers in early phases.
