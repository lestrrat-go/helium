# Feature: xsl:override component replacement

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-pkg-override`

## Goal

Implement `xsl:override` so that a using stylesheet can replace components
from a used package.

## Current State

- `xsl:override` children of `xsl:use-package` are parsed by the compiler
- But the override logic does not replace package components — it only adds
- The `override` W3C bucket has ~108 failures, mostly from missing replacement

## Required Outcomes

### 1. Override Mechanics

When `xsl:use-package` contains `xsl:override`:

- Each child of `xsl:override` (template, function, variable, attribute-set, param)
  replaces the corresponding component from the used package
- The replaced component must exist in the package and must be `public` or `final`
  (not `private` or `abstract` unless the override makes it concrete)
- The replacement component inherits the original's visibility unless explicitly changed

### 2. Template Override

- Named template override: match by name
- Match template override: match by pattern + mode + priority
- The overriding template completely replaces the original
- `xsl:original` instruction calls the overridden template from within the replacement

### 3. Function Override

- Match by QName + arity
- Overriding function replaces the original
- `xsl:original` calls the overridden function

### 4. Variable/Param Override

- Match by QName
- New value replaces original

### 5. `xsl:original`

- Only valid inside `xsl:override` children
- Calls the component being overridden
- For templates: acts like `xsl:apply-templates` or `xsl:call-template` to the original
- For functions: acts like a function call to the original
- Static error if used outside override context

### 6. Static Errors

- `XTSE3058`: `xsl:override` child has no matching component in used package
- `XTSE3062`: overriding a private component
- `XTSE3070`: overriding a final component

## Key Files

- `xslt3/compile.go` — `compileUsePackage`, `mergePackageComponents`
- `xslt3/compile_instructions.go` — `xsl:original` instruction compile
- `xslt3/execute_instructions.go` — `xsl:original` execution
- `xslt3/instruction.go` — `OriginalInst` type
- `xslt3/stylesheet.go` — override chain tracking

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_override$' -v -count=1 -p 1 -timeout 600s > .tmp/override.txt 2>&1
```

## Acceptance

- Override replaces package components, not just adds alongside
- `xsl:original` calls the overridden component
- Static errors raised for invalid overrides
- Override bucket failures drop significantly
