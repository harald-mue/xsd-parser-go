---
name: xsd-go-backend
description: Implement or modify the native Go XSD pipeline and Go code generator. Use when working on parser, schema model, interpreter, optimizer, Go lowering, rendering, naming, scalar mapping, or generated Go code structure.
---

# XSD Go Backend Skill

Use this skill when implementing the core native Go generator.

## Required reading

Before changing parser, lowering, or rendering, read:

- `AGENTS.md`
- `docs/ARCHITECTURE.md`
- `docs/IMPLEMENTATION_PLAN.md`

## Workflow

1. Identify the XSD construct being implemented.
2. Add or select a minimal fixture.
3. Extend the native Go parser/schema model if needed.
4. Extend the Go semantic meta model if needed.
5. Implement interpretation and optimization.
6. Implement Go lowering.
7. Render deterministic Go code.
8. Add tests or expected output.
9. Run:

```bash
go test ./...
go run ./cmd/xsd-parser-go inspect examples/minimal/schema.xsd
```

## Design rules

- Do not lower directly to strings when a typed Go model is needed.
- Preserve XML namespace and cardinality metadata.
- Prefer simple `encoding/xml` tags only for simple cases.
- Use custom marshal/unmarshal for polymorphic or order-sensitive cases.

## Output quality checklist

Generated Go should be:

- deterministic
- gofmt-compatible
- namespace-aware
- explicit about optional/repeated fields
- suitable for snapshot tests
