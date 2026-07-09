package optimizer_test

import (
	"testing"

	"github.com/harald-mue/xsd-parser-go/internal/interpreter"
	"github.com/harald-mue/xsd-parser-go/internal/optimizer"
	"github.com/harald-mue/xsd-parser-go/internal/parser"
	"github.com/harald-mue/xsd-parser-go/internal/schema"
)

func TestResolveTypedefsFollowsSimpleTypeChain(t *testing.T) {
	meta := interpreter.MetaTypes{
		Types: []interpreter.MetaType{
			{
				Kind: "simpleType", Name: "Base", Namespace: "http://example.com",
				Alias: &interpreter.MetaAlias{BaseRef: "xs:string", BaseName: schema.QName{Local: "string", Namespace: "http://www.w3.org/2001/XMLSchema"}},
			},
			{
				Kind: "simpleType", Name: "Mid", Namespace: "http://example.com",
				Alias: &interpreter.MetaAlias{BaseRef: "tns:Base", BaseName: schema.QName{Local: "Base", Namespace: "http://example.com"}},
			},
			{
				Kind: "simpleType", Name: "Leaf", Namespace: "http://example.com",
				Alias: &interpreter.MetaAlias{BaseRef: "tns:Mid", BaseName: schema.QName{Local: "Mid", Namespace: "http://example.com"}},
			},
			{
				Kind: "complexType", Name: "Holder", Namespace: "http://example.com",
				Fields: []interpreter.MetaField{{
					Name: "Value", TypeName: schema.QName{Local: "Leaf", Namespace: "http://example.com"},
				}},
			},
		},
	}
	out, err := optimizer.OptimizeWithFlags(meta, optimizer.DefaultFlags()&^optimizer.FlagUseUnrestrictedBaseSimple)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	leaf := out.Types[2]
	if leaf.Alias == nil || leaf.Alias.BaseName.Local != "Base" {
		t.Fatalf("Leaf base = %#v, want Base", leaf.Alias)
	}
	holder := out.Types[3]
	if holder.Fields[0].TypeName.Local != "Base" {
		t.Fatalf("Holder field type = %#v, want Base", holder.Fields[0].TypeName)
	}
}

func TestResolveTypedefsPreservesChardataContentTypes(t *testing.T) {
	meta := interpreter.MetaTypes{
		Types: []interpreter.MetaType{
			{
				Kind: "simpleType", Name: "CustomString", Namespace: "http://example.com",
				Alias: &interpreter.MetaAlias{BaseRef: "xs:string", BaseName: schema.QName{Local: "string", Namespace: "http://www.w3.org/2001/XMLSchema"}},
			},
			{
				Kind: "simpleType", Name: "UnitNameTypeContentType", Namespace: "http://example.com",
				Alias: &interpreter.MetaAlias{BaseRef: "tns:CustomString", BaseName: schema.QName{Local: "CustomString", Namespace: "http://example.com"}},
			},
			{
				Kind: "complexType", Name: "UnitNameType", Namespace: "http://example.com",
				Fields: []interpreter.MetaField{{
					Name: "Content", Chardata: true,
					TypeName: schema.QName{Local: "UnitNameTypeContentType", Namespace: "http://example.com"},
				}},
			},
		},
	}
	out, err := optimizer.OptimizeWithFlags(meta, optimizer.DefaultFlags()&^optimizer.FlagUseUnrestrictedBaseSimple)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	unit := out.Types[2]
	if unit.Fields[0].TypeName.Local != "UnitNameTypeContentType" {
		t.Fatalf("chardata content type = %#v, want UnitNameTypeContentType", unit.Fields[0].TypeName)
	}
}

func TestRemoveDuplicatesMarksSecondComplexType(t *testing.T) {
	field := interpreter.MetaField{
		Name: "Id", XMLName: "id", Attribute: true, Required: true,
		TypeRef: "xs:int", TypeName: schema.QName{Local: "int", Namespace: "http://www.w3.org/2001/XMLSchema"},
	}
	meta := interpreter.MetaTypes{
		Types: []interpreter.MetaType{
			{Kind: "complexType", Name: "First", Namespace: "http://example.com", Fields: []interpreter.MetaField{field}},
			{Kind: "complexType", Name: "Second", Namespace: "http://example.com", Fields: []interpreter.MetaField{field}},
		},
	}
	out, err := optimizer.OptimizeWithFlags(meta, optimizer.DefaultFlags()&^optimizer.FlagUseUnrestrictedBaseSimple)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	second := out.Types[1]
	if second.DuplicateOf.Local != "First" {
		t.Fatalf("Second.DuplicateOf = %#v, want First", second.DuplicateOf)
	}
	if len(second.Fields) != 0 {
		t.Fatalf("duplicate should have cleared fields, got %d", len(second.Fields))
	}
}

