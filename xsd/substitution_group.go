package xsd

// inheritedTypeFromFirstSubstitutionHead resolves the effective type of an
// untyped substitution-group member by following only the first head QName at
// each declaration. XSD 1.1 permits a list-valued substitutionGroup, but the
// declaration's inherited type is defined by the first item in the actual value;
// the full head list is still used elsewhere for affiliation membership.
func inheritedTypeFromFirstSubstitutionHead(decl *ElementDecl, lookup func(QName) (*ElementDecl, bool)) *TypeDef {
	if decl == nil || lookup == nil {
		return nil
	}
	seen := map[QName]struct{}{decl.Name: {}}
	for {
		heads := decl.substitutionGroupHeads()
		if len(heads) == 0 {
			return nil
		}
		head := heads[0]
		if _, ok := seen[head]; ok {
			return nil
		}
		seen[head] = struct{}{}
		headDecl, ok := lookup(head)
		if !ok || headDecl == nil {
			return nil
		}
		if headDecl.Type != nil {
			return headDecl.Type
		}
		decl = headDecl
	}
}
