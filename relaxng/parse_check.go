package relaxng

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
)

// ruleFlags tracks ancestor context for forbidden-pattern-nesting checks.
type ruleFlags int

const (
	inAttribute     ruleFlags = 1 << iota
	inOneOrMore               // inside oneOrMore or zeroOrMore
	inList                    // inside list
	inDataExcept              // inside data/except
	inStart                   // top-level start pattern
	inOOMGroup                // inside oneOrMore//group
	inOOMInterleave           // inside oneOrMore//interleave
)

// checkRules walks the compiled pattern tree and reports forbidden nesting
// errors (e.g. list//element, attribute//attribute) and warnings.
func (c *compiler) checkRules(ctx context.Context) {
	if c.grammar.start == nil {
		return
	}
	visited := make(map[string]int8) // 0=unseen, 1=in-progress, 2=done
	c.checkPattern(ctx, c.grammar.start, inStart, visited)
}

// checkPattern recursively checks a pattern node for forbidden nestings,
// then recurses into children with updated flags.
func (c *compiler) checkPattern(ctx context.Context, pat *pattern, flags ruleFlags, visited map[string]int8) {
	if pat == nil {
		return
	}

	switch pat.kind {
	case patternElement:
		if flags&inList != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern list//element")
		}
		if flags&inAttribute != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern attribute//element")
		}
		if flags&inDataExcept != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern data/except//element")
		}
		// Element resets context: recurse attrs and children with 0.
		// (Attribute patterns in pat.attrs set inAttribute on their own children.)
		for _, attr := range pat.attrs {
			c.checkPattern(ctx, attr, 0, visited)
		}
		for _, child := range pat.children {
			c.checkPattern(ctx, child, 0, visited)
		}
		return

	case patternAttribute:
		if flags&inAttribute != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern attribute//attribute")
		}
		if flags&inList != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern list//attribute")
		}
		if flags&inStart != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern start//attribute")
		}
		if flags&inDataExcept != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern data/except//attribute")
		}
		if flags&inOOMGroup != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern oneOrMore//group//attribute")
		}
		if flags&inOOMInterleave != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern oneOrMore//interleave//attribute")
		}
		// Warnings for anyName/nsName without oneOrMore ancestor.
		if pat.nameClass != nil && flags&inOneOrMore == 0 {
			if pat.nameClass.kind == ncAnyName {
				c.addPatternWarning(ctx, pat, "Found anyName attribute without oneOrMore ancestor")
			}
			if pat.nameClass.kind == ncNsName {
				c.addPatternWarning(ctx, pat, "Found nsName attribute without oneOrMore ancestor")
			}
		}
		childFlags := flags | inAttribute
		for _, child := range pat.children {
			c.checkPattern(ctx, child, childFlags, visited)
		}
		return

	case patternList:
		if flags&inList != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern list//list")
		}
		if flags&inDataExcept != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern data/except//list")
		}
		childFlags := flags | inList
		for _, child := range pat.children {
			c.checkPattern(ctx, child, childFlags, visited)
		}
		return

	case patternText:
		if flags&inList != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern list//text")
		}
		if flags&inDataExcept != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern data/except//text")
		}
		return

	case patternInterleave:
		if flags&inList != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern list//interleave")
		}
		if flags&inDataExcept != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern data/except//interleave")
		}
		childFlags := flags
		if flags&inOneOrMore != 0 {
			childFlags |= inOOMInterleave
		}
		for _, child := range pat.children {
			c.checkPattern(ctx, child, childFlags, visited)
		}
		return

	case patternOneOrMore, patternZeroOrMore:
		childFlags := flags | inOneOrMore
		for _, child := range pat.children {
			c.checkPattern(ctx, child, childFlags, visited)
		}
		return

	case patternGroup:
		childFlags := flags
		if flags&inOneOrMore != 0 {
			childFlags |= inOOMGroup
		}
		for _, child := range pat.children {
			c.checkPattern(ctx, child, childFlags, visited)
		}
		return

	case patternData:
		// Children of data are except patterns.
		childFlags := flags | inDataExcept
		for _, child := range pat.children {
			c.checkPattern(ctx, child, childFlags, visited)
		}
		return

	case patternRef, patternParentRef:
		if flags&inDataExcept != 0 {
			c.addPatternError(ctx, pat, "Found forbidden pattern data/except//ref")
		}
		def, ok := c.grammar.defines[pat.name]
		if !ok {
			return
		}
		state := visited[pat.name]
		if state != 0 {
			return // in-progress or done
		}
		visited[pat.name] = 1 // in-progress
		c.checkPattern(ctx, def, flags, visited)
		visited[pat.name] = 2 // done
		return
	}

	// Default: choice, optional, mixed — pass flags through.
	for _, child := range pat.children {
		c.checkPattern(ctx, child, flags, visited)
	}
}

// addPatternError records a forbidden-nesting error.
func (c *compiler) addPatternError(ctx context.Context, p *pattern, msg string) {
	elemName := patternElemName(p.kind)
	var formatted string
	if c.filename != "" {
		formatted = rngParserErrorAt(c.filename, p.line, elemName, msg)
	} else {
		formatted = rngParserError(msg)
	}
	c.errorHandler.Handle(ctx, helium.NewLeveledError(formatted, helium.ErrorLevelFatal))
	c.errorCount++
}

// addPatternWarning records a forbidden-nesting warning.
func (c *compiler) addPatternWarning(ctx context.Context, p *pattern, msg string) {
	elemName := patternElemName(p.kind)
	var formatted string
	if c.filename != "" {
		formatted = fmt.Sprintf("%s:%d: element %s: Relax-NG parser warning : %s\n",
			c.filename, p.line, elemName, msg)
	} else {
		formatted = fmt.Sprintf("Relax-NG parser warning : %s\n", msg)
	}
	c.errorHandler.Handle(ctx, helium.NewLeveledError(formatted, helium.ErrorLevelWarning))
}

// patternElemName returns the XML element name for a pattern kind.
func patternElemName(k patternKind) string {
	switch k {
	case patternElement:
		return "element"
	case patternAttribute:
		return "attribute"
	case patternList:
		return "list"
	case patternText:
		return "text"
	case patternInterleave:
		return "interleave" //nolint:goconst
	case patternRef:
		return "ref"
	case patternParentRef:
		return "parentRef"
	case patternData:
		return "data"
	case patternGroup:
		return "group"
	case patternChoice:
		return "choice" //nolint:goconst
	case patternOneOrMore:
		return "oneOrMore"
	case patternZeroOrMore:
		return "zeroOrMore"
	case patternOptional:
		return "optional"
	default:
		return "unknown"
	}
}