func TestRemoveDuplicatesDoesNotMergeDifferentTypes(t *testing.T) {
	meta := interpreter.MetaTypes{
		Types: []interpreter.MetaType{
			{
				Kind: "complexType", Name: "A", Namespace: "http://example.com",
				Fields: []interpreter.MetaField{{Name: "X", TypeName: schema.QName{Local: "string", Namespace: "http://www.w3.org/2001/XMLSchema"}}},
			},
			{
				Kind: "complexType", Name: "B", Namespace: "http://example.com",
				Fields: []interpreter.MetaField{{Name: "Y", TypeName: schema.QName{Local: "string", Namespace: "http://www.w3.org/2001/XMLSchema"}}},
			},
		},
	}
	out, err := optimizer.OptimizeWithFlags(meta, optimizer.DefaultFlags()&^optimizer.FlagUseUnrestrictedBaseSimple)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	if out.Types[1].DuplicateOf.Local != "" {
		t.Fatalf("different types should not be merged")
	}
}

func TestRemoveEmptyEnumVariantsDropsEmptyValue(t *testing.T) {
	meta := interpreter.MetaTypes{
		Types: []interpreter.MetaType{{
			Kind: "simpleType", Name: "MyEnum", Namespace: "http://example.com",
			Alias: &interpreter.MetaAlias{
				BaseName: schema.QName{Local: "string", Namespace: "http://www.w3.org/2001/XMLSchema"},
				Facets: []schema.Facet{
					{Name: "enumeration", Value: ""},
					{Name: "enumeration", Value: "A"},
				},
			},
		}},
	}
	out, err := optimizer.Optimize(meta)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	facets := out.Types[0].Alias.Facets
	if len(facets) != 1 || facets[0].Value != "A" {
		t.Fatalf("facets = %#v, want only A", facets)
	}
}

func TestRemoveEmptyEnumsCollapsesEnumWithoutVariants(t *testing.T) {
	meta := interpreter.MetaTypes{
		Types: []interpreter.MetaType{{
			Kind: "simpleType", Name: "MyEnum", Namespace: "http://example.com",
			Alias: &interpreter.MetaAlias{
				BaseName: schema.QName{Local: "string", Namespace: "http://www.w3.org/2001/XMLSchema"},
				Facets:   []schema.Facet{{Name: "enumeration", Value: ""}},
			},
		}},
	}
	out, err := optimizer.Optimize(meta)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	if out.Types[0].DuplicateOf.Local != "string" {
		t.Fatalf("DuplicateOf = %#v, want xs:string", out.Types[0].DuplicateOf)
	}
}

func TestRemoveDuplicateUnionVariants(t *testing.T) {
	meta := interpreter.MetaTypes{
		Types: []interpreter.MetaType{{
			Kind: "simpleType", Name: "MyUnion", Namespace: "http://example.com",
			Union: &interpreter.MetaUnion{
				MemberNames: []schema.QName{
					{Local: "language", Namespace: "http://www.w3.org/2001/XMLSchema"},
					{Local: "string", Namespace: "http://www.w3.org/2001/XMLSchema"},
				},
			},
		}},
	}
	out, err := optimizer.OptimizeWithFlags(meta, optimizer.FlagRemoveDuplicateUnionVariants)
	if err != nil {
		t.Fatalf("OptimizeWithFlags: %v", err)
	}
	if len(out.Types[0].Union.MemberNames) != 1 {
		t.Fatalf("members = %#v, want one scalar bucket", out.Types[0].Union.MemberNames)
	}
}

func TestRemoveEmptyUnionCollapsesSingleMember(t *testing.T) {
	meta := interpreter.MetaTypes{
		Types: []interpreter.MetaType{{
			Kind: "simpleType", Name: "MyUnion", Namespace: "http://example.com",
			Union: &interpreter.MetaUnion{
				MemberNames: []schema.QName{
					{Local: "string", Namespace: "http://www.w3.org/2001/XMLSchema"},
				},
			},
		}},
	}
	out, err := optimizer.Optimize(meta)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	if out.Types[0].DuplicateOf.Local != "string" {
		t.Fatalf("DuplicateOf = %#v, want xs:string", out.Types[0].DuplicateOf)
	}
}

func TestFlattenUnionsFromFixture(t *testing.T) {
	schemas, err := parser.ParseFiles([]string{"testdata/optimizer_union_flatten/schema.xsd"})
	if err != nil {
		t.Fatalf("ParseFiles: %v", err)
	}
	meta, err := interpreter.Interpret(schemas)
	if err != nil {
		t.Fatalf("Interpret: %v", err)
	}
	out, err := optimizer.OptimizeWithFlags(meta, optimizer.DefaultFlags()|optimizer.FlagFlattenUnions)
	if err != nil {
		t.Fatalf("OptimizeWithFlags: %v", err)
	}
	union := findType(out.Types, "MyUnion")
	if union == nil || union.Union == nil {
		t.Fatal("MyUnion not found")
	}
	if len(union.Union.MemberNames) < 3 {
		t.Fatalf("flattened members = %d, want at least 3", len(union.Union.MemberNames))
	}
}

func findType(types []interpreter.MetaType, name string) *interpreter.MetaType {
	for i := range types {
		if types[i].Name == name {
			return &types[i]
		}
	}
	return nil
}
