package xslt3

import (
	"context"
	"fmt"
	"io"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// compileUsePackage handles xsl:use-package by resolving and compiling the
// referenced package, then merging its public components into the current
// stylesheet.
func (c *compiler) compileUsePackage(elem *helium.Element) error {
	pkgName := getAttr(elem, "name")
	if pkgName == "" {
		return staticError(errCodeXTSE0010, "xsl:use-package requires name attribute")
	}
	pkgVersion := getAttr(elem, "package-version")

	if c.packageResolver == nil {
		// No package resolver configured; try to ignore gracefully.
		// This is a real limitation: without a resolver, xsl:use-package cannot work.
		return nil
	}

	rc, pkgBaseURI, err := c.packageResolver.ResolvePackage(pkgName, pkgVersion)
	if err != nil {
		return fmt.Errorf("xsl:use-package: cannot resolve package %q version %q: %w", pkgName, pkgVersion, err)
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("xsl:use-package: cannot read package %q: %w", pkgName, err)
	}

	ctx := context.Background()
	doc, err := helium.Parse(ctx, data)
	if err != nil {
		return fmt.Errorf("xsl:use-package: cannot parse package %q: %w", pkgName, err)
	}

	// Compile the package with its own compiler
	pkgCfg := &compileConfig{
		baseURI:         pkgBaseURI,
		resolver:        c.resolver,
		packageResolver: c.packageResolver,
	}
	pkgSS, err := compile(doc, pkgCfg)
	if err != nil {
		return fmt.Errorf("xsl:use-package: cannot compile package %q: %w", pkgName, err)
	}

	c.stylesheet.usedPackages = append(c.stylesheet.usedPackages, pkgSS)

	// Process xsl:override children (compile overrides, validate against package)
	oset, err := c.processOverrides(elem, pkgSS)
	if err != nil {
		return err
	}

	// Merge components from the package, respecting visibility, xsl:accept rules, and overrides.
	if err := c.mergePackageComponents(pkgSS, elem, oset); err != nil {
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
func (c *compiler) mergePackageComponents(pkg *Stylesheet, usePackageElem *helium.Element, oset *overrideSet) error {
	// Collect namespace bindings from the use-package element
	nsBindings := c.collectElemNamespaces(usePackageElem)

	// Parse xsl:accept rules
	acceptRules := parseAcceptRules(usePackageElem, nsBindings)

	// Parse xsl:override children (collect overridden component names)
	overrideNames := c.collectOverrideNames(usePackageElem, nsBindings)

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

		tmpl.ImportPrec = c.importPrec - 1
		tmpl.MinImportPrec = tmpl.ImportPrec // package templates have no sub-imports
		c.stylesheet.templates = append(c.stylesheet.templates, tmpl)
		if tmpl.Name != "" {
			if _, exists := c.stylesheet.namedTemplates[tmpl.Name]; !exists {
				c.stylesheet.namedTemplates[tmpl.Name] = tmpl
			}
		}
		if tmpl.Match != nil {
			modes := []string{tmpl.Mode}
			if tmpl.Mode == "#all" {
				modes = []string{""}
			}
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
		if _, exists := c.stylesheet.functions[fk]; !exists {
			c.stylesheet.functions[fk] = fn
		}
	}

	// Merge global variables
	for _, v := range pkg.globalVars {
		pkgVis := getComponentVisibility(pkg, xslElemVariable, v.Name)
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
		if _, overridden := overrideNames[xslElemVariable+":"+v.Name]; overridden {
			continue
		}
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

	// Merge keys
	for name, defs := range pkg.keys {
		c.stylesheet.keys[name] = append(c.stylesheet.keys[name], defs...)
	}

	// Merge decimal formats
	if pkg.decimalFormats != nil {
		if c.stylesheet.decimalFormats == nil {
			c.stylesheet.decimalFormats = make(map[xpath3.QualifiedName]xpath3.DecimalFormat)
			c.stylesheet.decimalFmtPrec = make(map[xpath3.QualifiedName]int)
			c.stylesheet.decimalFmtSet = make(map[xpath3.QualifiedName]map[string]struct{})
		}
		for qn, df := range pkg.decimalFormats {
			if _, exists := c.stylesheet.decimalFormats[qn]; !exists {
				c.stylesheet.decimalFormats[qn] = df
			}
		}
	}

	// Merge outputs
	for name, od := range pkg.outputs {
		if _, exists := c.stylesheet.outputs[name]; !exists {
			c.stylesheet.outputs[name] = od
		}
	}

	// Merge mode definitions
	if pkg.modeDefs != nil {
		if c.stylesheet.modeDefs == nil {
			c.stylesheet.modeDefs = make(map[string]*ModeDef)
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
			if _, exists := c.stylesheet.modeDefs[name]; !exists {
				c.stylesheet.modeDefs[name] = md
			}
		}
	}

	// Merge accumulators
	for name, acc := range pkg.accumulators {
		if _, exists := c.stylesheet.accumulators[name]; !exists {
			c.stylesheet.accumulators[name] = acc
			c.stylesheet.accumulatorOrder = append(c.stylesheet.accumulatorOrder, name)
		}
	}

	// Merge attribute sets
	if pkg.attributeSets != nil {
		if c.stylesheet.attributeSets == nil {
			c.stylesheet.attributeSets = make(map[string]*AttributeSetDef)
		}
		for name, as := range pkg.attributeSets {
			pkgVis := getComponentVisibility(pkg, xslElemAttributeSet, name)
			if !isVisibleFromOutside(pkgVis) {
				continue
			}
			if len(acceptRules) > 0 {
				acceptVis := applyAcceptRules(xslElemAttributeSet, name, acceptRules, pkgVis)
				if acceptVis == visHidden {
					continue
				}
			}
			if _, exists := c.stylesheet.attributeSets[name]; !exists {
				c.stylesheet.attributeSets[name] = as
			}
		}
	}

	return nil
}
