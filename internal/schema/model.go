package schema

// Schemas is the parsed representation of one or more XSD files.
type Schemas struct {
	Files []File
}

// QName is a namespace-aware XML qualified name.
//
// Raw keeps the lexical value as it appeared in the XSD attribute, Prefix keeps
// the lexical prefix when present, and Namespace is resolved from the in-scope
// namespace declarations when possible.
type QName struct {
	Raw       string
	Prefix    string
	Local     string
	Namespace string
}

// File contains the declarations discovered in one XSD document.
type File struct {
	Path                 string
	TargetNamespace      string
	ElementFormDefault   string
	AttributeFormDefault string
	Namespaces           map[string]string
	Includes             []Include
	Imports              []Import
	Overrides            []Override
	Redefines            []Redefine
	Elements             []Element
	Attributes           []Attribute
	AttributeGroups      []AttributeGroup
	Groups               []Group
	ComplexTypes         []ComplexType
	SimpleTypes          []SimpleType
}

// Include is a top-level xs:include declaration.
type Include struct {
	SchemaLocation string
}

// Import is a top-level xs:import declaration.
type Import struct {
	Namespace      string
	SchemaLocation string
}

// Override is a top-level xs:override declaration containing replacement
// declarations for components from the referenced schema.
type Override struct {
	SchemaLocation  string
	Elements        []Element
	Attributes      []Attribute
	AttributeGroups []AttributeGroup
	Groups          []Group
	ComplexTypes    []ComplexType
	SimpleTypes     []SimpleType
}

// Redefine is a top-level xs:redefine declaration containing replacement
// declarations that may derive from same-named components in the referenced
// schema.
type Redefine struct {
	SchemaLocation  string
	Elements        []Element
	Attributes      []Attribute
	AttributeGroups []AttributeGroup
	Groups          []Group
	ComplexTypes    []ComplexType
	SimpleTypes     []SimpleType
}

// Occurrence constants used by minOccurs/maxOccurs.
const (
	OccursDefault   = 1
	OccursUnbounded = -1
)

// Element is an xs:element declaration. It is used for both top-level and
// nested element declarations.
type Element struct {
	Name    string
	Ref     string
	Type    string
	Form    string
	Default string
	Fixed   string

	QName                 QName
	RefName               QName
	TypeName              QName
	SubstitutionGroup     string
	SubstitutionGroupName QName
	Abstract              bool
	Nillable              bool

	MinOccurs int
	MaxOccurs int

	Annotation           Annotation
	AnonymousComplexType *ComplexType
	AnonymousSimpleType  *SimpleType
}

// Attribute is an xs:attribute declaration.
type Attribute struct {
	Name    string
	Ref     string
	Type    string
	Use     string
	Form    string
	Default string
	Fixed   string

	QName    QName
	RefName  QName
	TypeName QName

	Annotation          Annotation
	AnonymousSimpleType *SimpleType
}

// ComplexType is an xs:complexType declaration.
type ComplexType struct {
	Name     string
	QName    QName
	Abstract bool
	Mixed    bool

	Annotation      Annotation
	Content         []Particle
	Attributes      []Attribute
	AttributeGroups []AttributeGroupRef
	AnyAttributes   []AnyAttribute
	Extension       *ComplexExtension
	Restriction     *ComplexRestriction
	SimpleContent   *SimpleContent
}

// ComplexExtension is an xs:extension inside xs:complexContent.
type ComplexExtension struct {
	Base     string
	BaseName QName

	Content         []Particle
	Attributes      []Attribute
	AttributeGroups []AttributeGroupRef
	AnyAttributes   []AnyAttribute
}

// ComplexRestriction is an xs:restriction inside xs:complexContent.
type ComplexRestriction struct {
	Base     string
	BaseName QName

	Content         []Particle
	Attributes      []Attribute
	AttributeGroups []AttributeGroupRef
	AnyAttributes   []AnyAttribute
}

// SimpleContent is an xs:simpleContent declaration.
type SimpleContent struct {
	Extension   *SimpleContentExtension
	Restriction *SimpleContentRestriction
}

