# Feature: component visibility enforcement

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-pkg-visibility`

## Goal

Implement XSLT 3.0 component visibility rules so that `xsl:accept`, `xsl:expose`,
and package-level visibility attributes are enforced at compile time and runtime.

## Current State

- `mergePackageComponents` copies all components unconditionally
- No visibility metadata is tracked per component
- `xsl:accept` children are parsed but ignored
- `xsl:expose` children are parsed but ignored
- Components marked `visibility="private"` in packages are still accessible from using stylesheets

## Required Outcomes

### 1. Visibility Model

Add visibility tracking to each component type:

- Templates (named + match): `public`, `private`, `final`, `abstract`
- Functions: `public`, `private`, `final`, `abstract`
- Variables/params: `public`, `private`, `final`, `abstract`
- Attribute sets: `public`, `private`, `final`, `abstract`
- Modes: `public`, `private`, `final`

Default visibility for package components is `private` (XSLT 3.0 §3.6.1).
Default visibility for stylesheet components (non-package) is `public`.

### 2. `xsl:accept` Processing

- Filter incoming package components based on `xsl:accept` children of `xsl:use-package`
- `names="*"` with `visibility="public"` → accept all as public
- `names="*"` with `visibility="hidden"` → hide all
- Specific name patterns override wildcard
- Component types: `template`, `function`, `variable`, `attribute-set`, `mode`

### 3. `xsl:expose` Processing

- Override visibility of components declared in the package
- Applied at package definition time, not use-package time
- Changes the effective visibility for all users of the package

### 4. Compile-Time Errors

Implement these static errors:

- `XTSE3040`: using a hidden component
- `XTSE3050`: accepting a component that doesn't exist
- `XTSE3060`: conflicting accept rules
- `XTSE3070`: expose of non-existent component

### 5. Runtime Errors

Implement these dynamic errors:

- `XTDE0040`: initial template is not public
- `XTDE0045`: initial mode is not public

## Key Files

- `xslt3/compile.go` — `mergePackageComponents`, `compileUsePackage`
- `xslt3/stylesheet.go` — component visibility fields
- `xslt3/execute.go` — initial mode/template visibility checks

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_(accept|expose)$' -v -count=1 -p 1 -timeout 600s > .tmp/visibility.txt 2>&1
```

```bash
go test ./xslt3/ -run 'TestW3C_package$' -v -count=1 -p 1 -timeout 600s > .tmp/package-post-visibility.txt 2>&1
```

## Acceptance

- `xsl:accept` filters components by visibility rules
- `xsl:expose` changes component visibility
- Private components are inaccessible from using stylesheets
- XTSE3040/3050/3060/3070 raised at compile time
- XTDE0040/0045 raised at runtime for private initial template/mode
