package xslt3

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// compileUsePackage handles xsl:use-package by resolving and compiling the
// referenced package, then merging its public components into the current
// stylesheet.
func (c *compiler) compileUsePackage(ctx context.Context, elem *helium.Element) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	pkgName := getAttr(elem, "name")
	if pkgName == "" {
		return staticError(errCodeXTSE0010, "xsl:use-package requires name attribute")
	}
	pkgVersion := getAttr(elem, "package-version")

	if c.packageResolver == nil {
		return staticError(errCodeXTSE3000,
			"xsl:use-package requires a PackageResolver but none is configured (package %q)", pkgName)
	}

	rc, pkgBaseURI, err := c.packageResolver.ResolvePackage(pkgName, pkgVersion)
	if err != nil {
		return staticError(errCodeXTSE3000,
			"xsl:use-package: cannot resolve package %q version %q: %v", pkgName, pkgVersion, err)
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("xsl:use-package: cannot read package %q: %w", pkgName, err)
	}

	doc, err := helium.NewParser().Parse(ctx, data)
	if err != nil {
		return fmt.Errorf("xsl:use-package: cannot parse package %q: %w", pkgName, err)
	}

	// Compile the package with its own compiler
	pkgCfg := &compileConfig{
		baseURI:         pkgBaseURI,
		resolver:        c.resolver,
		packageResolver: c.packageResolver,
		isSubPackage:    true,
	}
	pkgSS, err := compile(ctx, doc, pkgCfg)
	if err != nil {
		return fmt.Errorf("xsl:use-package: cannot compile package %q: %w", pkgName, err)
	}

	c.stylesheet.usedPackages = append(c.stylesheet.usedPackages, pkgSS)

	// Process xsl:override children (compile overrides, validate against package)
	oset, err := c.processOverrides(ctx, elem, pkgSS)
	if err != nil {
		return err
	}

	// Merge components from the package, respecting visibility, xsl:accept rules, and overrides.
	if err := c.mergePackageComponents(ctx, pkgSS, elem, oset); err != nil {
		return err
	}

	return nil
}

// getComponentVisibility returns the effective visibility of a component.
// It checks the per-component visibility maps first, then falls back to
// the component's own Visibility field, then to the default.
func getComponentVisibility(pkg *Stylesheet, compType, compName string) string {
	switch compType {
	case xslElemTemplate:
		if pkg.templateVisibility != nil {
			if v, ok := pkg.templateVisibility[compName]; ok {
				return v
			}
		}
	case xslElemFunction:
		if pkg.functionVisibility != nil {
			if v, ok := pkg.functionVisibility[compName]; ok {
				return v
			}
		}
	case xslElemVariable:
		if pkg.variableVisibility != nil {
			if v, ok := pkg.variableVisibility[compName]; ok {
				return v
			}
		}
		if pkg.globalParamVisibility != nil {
			if v, ok := pkg.globalParamVisibility[compName]; ok {
				return v
			}
		}
	case xslElemAttributeSet:
		if pkg.attrSetVisibility != nil {
			if v, ok := pkg.attrSetVisibility[compName]; ok {
				return v
			}
		}
	case xslElemMode:
		if pkg.modeDefs != nil {
			if md, ok := pkg.modeDefs[compName]; ok && md.Visibility != "" {
				return md.Visibility
			}
		}
	}
	return defaultComponentVisibility(pkg)
}

// isVisibleFromOutside returns true if the component's visibility allows
// it to be seen from a using stylesheet.
func isVisibleFromOutside(vis string) bool {
	return vis == visPublic || vis == visFinal || vis == visAbstract
}

