# Feature: package version matching + dependency chains

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-pkg-versions`

## Goal

Implement package version matching and multi-level package dependency resolution.

## Current State

- `xsl:use-package` resolves packages by name but ignores `package-version`
- Version `"*"` (any version) is not handled
- Version ranges (`"1.0 to 2.0"`, `"1.*"`) are not parsed
- Package-to-package `xsl:use-package` chains partially work but version resolution
  is missing

## Required Outcomes

### 1. Version Matching

Implement XSLT 3.0 §3.7 version matching:

- Exact match: `package-version="1.0"` → must match exactly
- Wildcard: `package-version="*"` → any version
- Prefix wildcard: `package-version="1.*"` → any version starting with `1.`
- Range: `package-version="1.0 to 2.0"` → inclusive range
- Absent `package-version` → same as `"*"`

### 2. Version Comparison

- Versions are dot-separated integers: `1.0`, `2.3.1`
- Compare component by component, left to right
- Missing trailing components treated as zero: `1.0` = `1.0.0`

### 3. PackageResolver Version Support

- Extend `PackageResolver` interface if needed to support version queries
- The resolver receives name + version constraint and returns the matching package
- If multiple versions available, select the highest matching version

### 4. Dependency Chains

- Package A uses Package B which uses Package C
- Resolve transitively, detecting circular dependencies
- Each package in the chain gets its own compilation scope
- Components flow upward through the chain respecting visibility at each level

### 5. Static Errors

- `XTSE3000`: no package found matching name + version
- `XTSE3005`: circular package dependency

## Key Files

- `xslt3/compile.go` — `compileUsePackage`, version resolution
- `xslt3/options.go` — `PackageResolver` interface
- `xslt3/stylesheet.go` — `packageVersion` field

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_use_package$' -v -count=1 -p 1 -timeout 600s > .tmp/use-package.txt 2>&1
```

```bash
go test ./xslt3/ -run 'TestW3C_package$' -v -count=1 -p 1 -timeout 600s > .tmp/package-post-versions.txt 2>&1
```

## Acceptance

- `package-version="*"` resolves to any available version
- Version ranges and prefix wildcards work
- Multi-level package chains resolve correctly
- Circular dependencies detected with XTSE3005
- use-package bucket failures drop for version-related cases
