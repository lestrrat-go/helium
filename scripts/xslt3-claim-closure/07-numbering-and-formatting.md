# Feature: numbering and decimal-format closure

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-numbering-formatting`

## Goal

Close remaining `xsl:number` + `format-number()` failures needed for an honest
XSLT 3.0 claim.

## Current State

Review found active failures in:

- `number`
- `format_number`

Signals included:

- missing static errors: `XTSE1295`, `XTSE1300`, `XTSE1290`
- missing dynamic errors: `XTTE0990`, `XTTE1000`, `XTDE0030`, `XTDE0980`
- wrong decimal-format lookup
- wrong localized / ordinal / titlecase number words

## Required Outcomes

### 1. Decimal Format Handling

- Fix declaration lookup
- Fix named decimal-format resolution
- Raise expected static errors

### 2. `xsl:number`

- Fix runtime semantics
- Fix expected dynamic errors
- Fix grouping / level / AVT interactions in failing cases

### 3. Localized Word Output

- Fix non-English word-number output required by W3C cases
- Cover ordinal + uppercase/titlecase variants

## Key Files

- `xslt3/compile.go`
- `xslt3/compile_instructions.go`
- `xslt3/execute_instructions.go`
- `xslt3/functions.go`
- `xpath3/functions_numeric.go`

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_(number|format_number)$' -v -count=1 -p 1 -timeout 600s > .tmp/numbering-formatting.txt 2>&1
```

## Acceptance

- `number` bucket passes
- `format_number` bucket passes
- No review-log failures remain for localized words or missing decimal-format errors
