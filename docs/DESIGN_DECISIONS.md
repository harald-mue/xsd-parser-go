# Design Decisions

## Decision 1: Implement the full pipeline in Go

This project must own its XSD parser, interpreter, optimizer, Go lowering, and renderer.

It is inspired by [`xsd-parser`](https://github.com/Bergmann89/xsd-parser) but is a fully independent Go implementation.

## Decision 2: Keep staged architecture

Use a staged pipeline:

```text
raw schema model -> semantic meta model -> optimized meta model -> Go model -> rendered Go code
```

This keeps parsing, interpretation, optimization, and rendering testable independently.

## Decision 3: Use Go-owned intermediate models

Do not render directly from XML tokens. Define Go-native models that are appropriate for this implementation.

## Decision 4: Struct tags are an optimization, not the whole strategy

`encoding/xml` tags work for simple schemas but fail for many polymorphic and order-sensitive constructs. The generator should use tags when possible and generated methods when necessary.

## Decision 5: Polymorphism is a core requirement

The project exists mainly because Java/JAXB-style XSDs are difficult for existing Go generators. Support for `xsi:type`, abstract types, substitution groups, and extension hierarchies must be designed explicitly.

### CustomXML: mixed polymorphic + regular fields

When a `complexType` contains both polymorphic child elements (substitution groups or `xsi:type`) **and** regular child elements, `encoding/xml` struct tags alone are insufficient — the polymorphic dispatch requires a token-by-token `UnmarshalXML`. In this case the generator emits a custom `UnmarshalXML` that:

1. Checks each `StartElement` against the polymorphic registry first.
2. Falls through to a `switch` that decodes regular fields by namespace + local name.
3. Calls `d.Skip()` only for truly unknown elements.

The corresponding `MarshalXML` encodes regular fields (in XSD sequence order, before polymorphic fields) followed by the polymorphic children via registry dispatch.

### xsi:type namespace resolution

Go's `xml.Decoder` resolves element namespace prefixes using all in-scope `xmlns:*` declarations, but only includes the declarations from the **current element** in `StartElement.Attr`. If an `xsi:type` value such as `ns2:CITypeTargetSystem` uses a prefix declared on an **ancestor** element, the prefix cannot be resolved from `Attr` alone. As a fallback, `XSITypeQName` uses `start.Name.Space` (the element's own resolved namespace) when the prefix lookup fails. This is correct for the common case where the element and its `xsi:type` share the same namespace prefix.

### xsi:type marshalling — namespace-qualified value

When marshalling a polymorphic field with `xsi:type`, the type value must be **namespace-qualified** (e.g., `tns:DeploymentIdIdentifier`, not just `DeploymentIdIdentifier`). An unqualified `xsi:type` value is resolved against the default namespace (usually empty), causing JAXB and other strict XML parsers to fail with `InstantiationException` when they cannot locate the concrete type. The generated `MarshalXML` looks up `namespacePrefixes[entry.XSIType.Namespace]` and emits `prefix:Local` when a prefix is available.

## Decision 6: Fixture-driven implementation

Every feature should start with a minimal XSD fixture and, when applicable, XML examples. Real-world schemas should be reduced before being committed as test cases.

## Decision 7: Standard library first

Prefer Go's standard library for XML parsing, formatting, and CLI scaffolding until a clear need for third-party dependencies appears.

## Open decisions

### Integer mapping

Possible mappings:

- map XSD integers to native Go integer types when bounded
- map unbounded `xs:integer` to `math/big.Int`
- initially map difficult numeric types to `string`

### Date/time mapping

Possible mappings:

- `time.Time`
- custom date/time wrapper types
- `string` initially

### Decimal mapping

Possible mappings:

- `string`
- third-party decimal package
- generated wrapper type

### Runtime packaging

Options:

1. generate all helpers into each output package
2. provide a small reusable Go runtime module
3. support both modes

Default for early phases: generate self-contained helpers to reduce dependency friction.
