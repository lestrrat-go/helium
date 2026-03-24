# Feature: xsl:analyze-string

Read `00-shared-preamble.md` first.

## Branch

`feat-xslt3-analyze-string`

## Goal

Implement `xsl:analyze-string` — 55 failing tests in `TestW3C_analyze_string$`.

## XSLT 3.0 Specification

`xsl:analyze-string` applies a regular expression to a string and processes
matching and non-matching substrings separately.

```xml
<xsl:analyze-string select="expression" regex="regex" flags="flags">
  <xsl:matching-substring>
    <!-- body executed for each matching substring -->
  </xsl:matching-substring>
  <xsl:non-matching-substring>
    <!-- body executed for each non-matching substring -->
  </xsl:non-matching-substring>
  <xsl:fallback>...</xsl:fallback>
</xsl:analyze-string>
```

Within `xsl:matching-substring`:
- `.` (context item) is the matched substring as `xs:string`
- `regex-group(N)` returns the Nth captured group

Within `xsl:non-matching-substring`:
- `.` is the non-matching substring as `xs:string`

## Current State

`compile_instructions.go` line 291: the instruction compiles to an empty
`SequenceInst{}` — a complete no-op.

## Implementation Plan

### 1. Define Instruction Type

In `instruction.go`, add:

```go
type AnalyzeStringInst struct {
    Select           *xpath3.Expression
    Regex            *AVT
    Flags            *AVT
    MatchingBody     []Instruction
    NonMatchingBody  []Instruction
}
```

### 2. Compile

In `compile_instructions.go`, replace the stub:
- Parse `select`, `regex`, `flags` attributes (all are AVTs except `select`)
- Compile `xsl:matching-substring` and `xsl:non-matching-substring` children
- Register `regex-group()` as an available function

### 3. Execute

In `execute_instructions.go`, add `execAnalyzeString`:
- Evaluate `select` to get the input string
- Evaluate `regex` and `flags` AVTs
- Use Go's `regexp` package (translate XPath regex flags to Go flags)
- Split the input string into alternating match/non-match segments
- For each segment, set the context item to the substring and execute
  the appropriate body
- For matching segments, store captured groups for `regex-group()`

### 4. Register `regex-group()` Function

In `functions.go`, add `fnRegexGroup`:
- Takes one `xs:integer` argument (group number)
- Returns the captured group string, or empty string if out of range
- Use an `execContext` field to store current captured groups

### 5. XPath Regex to Go Regex

XPath regular expressions use a slightly different syntax from Go:
- XPath `\d` = Go `\d` (same)
- XPath `\i` and `\c` (XML name chars) need translation
- XPath `.` matches any char including newline in `s` flag mode
- XPath `x` flag (free-spacing) removes unescaped whitespace
- XPath `q` flag means literal (quote) the entire regex

Create a `translateXPathRegex(pattern, flags string) (string, error)` helper.

## Key Files

- `xslt3/instruction.go` — new `AnalyzeStringInst` type
- `xslt3/compile_instructions.go` — compile the instruction
- `xslt3/execute_instructions.go` — execute the instruction
- `xslt3/functions.go` — `regex-group()` function
- `xslt3/streamability_analysis.go` — add to instruction switches

## XPath Regex Reference

The XPath regex syntax is documented in F&O §5.6.1. Go's `regexp` package
uses RE2 syntax which is broadly compatible but lacks:
- `\p{IsBasicLatin}` Unicode block names (Go uses `\p{Latin}`)
- `\i` / `\c` XML name character classes
- Character class subtraction `[a-z-[aeiou]]`

For initial implementation, translate the most common patterns and let
edge cases fail gracefully.

## Verification

```bash
go test ./xslt3/ -run 'TestW3C_analyze_string$' -v -count=1 2>&1 > .tmp/analyze-string-final.txt
```