// mergePackageComponents merges components from a used package into the
// current stylesheet, respecting visibility and xsl:accept rules.
func (c *compiler) mergePackageComponents(ctx context.Context, pkg *Stylesheet, usePackageElem *helium.Element, oset *overrideSet) error {
	// Collect namespace bindings from the use-package element
	nsBindings := c.collectElemNamespaces(ctx, usePackageElem)

	// Parse xsl:accept rules
	acceptRules := parseAcceptRules(usePackageElem, nsBindings)

	// XTSE3032: if xsl:accept has component="*", names must be a wildcard.
	for _, rule := range acceptRules {
		if rule.component == xslWildcard && !isWildcard(rule.names) {
			return staticError(errCodeXTSE3032,
				"xsl:accept with component='*' requires names to be a wildcard, got %q", rule.names)
		}
	}

	// Parse xsl:override children (collect overridden component names)
	overrideNames := c.collectOverrideNames(ctx, usePackageElem, nsBindings)

	// XTSE3055: it is a static error if an override declaration is homonymous
	// with any other declaration in the using package, regardless of import
	// precedence.
	if oset != nil {
		for name := range oset.namedTemplates {
			if _, exists := c.localTemplateNames[name]; exists {
				return staticError(errCodeXTSE3055,
					"override template %q conflicts with a local template of the same name", name)
			}
		}
		for fk := range oset.functions {
			if _, exists := c.stylesheet.functions[fk]; exists {
				return staticError(errCodeXTSE3055,
					"override function %s#%d conflicts with a local function of the same name",
					fmt.Sprintf("{%s}%s", fk.Name.URI, fk.Name.Name), fk.Arity)
			}
		}
		for name := range oset.variables {
			for _, v := range c.stylesheet.globalVars {
				if v.Name == name && v.OwnerPackage == nil {
					return staticError(errCodeXTSE3055,
						"override variable %q conflicts with a local variable of the same name", name)
				}
			}
		}
	}

	// XTSE3040: it is a static error if the visibility assigned by xsl:accept
	// is incompatible with the component's declared visibility, unless the
	// matching token is a wildcard.
	if len(acceptRules) > 0 {
		for _, rule := range acceptRules {
			if isWildcard(rule.names) {
				continue // wildcards are exempt from XTSE3040
			}
			// Check templates
			if rule.component == xslWildcard || rule.component == xslElemTemplate {
				for _, tmpl := range pkg.templates {
					if tmpl.Name == "" {
						continue
					}
					if !componentNameMatches(tmpl.Name, rule.names) {
						continue
					}
					pkgVis := getComponentVisibility(pkg, xslElemTemplate, tmpl.Name)
					if !isAcceptVisibilityCompatible(pkgVis, rule.visibility) {
						return staticError(errCodeXTSE3040,
							"xsl:accept: visibility %q is incompatible with component %q (declared %s)",
							rule.visibility, tmpl.Name, pkgVis)
					}
				}
			}
			// Check functions
			if rule.component == xslWildcard || rule.component == xslElemFunction {
				for fk, fn := range pkg.functions {
					key := functionVisKey(fk.Name, len(fn.Params))
					if !componentNameMatches(key, rule.names) {
						continue
					}
					pkgVis := getComponentVisibility(pkg, xslElemFunction, key)
					if !isAcceptVisibilityCompatible(pkgVis, rule.visibility) {
						return staticError(errCodeXTSE3040,
							"xsl:accept: visibility %q is incompatible with component %q (declared %s)",
							rule.visibility, key, pkgVis)
					}
				}
			}
			// Check variables
			if rule.component == xslWildcard || rule.component == xslElemVariable {
				for _, v := range pkg.globalVars {
					if !componentNameMatches(v.Name, rule.names) {
						continue
					}
					pkgVis := getComponentVisibility(pkg, xslElemVariable, v.Name)
					if !isAcceptVisibilityCompatible(pkgVis, rule.visibility) {
						return staticError(errCodeXTSE3040,
							"xsl:accept: visibility %q is incompatible with component %q (declared %s)",
							rule.visibility, v.Name, pkgVis)
					}
				}
			}
		}
	}

	// XTSE3051: it is a static error if a non-wildcard token in the names
	// attribute of xsl:accept matches the symbolic name of a component
	// declared within an xsl:override child of the same xsl:use-package.
	if len(acceptRules) > 0 && len(overrideNames) > 0 {
		for _, rule := range acceptRules {
			if isWildcard(rule.names) {
				continue // wildcards are exempt
			}
			// Check if any override name matches this accept name
			for overKey := range overrideNames {
				// overKey is "type:name" (e.g., "template:t-public")
				parts := splitOverrideKey(overKey)
				if parts == nil {
					continue
				}
				overType, overName := parts[0], parts[1]
				if rule.component != xslWildcard && rule.component != overType {
					continue
				}
				if componentNameMatches(overName, rule.names) {
					return staticError(errCodeXTSE3051,
						"xsl:accept name %q matches overridden component %q", rule.names, overName)
				}
			}
		}
	}

	// Merge templates (with lower import precedence than current)
	for _, tmpl := range pkg.templates {
		if tmpl.Name != "" {
			pkgVis := getComponentVisibility(pkg, xslElemTemplate, tmpl.Name)
			if !isVisibleFromOutside(pkgVis) {
				continue // private components are not merged
			}
			// Apply accept rules
			if len(acceptRules) > 0 {
				acceptVis := applyAcceptRules(xslElemTemplate, tmpl.Name, acceptRules, pkgVis)
				if acceptVis == visHidden {
					continue
				}
				tmpl.Visibility = acceptVis
			}
		}

		// Skip overridden components (they'll be replaced)
		if tmpl.Name != "" {
			if _, overridden := overrideNames[xslElemTemplate+":"+tmpl.Name]; overridden {
				continue
			}
		}

		// XTSE3050: local template with same name as package component
		if tmpl.Name != "" {
			if _, local := c.localTemplateNames[tmpl.Name]; local {
				return staticError(errCodeXTSE3050,
					"local template %q conflicts with public component from used package", tmpl.Name)
			}
		}

		tmpl.ImportPrec = c.importPrec - 1
		tmpl.MinImportPrec = tmpl.ImportPrec // package templates have no sub-imports
		tmpl.OwnerPackage = pkg
		c.stylesheet.templates = append(c.stylesheet.templates, tmpl)
		if tmpl.Name != "" {
			if existing, exists := c.stylesheet.namedTemplates[tmpl.Name]; !exists {
				c.stylesheet.namedTemplates[tmpl.Name] = tmpl
			} else if existing.OwnerPackage != nil && existing.OwnerPackage != pkg {
				// XTSE3050: two use-packages accept homonymous components
				// with non-hidden visibility.
				return staticError(errCodeXTSE3050,
					"template %q accepted from multiple packages with non-hidden visibility",
					tmpl.Name)
			}
		}
		if tmpl.Match != nil {
			modes := resolveTemplateModes(tmpl.Mode)
			for _, mode := range modes {
				c.stylesheet.modeTemplates[mode] = append(c.stylesheet.modeTemplates[mode], tmpl)
			}
		}
	}

	// Merge functions
	for fk, fn := range pkg.functions {
		key := functionVisKey(fk.Name, len(fn.Params))
		pkgVis := getComponentVisibility(pkg, xslElemFunction, key)
		if !isVisibleFromOutside(pkgVis) {
			continue
		}
		if len(acceptRules) > 0 {
			acceptVis := applyAcceptRules(xslElemFunction, key, acceptRules, pkgVis)
			if acceptVis == visHidden {
				continue
			}
			fn.Visibility = acceptVis
		}
		if _, overridden := overrideNames[xslElemFunction+":"+key]; overridden {
			continue
		}
		if fn.OwnerPackage == nil {
			fn.OwnerPackage = pkg
		}
		fn.AcceptedFrom = pkg
		if existing, exists := c.stylesheet.functions[fk]; !exists {
			c.stylesheet.functions[fk] = fn
		} else if existing.AcceptedFrom != nil && existing.AcceptedFrom != pkg {
			// XTSE3050: two use-packages accept the same function
			// with non-hidden visibility.
			return staticError(errCodeXTSE3050,
				"function %q accepted from multiple packages with non-hidden visibility",
				key)
		}
	}

	// Merge global variables from the used package.
	for _, v := range pkg.globalVars {
		pkgVis := getComponentVisibility(pkg, xslElemVariable, v.Name)
		if _, overridden := overrideNames[xslElemVariable+":"+v.Name]; overridden {
			continue
		}
		if !isVisibleFromOutside(pkgVis) {
			continue
		}
		if len(acceptRules) > 0 {
			acceptVis := applyAcceptRules(xslElemVariable, v.Name, acceptRules, pkgVis)
			if acceptVis == visHidden {
				continue
			}
			v.Visibility = acceptVis
		}
		// XTSE3050: local variable with same name as package component
		if _, local := c.localVarNames[v.Name]; local {
			return staticError(errCodeXTSE3050,
				"local variable %q conflicts with public component from used package", v.Name)
		}
		v.OwnerPackage = pkg
		c.stylesheet.globalVars = append(c.stylesheet.globalVars, v)
	}

	// Merge global params
	for _, p := range pkg.globalParams {
		pkgVis := getComponentVisibility(pkg, xslElemVariable, p.Name)
		if !isVisibleFromOutside(pkgVis) {
			continue
		}
		if len(acceptRules) > 0 {
			acceptVis := applyAcceptRules(xslElemVariable, p.Name, acceptRules, pkgVis)
			if acceptVis == visHidden {
				continue
			}
		}
		c.stylesheet.globalParams = append(c.stylesheet.globalParams, p)
	}

	// Note: keys are package-scoped and NOT merged into the using
	// stylesheet. Package code uses its own keys via effectiveKeys().

	// Note: decimal formats are package-scoped and NOT merged into the
	// using stylesheet. Package code uses its own formats via effectiveDecimalFormats().

	// Note: named outputs are package-scoped and NOT merged into the using
	// stylesheet. Package code references its own outputs via effectiveOutputs().

	// Merge mode definitions
	if pkg.modeDefs != nil {
		if c.stylesheet.modeDefs == nil {
			c.stylesheet.modeDefs = make(map[string]*modeDef)
		}
		for name, md := range pkg.modeDefs {
			if !isVisibleFromOutside(md.Visibility) {
				continue
			}
			if len(acceptRules) > 0 {
				acceptVis := applyAcceptRules(xslElemMode, name, acceptRules, md.Visibility)
				if acceptVis == visHidden {
					continue
				}
				md.Visibility = acceptVis
			}
			// XTSE3050: local mode with same name as package mode
			if _, local := c.localModeNames[name]; local {
				return staticError(errCodeXTSE3050,
					"local mode %q conflicts with public component from used package", name)
			}
			if _, exists := c.stylesheet.modeDefs[name]; !exists {
				c.stylesheet.modeDefs[name] = md
			}
		}
	}

	// Merge accumulators from the used package. Mark them as package-imported
	// so that a local accumulator with the same name can shadow them without
	// raising XTSE3350 (accumulators in different packages may share a name).
	for name, acc := range pkg.accumulators {
		if _, exists := c.stylesheet.accumulators[name]; !exists {
			acc.FromPackage = true
			c.stylesheet.accumulators[name] = acc
			c.stylesheet.accumulatorOrder = append(c.stylesheet.accumulatorOrder, name)
		}
	}

	// Merge attribute sets. Private attribute-sets are also merged (for
	// package-internal use-attribute-sets references) but keep their visibility.
	if pkg.attributeSets != nil {
		if c.stylesheet.attributeSets == nil {
			c.stylesheet.attributeSets = make(map[string]*attributeSetDef)
		}
		for name, as := range pkg.attributeSets {
			if _, overridden := overrideNames[xslElemAttributeSet+":"+name]; overridden {
				continue
			}
			pkgVis := getComponentVisibility(pkg, xslElemAttributeSet, name)
			if len(acceptRules) > 0 {
				acceptVis := applyAcceptRules(xslElemAttributeSet, name, acceptRules, pkgVis)
				if acceptVis == visHidden {
					// Mark as hidden but still merge so that package-
					// internal use-attribute-sets references resolve.
					as.Visibility = visHidden
				}
			}
			as.OwnerPackage = pkg
			if _, exists := c.stylesheet.attributeSets[name]; !exists {
				c.stylesheet.attributeSets[name] = as
			}
		}
	}

	// Merge override components (these replace the originals).
	// Override functions/templates/variables also replace the entries in the
	// package's own maps so that package-internal calls dispatch to the
	// override (XSLT 3.0 late-binding / virtual-dispatch semantics).
	if oset != nil {
		for qn, fn := range oset.functions {
			c.stylesheet.functions[qn] = fn
			// Update the package's function map for late binding.
			// Package-internal calls dispatch to the override.
			pkg.functions[qn] = fn
		}
		for name, tmpl := range oset.namedTemplates {
			tmpl.ImportPrec = c.importPrec - 1
			tmpl.OwnerPackage = pkg
			c.stylesheet.templates = append(c.stylesheet.templates, tmpl)
			c.stylesheet.namedTemplates[name] = tmpl
			// Update the package's own named template map for late binding
			pkg.namedTemplates[name] = tmpl
		}
		for _, tmpl := range oset.matchTemplates {
			tmpl.ImportPrec = c.importPrec - 1
			tmpl.OwnerPackage = pkg
			c.stylesheet.templates = append(c.stylesheet.templates, tmpl)
			// Resolve mode list: may be multi-mode (e.g. "m3 m4")
			modes := resolveTemplateModes(tmpl.Mode)
			for _, mode := range modes {
				c.stylesheet.modeTemplates[mode] = append(c.stylesheet.modeTemplates[mode], tmpl)
				// Update the package's mode template list for late binding
				pkg.modeTemplates[mode] = append(pkg.modeTemplates[mode], tmpl)
			}
		}
		for _, v := range oset.variables {
			c.stylesheet.globalVars = append(c.stylesheet.globalVars, v)
			// Update the package's own global vars for late binding
			for i, pv := range pkg.globalVars {
				if pv.Name == v.Name {
					pkg.globalVars[i] = v
					break
				}
			}
		}
		for _, p := range oset.params {
			c.stylesheet.globalParams = append(c.stylesheet.globalParams, p)
		}
		for name, as := range oset.attributeSets {
			if c.stylesheet.attributeSets == nil {
				c.stylesheet.attributeSets = make(map[string]*attributeSetDef)
			}
			// Link original attribute-set for xsl:original support.
			// The original comes from the used package, not the
			// stylesheet (it was skipped during merge because it's
			// overridden).
			if pkg.attributeSets != nil {
				if origAS, ok := pkg.attributeSets[name]; ok {
					as.OriginalAttrSet = origAS
				}
			}
			c.stylesheet.attributeSets[name] = as
			// Update the package's own attribute set map for late binding
			if pkg.attributeSets == nil {
				pkg.attributeSets = make(map[string]*attributeSetDef)
			}
			pkg.attributeSets[name] = as
		}
	}

	return nil
}

