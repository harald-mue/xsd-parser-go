package optimizer

import "github.com/harald-mue/xsd-parser-go/internal/interpreter"

// Optimizer applies semantic simplifications to interpreted schema types.
type Optimizer struct {
	meta          interpreter.MetaTypes
	flags         Flags
	typedefs      map[typeKey]typeKey
	enumOnlyTypes map[typeKey]struct{}
}

// New creates an optimizer with the default flag set.
func New(meta interpreter.MetaTypes) *Optimizer {
	return &Optimizer{meta: meta, flags: DefaultFlags()}
}

// WithFlags returns a copy that uses the given flag set.
func (o *Optimizer) WithFlags(flags Flags) *Optimizer {
	next := *o
	next.flags = flags
	return &next
}

// Finish returns the optimized meta model.
func (o *Optimizer) Finish() interpreter.MetaTypes {
	return o.meta
}

// Optimize applies the default optimizer passes.
func Optimize(meta interpreter.MetaTypes) (interpreter.MetaTypes, error) {
	return New(meta).Run()
}

// OptimizeWithFlags applies the selected optimizer passes.
func OptimizeWithFlags(meta interpreter.MetaTypes, flags Flags) (interpreter.MetaTypes, error) {
	return New(meta).WithFlags(flags).Run()
}

// Run executes all enabled passes in pipeline order.
func (o *Optimizer) Run() (interpreter.MetaTypes, error) {
	o.invalidateTypedefs()

	if o.flags&FlagUseUnrestrictedBase != 0 {
		o.useUnrestrictedBaseType()
	}
	if o.flags&FlagReplaceXSAnyType != 0 {
		o.replaceXSAnyType()
	}
	if o.flags&FlagRemoveEmptyEnumVariants != 0 {
		o.removeEmptyEnumVariants()
	}
	if o.flags&FlagRemoveEmptyEnums != 0 {
		o.removeEmptyEnums()
	}
	if o.flags&FlagMergeDynamicTypes != 0 {
		o.mergeDynamicTypes()
	}
	if o.flags&FlagConvertDynamicToChoice != 0 {
		o.convertDynamicsToChoices()
	}
	if o.flags&FlagFlattenComplexTypes != 0 {
		o.flattenComplexTypes()
	}
	if o.flags&FlagFlattenUnions != 0 {
		o.flattenUnions()
	}
	if o.flags&FlagMergeEnumUnions != 0 {
		o.mergeEnumUnions()
	}
	if o.flags&FlagResolveTypedefs != 0 {
		o.meta = resolveTypedefs(o.meta)
		o.invalidateTypedefs()
	}
	if o.flags&FlagRemoveDuplicateUnionVariants != 0 {
		o.removeDuplicateUnionVariants()
	}
	if o.flags&FlagRemoveEmptyUnions != 0 {
		o.removeEmptyUnions()
	}
	if o.flags&FlagRemoveDuplicates != 0 {
		o.meta = removeDuplicates(o.meta)
	}
	if o.flags&FlagResolveTypedefs != 0 {
		o.meta = resolveTypedefs(o.meta)
		o.invalidateTypedefs()
	}
	if o.flags&FlagRemoveEmptyEnumVariants != 0 {
		o.removeEmptyEnumVariants()
	}
	if o.flags&FlagRemoveEmptyEnums != 0 {
		o.removeEmptyEnums()
	}
	if o.flags&FlagRemoveDuplicateUnionVariants != 0 {
		o.removeDuplicateUnionVariants()
	}
	if o.flags&FlagRemoveEmptyUnions != 0 {
		o.removeEmptyUnions()
	}
	if o.flags&FlagMergeChoiceCardinalities != 0 {
		o.mergeChoiceCardinalities()
	}
	if o.flags&FlagSimplifyMixedTypes != 0 {
		o.simplifyMixedTypes()
	}

	return o.meta, nil
}

func (o *Optimizer) invalidateTypedefs() {
	o.typedefs = nil
}

func (o *Optimizer) getTypedefs() map[typeKey]typeKey {
	if o.typedefs == nil {
		o.typedefs = buildTypedefMap(o.meta.Types)
	}
	return o.typedefs
}
