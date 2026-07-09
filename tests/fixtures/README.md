# Fixtures

Add reduced XSD and XML fixtures here. Each fixture is a minimal, self-contained
XSD that exercises one XSD construct and is checked against a committed snapshot
of the generated Go code.

## Layout

```text
tests/fixtures/<feature>/
  schema.xsd                  # entry schema (additional files via xs:include/xs:import)
  input.xml                   # optional XML sample for round-trip tests
  expected/model.go.golden    # committed snapshot of the generated models.go
  catalog.xml                 # optional OASIS XML catalog (auto-detected by snapshot test)
  bindings.xjb                # optional JAXB .xjb binding subset (auto-detected)
  optimizer_serde             # optional marker file to enable --optimizer-serde
```

## Snapshot tests

The full pipeline is run against every `schema.xsd` under `tests/fixtures/` and
the rendered output is compared byte-for-byte against the committed
`model.go.golden` file:

```bash
go test ./tests/snapshot
```

To regenerate snapshots after an intentional generator change:

```bash
go test ./tests/snapshot -update
```

The snapshot harness auto-detects optional fixture files:

- `catalog.xml` — passed as `--catalog`
- `bindings.xjb` — passed as `--binding`
- `optimizer_serde` — enables `--optimizer-serde`

The snapshot files use the `.golden` extension so they are not compiled as Go
packages by `go test ./...`.

## XML round-trip tests

Fixtures that ship an `input.xml` are additionally checked by a compile +
round-trip harness in `tests/roundtrip`:

```bash
go test ./tests/roundtrip
```

The harness renders the generated package into a temp Go module, writes a small
`Unmarshal -> Marshal -> Unmarshal` test that compares the two decoded values
with `reflect.DeepEqual`, and runs `go test` there. This validates that the
generated struct tags produce working encode/decode and that the generated code
compiles standalone.

To add round-trip coverage for a fixture, drop a sample XML at
`tests/fixtures/<feature>/input.xml`. The root Go type is assumed to be
`<RootElement>"Type"`.

## Current fixtures

| Fixture | Feature |
|---------|---------|
| `all` | `xs:all` group |
| `any` | `xs:any` / `xs:anyAttribute` wildcards |
| `any_namespace` | wildcard namespace constraints |
| `any_strict` | `processContents="strict"` with global declarations |
| `any_strict_local` | strict with local declarations |
| `any_type` | `xs:anyType` / `xs:anySimpleType` |
| `attribute_group` | `xs:attributeGroup` with nested refs |
| `binding` | JAXB `.xjb` class/property overrides |
| `catalog` | OASIS XML catalog resolution |
| `choice` | `xs:choice` safe pure |
| `choice_repeated` | repeated ordered `[]Content` choices |
| `complex_content_restriction` | complexContent restriction |
| `complex_type_with_group` | `xs:group` refs |
| `default_fixed` | default/fixed values on elements and attributes |
| `duplicate_types` | optimizer duplicate removal |
| `dynamic_type` | dynamic type dispatch |
| `dynamic_type_substitution_group` | substitution group polymorphism |
| `element_refs_with_ns` | cross-namespace element refs |
| `enumeration` | `xs:enumeration` constants |
| `extension_base` | complexContent extension |
| `extension_base_two_files` | multi-file extension |
| `extension_simple_content` | simpleContent extension |
| `facet_validation` | facet validation (length, pattern, etc.) |
| `facet_digits_whitespace` | totalDigits, fractionDigits, whiteSpace |
| `idref` | `xs:ID` / `xs:IDREF` / `xs:IDREFS` |
| `imported_type_element` | top-level element typed with imported complex type (legacy device-monitor stub) |
| `global_element_type_qname` | local element name vs global element QName (legacy WebRTC stub) |
| `imported_extension_base` | imported base type extension |
| `list_union` | `xs:list` / `xs:union` |
| `mixed_content` | `mixed="true"` order-preserving |
| `name_collision` | namespace/name collision handling |
| `nillable` | `xsi:nil` via `Nillable[T]` |
| `optimizer_enum_empty` | empty enum optimizer pass |
| `optimizer_enum_empty_variant` | empty enum variant removal |
| `optimizer_union_duplicate` | duplicate union variant removal |
| `optimizer_union_empty` | empty union collapse |
| `optimizer_union_flatten` | union flattening (serde) |
| `override` | `xs:override` declaration replacement |
| `package_mapping` | namespace-to-package mapping |
| `package_mapping_polymorphic` | cross-package polymorphism |
| `polymorphic_multi_field` | multiple polymorphic fields |
| `prefix_marshaling` | prefix-controlled marshaling |
| `presence_tracking` | required scalar presence tracking |
| `ref_to_attribute` | attribute refs |
| `redefine` | `xs:redefine` with self-derivation |
| `simple_content` | simpleContent |
| `simple_content_with_extension` | simpleContent with extension |
| `strict_validation` | strict field validation |
| `substitution_group` | substitution groups |
| `typedef_chain` | typedef chain resolution |
| `type_name_clash` | type name collision |
| `xsd_scalar_types` | date/time/duration/decimal/binary/doc comments |

## Adding a new fixture

1. Add the minimal XSD under `tests/fixtures/<feature>/schema.xsd`.
2. Generate the snapshot:
   ```bash
   go test ./tests/snapshot -update -run TestSnapshots/<feature>
   ```
3. Inspect `expected/model.go.golden` to confirm the generated shape.
4. Add focused assertions in the relevant `internal/<stage>` test if the
   construct needs semantic checks beyond the snapshot.
