package binding

import (
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
)

// Options contains the supported JAXB binding subset.
type Options struct {
	TypeNameOverrides  map[string]string
	FieldNameOverrides map[string]string
}

// LoadFiles parses a small JAXB .xjb subset:
//
//	<jxb:bindings node="//xs:complexType[@name='Foo']"><jxb:class name="Bar"/></jxb:bindings>
//	<jxb:bindings node="//xs:element[@name='foo']"><jxb:property name="Bar"/></jxb:bindings>
//
// Keys are matched by local XSD name across namespaces.
func LoadFiles(paths []string) (Options, error) {
	out := Options{TypeNameOverrides: map[string]string{}, FieldNameOverrides: map[string]string{}}
	for _, path := range paths {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return out, fmt.Errorf("read binding %s: %w", path, err)
		}
		var doc node
		if err := xml.Unmarshal(data, &doc); err != nil {
			return out, fmt.Errorf("parse binding %s: %w", path, err)
		}
		walk(doc, &out)
	}
	return out, nil
}

type node struct {
	XMLName  xml.Name
	Node     string `xml:"node,attr"`
	Name     string `xml:"name,attr"`
	Children []node `xml:",any"`
}

var namePattern = regexp.MustCompile(`@name=['"]([^'"]+)['"]`)

func walk(n node, out *Options) {
	local := nodeLocalName(n.Node)
	if local != "" {
		for _, child := range n.Children {
			switch child.XMLName.Local {
			case "class", "typesafeEnumClass":
				if child.Name != "" {
					out.TypeNameOverrides[local] = child.Name
				}
			case "property":
				if child.Name != "" {
					out.FieldNameOverrides[local] = child.Name
				}
			}
		}
	}
	for _, child := range n.Children {
		walk(child, out)
	}
}

func nodeLocalName(expr string) string {
	m := namePattern.FindStringSubmatch(expr)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}
