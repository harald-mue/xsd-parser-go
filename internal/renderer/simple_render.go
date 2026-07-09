package renderer

import (
	"bytes"
	"fmt"

	"github.com/harald-mue/xsd-parser-go/internal/generator"
)

func renderListType(buf *bytes.Buffer, typ generator.TypeDecl) {
	itemType := typ.List.ItemType
	fmt.Fprintf(buf, "type %s []%s\n\n", typ.Name, itemType)
	fmt.Fprintf(buf, "func (v *%s) UnmarshalText(text []byte) error {\n", typ.Name)
	fmt.Fprintln(buf, "\tparts := strings.Fields(string(text))")
	fmt.Fprintf(buf, "\titems := make([]%s, 0, len(parts))\n", itemType)
	fmt.Fprintln(buf, "\tfor _, part := range parts {")
	if itemType == "string" {
		fmt.Fprintln(buf, "\t\titems = append(items, part)")
	} else {
		fmt.Fprintf(buf, "\t\tvar item %s\n", itemType)
		fmt.Fprintln(buf, "\t\tif _, err := fmt.Sscan(part, &item); err != nil {")
		fmt.Fprintln(buf, "\t\t\treturn err")
		fmt.Fprintln(buf, "\t\t}")
		fmt.Fprintln(buf, "\t\titems = append(items, item)")
	}
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintln(buf, "\t*v = items")
	fmt.Fprintln(buf, "\treturn nil")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
	fmt.Fprintf(buf, "func (v %s) MarshalText() ([]byte, error) {\n", typ.Name)
	fmt.Fprintln(buf, "\tparts := make([]string, 0, len(v))")
	fmt.Fprintln(buf, "\tfor _, item := range v {")
	fmt.Fprintln(buf, "\t\tparts = append(parts, fmt.Sprint(item))")
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintln(buf, "\treturn []byte(strings.Join(parts, \" \")), nil")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func renderUnionValidate(buf *bytes.Buffer, file generator.File, typ generator.TypeDecl) {
	if typ.Union == nil || len(typ.Union.MemberTypes) == 0 {
		return
	}
	enums := enumFacets(typ.Facets)
	fmt.Fprintf(buf, "func (v %s) Validate() error {\n", typ.Name)
	if len(enums) > 0 {
		fmt.Fprintln(buf, "\tswitch v {")
		for _, value := range enums {
			fmt.Fprintf(buf, "\tcase %s(%q):\n", typ.Name, value)
			fmt.Fprintln(buf, "\t\treturn nil")
		}
		fmt.Fprintln(buf, "\tdefault:")
		fmt.Fprintln(buf, "\t}")
	}
	fmt.Fprintln(buf, "\tvalue := string(v)")
	hasString := false
	for _, member := range typ.Union.MemberTypes {
		if member == "string" {
			hasString = true
			continue
		}
		if isScannableScalar(member) {
			fmt.Fprintf(buf, "\t{ var candidate %s; if _, err := fmt.Sscan(value, &candidate); err == nil { return nil } }\n", member)
			continue
		}
		fmt.Fprintf(buf, "\tif candidate, ok := any(%s(value)).(%s); ok {\n", member, validatableType(file))
		fmt.Fprintln(buf, "\t\tif err := candidate.Validate(); err == nil {")
		fmt.Fprintln(buf, "\t\t\treturn nil")
		fmt.Fprintln(buf, "\t\t}")
		fmt.Fprintln(buf, "\t}")
	}
	if hasString {
		fmt.Fprintln(buf, "\t_ = value")
		fmt.Fprintln(buf, "\treturn nil")
	} else {
		fmt.Fprintln(buf, "\t_ = value")
		fmt.Fprintf(buf, "\treturn fmt.Errorf(\"type %s does not match any union member\")\n", typ.Name)
	}
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func isScannableScalar(name string) bool {
	switch name {
	case "bool", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return true
	default:
		return false
	}
}

func renderConsts(buf *bytes.Buffer, consts []generator.ConstDecl) {
	if len(consts) == 0 {
		return
	}
	fmt.Fprintln(buf, "const (")
	for _, c := range consts {
		fmt.Fprintf(buf, "\t%s %s = %q\n", c.Name, c.Type, c.Value)
	}
	fmt.Fprintln(buf, ")")
	fmt.Fprintln(buf)
}

func renderEnumHelpers(buf *bytes.Buffer, typ generator.TypeDecl) {
	if len(typ.Consts) == 0 {
		return
	}
	fmt.Fprintf(buf, "func %sValues() []%s {\n", typ.Name, typ.Name)
	fmt.Fprintf(buf, "\treturn []%s{\n", typ.Name)
	for _, c := range typ.Consts {
		if c.Legacy {
			continue
		}
		fmt.Fprintf(buf, "\t\t%s,\n", c.Name)
	}
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
	fmt.Fprintf(buf, "func Parse%s(value string) (%s, error) {\n", typ.Name, typ.Name)
	fmt.Fprintf(buf, "\tv := %s(value)\n", typ.Name)
	fmt.Fprintln(buf, "\tif v.IsValid() {")
	fmt.Fprintln(buf, "\t\treturn v, nil")
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintf(buf, "\tvar zero %s\n", typ.Name)
	fmt.Fprintf(buf, "\treturn zero, fmt.Errorf(\"invalid %s %%q\", value)\n", typ.Name)
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
	fmt.Fprintf(buf, "func (v %s) IsValid() bool {\n", typ.Name)
	fmt.Fprintln(buf, "\tswitch v {")
	for _, c := range typ.Consts {
		if c.Legacy {
			continue
		}
		fmt.Fprintf(buf, "\tcase %s:\n", c.Name)
		fmt.Fprintln(buf, "\t\treturn true")
	}
	fmt.Fprintln(buf, "\tdefault:")
	fmt.Fprintln(buf, "\t\treturn false")
	fmt.Fprintln(buf, "\t}")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func renderFacetValidationHelpers(buf *bytes.Buffer) {
	fmt.Fprint(buf, `func digitCount(value string) int {
	count := 0
	for _, r := range value {
		if r >= '0' && r <= '9' {
			count++
		}
	}
	return count
}

func fractionDigitCount(value string) int {
	if i := strings.IndexAny(value, "."); i >= 0 {
		fraction := value[i+1:]
		if e := strings.IndexAny(fraction, "eE"); e >= 0 {
			fraction = fraction[:e]
		}
		return digitCount(fraction)
	}
	return 0
}

`)
}

func renderWhiteSpaceValidation(buf *bytes.Buffer, typeName, value string) {
	switch value {
	case "collapse":
		fmt.Fprintf(buf, "\tif strings.Join(strings.Fields(string(v)), \" \") != string(v) {\n")
		fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s whitespace must be collapsed\")\n", typeName)
		fmt.Fprintln(buf, "\t}")
	case "replace":
		fmt.Fprintln(buf, "\tif strings.ContainsAny(string(v), \"\\t\\n\\r\") {")
		fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s whitespace must not contain tabs or line breaks\")\n", typeName)
		fmt.Fprintln(buf, "\t}")
	}
}

func renderValidate(buf *bytes.Buffer, typ generator.TypeDecl) {
	if len(typ.Facets) == 0 {
		return
	}
	if typ.Union != nil && len(typ.Union.MemberTypes) > 0 {
		return
	}
	fmt.Fprintf(buf, "func (v %s) Validate() error {\n", typ.Name)
	for _, facet := range typ.Facets {
		switch facet.Name {
		case "enumeration":
			// handled below as one compact switch
		case "length":
			fmt.Fprintf(buf, "\tif len(string(v)) != %s {\n", facet.Value)
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s length must be %s\")\n", typ.Name, facet.Value)
			fmt.Fprintln(buf, "\t}")
		case "minLength":
			fmt.Fprintf(buf, "\tif len(string(v)) < %s {\n", facet.Value)
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s length must be >= %s\")\n", typ.Name, facet.Value)
			fmt.Fprintln(buf, "\t}")
		case "maxLength":
			fmt.Fprintf(buf, "\tif len(string(v)) > %s {\n", facet.Value)
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s length must be <= %s\")\n", typ.Name, facet.Value)
			fmt.Fprintln(buf, "\t}")
		case "pattern":
			fmt.Fprintf(buf, "\tif ok, err := regexp.MatchString(%q, string(v)); err != nil {\n", facet.Value)
			fmt.Fprintln(buf, "\t\treturn err")
			fmt.Fprintln(buf, "\t} else if !ok {")
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s does not match required pattern\")\n", typ.Name)
			fmt.Fprintln(buf, "\t}")
		case "whiteSpace":
			renderWhiteSpaceValidation(buf, typ.Name, facet.Value)
		case "totalDigits":
			fmt.Fprintln(buf, "\t_digits := digitCount(fmt.Sprint(v))")
			fmt.Fprintf(buf, "\tif _digits > %s {\n", facet.Value)
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s totalDigits must be <= %s\")\n", typ.Name, facet.Value)
			fmt.Fprintln(buf, "\t}")
		case "fractionDigits":
			fmt.Fprintln(buf, "\t_fractionDigits := fractionDigitCount(fmt.Sprint(v))")
			fmt.Fprintf(buf, "\tif _fractionDigits > %s {\n", facet.Value)
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s fractionDigits must be <= %s\")\n", typ.Name, facet.Value)
			fmt.Fprintln(buf, "\t}")
		case "minInclusive":
			fmt.Fprintf(buf, "\tif v < %s(%s) {\n", typ.Name, facet.Value)
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s must be >= %s\")\n", typ.Name, facet.Value)
			fmt.Fprintln(buf, "\t}")
		case "maxInclusive":
			fmt.Fprintf(buf, "\tif v > %s(%s) {\n", typ.Name, facet.Value)
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s must be <= %s\")\n", typ.Name, facet.Value)
			fmt.Fprintln(buf, "\t}")
		case "minExclusive":
			fmt.Fprintf(buf, "\tif v <= %s(%s) {\n", typ.Name, facet.Value)
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s must be > %s\")\n", typ.Name, facet.Value)
			fmt.Fprintln(buf, "\t}")
		case "maxExclusive":
			fmt.Fprintf(buf, "\tif v >= %s(%s) {\n", typ.Name, facet.Value)
			fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s must be < %s\")\n", typ.Name, facet.Value)
			fmt.Fprintln(buf, "\t}")
		}
	}
	enums := enumFacets(typ.Facets)
	if len(enums) > 0 {
		fmt.Fprintln(buf, "\tswitch v {")
		for _, value := range enums {
			fmt.Fprintf(buf, "\tcase %s(%q):\n", typ.Name, value)
		}
		fmt.Fprintln(buf, "\tdefault:")
		fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"type %s has invalid enumeration value %%v\", v)\n", typ.Name)
		fmt.Fprintln(buf, "\t}")
	}
	fmt.Fprintln(buf, "\treturn nil")
	fmt.Fprintln(buf, "}")
	fmt.Fprintln(buf)
}

func enumFacets(facets []generator.FacetDecl) []string {
	var values []string
	seen := make(map[string]bool)
	for _, facet := range facets {
		if facet.Name == "enumeration" && !seen[facet.Value] {
			seen[facet.Value] = true
			values = append(values, facet.Value)
		}
	}
	return values
}
