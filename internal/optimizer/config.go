package optimizer

// Flags controls which optimizer passes run.
type Flags uint32

const (
	FlagRemoveEmptyEnumVariants Flags = 1 << iota
	FlagRemoveEmptyEnums
	FlagRemoveDuplicateUnionVariants
	FlagRemoveEmptyUnions
	FlagUseUnrestrictedBaseComplex
	FlagUseUnrestrictedBaseSimple
	FlagUseUnrestrictedBaseEnum
	FlagUseUnrestrictedBaseUnion
	FlagConvertDynamicToChoice
	FlagFlattenComplexTypes
	FlagFlattenUnions
	FlagMergeEnumUnions
	FlagResolveTypedefs
	FlagRemoveDuplicates
	FlagMergeChoiceCardinalities
	FlagSimplifyMixedTypes
	FlagReplaceXSAnyType
	FlagMergeDynamicTypes
)

const (
	FlagUseUnrestrictedBase = FlagUseUnrestrictedBaseComplex |
		FlagUseUnrestrictedBaseSimple |
		FlagUseUnrestrictedBaseEnum |
		FlagUseUnrestrictedBaseUnion

	// FlagSerde groups passes that are commonly enabled for serde-oriented output.
	FlagSerde = FlagFlattenComplexTypes | FlagFlattenUnions | FlagMergeEnumUnions
)

// DefaultFlags enables the standard optimizer pass set, including typedef
// resolution and duplicate removal used by the Go pipeline.
func DefaultFlags() Flags {
	return FlagRemoveEmptyEnumVariants |
		FlagRemoveEmptyEnums |
		FlagRemoveDuplicateUnionVariants |
		FlagRemoveEmptyUnions |
		FlagUseUnrestrictedBaseSimple |
		FlagResolveTypedefs |
		FlagRemoveDuplicates
}