// resolveTemplateModes splits a mode string into individual modes for
// registration in modeTemplates. Handles multi-mode ("m3 m4"), #all,
// #default, #unnamed, and the empty string (default mode).
func resolveTemplateModes(mode string) []string {
	if mode == modeAll {
		return []string{""}
	}
	fields := strings.Fields(mode)
	if len(fields) == 0 {
		return []string{mode}
	}
	if len(fields) == 1 {
		m := fields[0]
		if m == lexicon.ModeDefault || m == lexicon.ModeUnnamed {
			return []string{""}
		}
		return []string{m}
	}
	var modes []string
	for _, m := range fields {
		if m == lexicon.ModeDefault || m == lexicon.ModeUnnamed {
			m = ""
		}
		modes = append(modes, m)
	}
	return modes
}

// isAcceptVisibilityCompatible checks whether changing a component's visibility
// from 'declared' to 'accepted' is allowed by the XSLT 3.0 spec §3.6.2.
// The rule: xsl:accept cannot increase the visibility of a component.
// private → public/final/abstract is not allowed.
// final → public is not allowed.
// hidden is always allowed (decreasing).
func isAcceptVisibilityCompatible(declared, accepted string) bool {
	if accepted == visHidden {
		return true // hiding is always allowed
	}
	// Cannot make private component visible (public/final/abstract)
	if declared == visPrivate && (accepted == visPublic || accepted == visFinal || accepted == visAbstract) {
		return false
	}
	// Cannot make final component public or abstract
	if declared == visFinal && (accepted == visPublic || accepted == visAbstract) {
		return false
	}
	// Cannot make non-abstract component abstract
	if declared != visAbstract && accepted == visAbstract {
		return false
	}
	return true
}

// splitOverrideKey splits an override key "type:name" into its components.
func splitOverrideKey(key string) []string {
	idx := strings.Index(key, ":")
	if idx < 0 {
		return nil
	}
	return []string{key[:idx], key[idx+1:]}
}