// SimpleContentExtension is an xs:extension inside xs:simpleContent.
type SimpleContentExtension struct {
	Base     string
	BaseName QName

	Attributes      []Attribute
	AttributeGroups []AttributeGroupRef
	AnyAttributes   []AnyAttribute
}

// SimpleContentRestriction is an xs:restriction inside xs:simpleContent.
type SimpleContentRestriction struct {
	Base     string
	BaseName QName

	Attributes      []Attribute
	AttributeGroups []AttributeGroupRef
	AnyAttributes   []AnyAttribute
	Facets          []Facet
}

// Particle kinds.
const (
	ParticleKindElement = "element"
	ParticleKindGroup   = "group"
	ParticleKindAny     = "any"
)

// Particle is one ordered content-model item.
type Particle struct {
	Kind    string
	Element *Element
	Group   *ParticleGroup
	Any     *Any
}

// Any is an xs:any wildcard declaration.
type Any struct {
	Namespace       string
	ProcessContents string
	MinOccurs       int
	MaxOccurs       int
}

// AnyAttribute is an xs:anyAttribute wildcard declaration.
type AnyAttribute struct {
	Namespace       string
	ProcessContents string
}

// AttributeGroup is a top-level xs:attributeGroup declaration.
type AttributeGroup struct {
	Name            string
	QName           QName
	Attributes      []Attribute
	AttributeGroups []AttributeGroupRef
	AnyAttributes   []AnyAttribute
}

// AttributeGroupRef is an xs:attributeGroup reference.
type AttributeGroupRef struct {
	Ref     string
	RefName QName
}

// Group is a top-level xs:group declaration.
type Group struct {
	Name    string
	QName   QName
	Content ParticleGroup
}

// ParticleGroup is an xs:sequence, xs:choice, xs:all, or xs:group ref.
//
// When Ref is set, this particle is a reference to a global xs:group and the
// Particles slice is empty until resolved by the interpreter.
type ParticleGroup struct {
	Kind      string
	MinOccurs int
	MaxOccurs int
	Ref       string
	RefName   QName
	Particles []Particle
}

// Content model group kinds.
const (
	GroupSequence = "sequence"
	GroupChoice   = "choice"
	GroupAll      = "all"
)

// SimpleType is an xs:simpleType declaration.
type SimpleType struct {
	Name  string
	QName QName

	Annotation  Annotation
	Restriction *SimpleRestriction
	List        *SimpleList
	Union       *SimpleUnion
}

// SimpleRestriction is an xs:restriction inside an xs:simpleType.
type SimpleRestriction struct {
	Base     string
	BaseName QName
	Facets   []Facet
}

// SimpleList is an xs:list inside an xs:simpleType.
type SimpleList struct {
	ItemType     string
	ItemTypeName QName
	SimpleType   *SimpleType
}

// SimpleUnion is an xs:union inside an xs:simpleType.
type SimpleUnion struct {
	MemberTypes     string
	MemberTypeNames []QName
	SimpleTypes     []SimpleType
}

// Facet is a simple-type constraining facet such as xs:enumeration.
type Facet struct {
	Name  string
	Value string
}

// Annotation captures xs:documentation text when present.
type Annotation struct {
	Documentation []string
}

// Stats returns simple aggregate counts for inspection commands.
func (s Schemas) Stats() Stats {
	stats := Stats{Files: len(s.Files), Namespaces: make(map[string]struct{})}
	for _, file := range s.Files {
		stats.Elements += len(file.Elements)
		stats.ComplexTypes += len(file.ComplexTypes)
		stats.SimpleTypes += len(file.SimpleTypes)
		if file.TargetNamespace != "" {
			stats.Namespaces[file.TargetNamespace] = struct{}{}
		}
	}
	return stats
}

// Stats contains aggregate schema counts.
type Stats struct {
	Files        int
	Elements     int
	ComplexTypes int
	SimpleTypes  int
	Namespaces   map[string]struct{}
}
