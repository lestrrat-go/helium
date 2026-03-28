package xpath1

import (
	"errors"
	"fmt"
	"strconv"

	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"github.com/lestrrat-go/helium/internal/xpath1/lexer"
)

const maxParseDepth = 5000

var (
	errExpectedRParenAfterNode    = errors.New("expected ')' after node(")
	errExpectedRParenAfterText    = errors.New("expected ')' after text(")
	errExpectedRParenAfterComment = errors.New("expected ')' after comment(")
	errExpectedRParenAfterPI      = errors.New("expected ')' after processing-instruction(")
)

// parser builds an AST from a token stream.
type parser struct {
	lexer *lexer.Lexer
	depth int
}

// Parse parses an XPath expression string into an AST.
func Parse(expr string) (Expr, error) {
	l, err := newLexer(expr)
	if err != nil {
		return nil, err
	}
	p := &parser{lexer: l}
	e, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if tok := p.lexer.Peek(); tok.Type != TokenEOF {
		return nil, fmt.Errorf("%w: %s after expression", ErrUnexpectedToken, tok)
	}
	return e, nil
}

// parseExpr parses → OrExpr.
func (p *parser) parseExpr() (Expr, error) {
	p.depth++
	if p.depth > maxParseDepth {
		return nil, ErrExprTooDeep
	}
	defer func() { p.depth-- }()
	return p.parseOrExpr()
}

// parseOrExpr parses → AndExpr ('or' AndExpr)*.
func (p *parser) parseOrExpr() (Expr, error) {
	left, err := p.parseAndExpr()
	if err != nil {
		return nil, err
	}
	for p.lexer.Peek().Type == TokenOr {
		p.lexer.Next()
		right, err := p.parseAndExpr()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: TokenOr, Left: left, Right: right}
	}
	return left, nil
}

// parseAndExpr parses → EqualityExpr ('and' EqualityExpr)*.
func (p *parser) parseAndExpr() (Expr, error) {
	left, err := p.parseEqualityExpr()
	if err != nil {
		return nil, err
	}
	for p.lexer.Peek().Type == TokenAnd {
		p.lexer.Next()
		right, err := p.parseEqualityExpr()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: TokenAnd, Left: left, Right: right}
	}
	return left, nil
}

// parseEqualityExpr parses → RelationalExpr (('=' | '!=') RelationalExpr)*.
func (p *parser) parseEqualityExpr() (Expr, error) {
	left, err := p.parseRelationalExpr()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.lexer.Peek()
		if tok.Type != TokenEquals && tok.Type != TokenNotEquals {
			break
		}
		p.lexer.Next()
		right, err := p.parseRelationalExpr()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: tok.Type, Left: left, Right: right}
	}
	return left, nil
}

// parseRelationalExpr parses → AdditiveExpr (('<'|'>'|'<='|'>=') AdditiveExpr)*.
func (p *parser) parseRelationalExpr() (Expr, error) {
	left, err := p.parseAdditiveExpr()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.lexer.Peek()
		if tok.Type != TokenLess && tok.Type != TokenGreater &&
			tok.Type != TokenLessEq && tok.Type != TokenGreaterEq {
			break
		}
		p.lexer.Next()
		right, err := p.parseAdditiveExpr()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: tok.Type, Left: left, Right: right}
	}
	return left, nil
}

// parseAdditiveExpr parses → MultiplicativeExpr (('+' | '-') MultiplicativeExpr)*.
func (p *parser) parseAdditiveExpr() (Expr, error) {
	left, err := p.parseMultiplicativeExpr()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.lexer.Peek()
		if tok.Type != TokenPlus && tok.Type != TokenMinus {
			break
		}
		p.lexer.Next()
		right, err := p.parseMultiplicativeExpr()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: tok.Type, Left: left, Right: right}
	}
	return left, nil
}

// parseMultiplicativeExpr parses → UnaryExpr (('*' | 'div' | 'mod') UnaryExpr)*.
func (p *parser) parseMultiplicativeExpr() (Expr, error) {
	left, err := p.parseUnaryExpr()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.lexer.Peek()
		// Star here is the multiply operator (not name test).
		// It's multiply when the preceding token was a value-producing expression.
		if tok.Type != TokenStar && tok.Type != TokenDiv && tok.Type != TokenMod {
			break
		}
		p.lexer.Next()
		right, err := p.parseUnaryExpr()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: tok.Type, Left: left, Right: right}
	}
	return left, nil
}

// parseUnaryExpr parses → '-'* UnionExpr.
func (p *parser) parseUnaryExpr() (Expr, error) {
	negate := 0
	for p.lexer.Peek().Type == TokenMinus {
		p.lexer.Next()
		negate++
	}
	expr, err := p.parseUnionExpr()
	if err != nil {
		return nil, err
	}
	for range negate {
		expr = UnaryExpr{Operand: expr}
	}
	return expr, nil
}

// parseUnionExpr parses → PathExpr ('|' PathExpr)*.
func (p *parser) parseUnionExpr() (Expr, error) {
	left, err := p.parsePathExpr()
	if err != nil {
		return nil, err
	}
	for p.lexer.Peek().Type == TokenPipe {
		p.lexer.Next()
		right, err := p.parsePathExpr()
		if err != nil {
			return nil, err
		}
		left = UnionExpr{Left: left, Right: right}
	}
	return left, nil
}

// parsePathExpr → LocationPath | FilterExpr (('/' | '//') RelativeLocationPath)?
func (p *parser) parsePathExpr() (Expr, error) {
	tok := p.lexer.Peek()

	// Absolute location path starts with / or //
	if tok.Type == TokenSlash || tok.Type == TokenSlashSlash {
		return p.parseLocationPath()
	}

	// Check if this looks like a step (axis, name test, ., .., @)
	if p.looksLikeStep() {
		return p.parseLocationPath()
	}

	// Otherwise it's a filter expression (primary expr with optional predicates)
	filter, err := p.parseFilterExpr()
	if err != nil {
		return nil, err
	}

	// Optional trailing location path
	tok = p.lexer.Peek()
	if tok.Type == TokenSlash || tok.Type == TokenSlashSlash {
		p.lexer.Next()
		path := &LocationPath{Absolute: false}
		if tok.Type == TokenSlashSlash {
			// // is shorthand for /descendant-or-self::node()/
			path.Steps = append(path.Steps, Step{
				Axis:     AxisDescendantOrSelf,
				NodeTest: TypeTest{Type: NodeTestNode},
			})
		}
		steps, err := p.parseRelativeLocationPath()
		if err != nil {
			return nil, err
		}
		path.Steps = append(path.Steps, steps...)
		return PathExpr{Filter: filter, Path: path}, nil
	}

	return filter, nil
}

// parseFilterExpr parses → PrimaryExpr Predicate*.
func (p *parser) parseFilterExpr() (Expr, error) {
	primary, err := p.parsePrimaryExpr()
	if err != nil {
		return nil, err
	}

	var predicates []Expr
	for p.lexer.Peek().Type == TokenLBracket {
		pred, err := p.parsePredicate()
		if err != nil {
			return nil, err
		}
		predicates = append(predicates, pred)
	}

	if len(predicates) > 0 {
		return FilterExpr{Expr: primary, Predicates: predicates}, nil
	}
	return primary, nil
}

// parsePrimaryExpr parses → VariableRef | '(' Expr ')' | Literal | Number | FunctionCall.
func (p *parser) parsePrimaryExpr() (Expr, error) {
	tok := p.lexer.Peek()

	switch tok.Type {
	case TokenVariableRef:
		p.lexer.Next()
		return VariableExpr{Name: tok.Value}, nil

	case TokenLParen:
		return p.parseParenExpr()

	case TokenString:
		p.lexer.Next()
		return LiteralExpr{Value: tok.Value}, nil

	case TokenNumber:
		p.lexer.Next()
		return parseNumberLiteral(tok.Value)

	case TokenName:
		p.lexer.Next()

		// Unqualified function call: name(
		if p.lexer.Peek().Type == TokenLParen {
			return p.parseFunctionCall("", tok.Value)
		}

		// QName function call: prefix:name(
		if p.lexer.Peek().Type == TokenColon {
			p.lexer.Next() // consume ':'
			localTok := p.lexer.Peek()
			if localTok.Type == TokenName {
				p.lexer.Next() // consume local name
				if p.lexer.Peek().Type == TokenLParen {
					return p.parseFunctionCall(tok.Value, localTok.Value)
				}
				p.lexer.Backup() // local name
			}
			p.lexer.Backup() // ':'
		}

		// Not a function call — put the name back and let the caller handle it
		p.lexer.Backup()
		return nil, fmt.Errorf("%w: %s in primary expression", ErrUnexpectedToken, tok)

	default:
		return nil, fmt.Errorf("%w: %s in primary expression", ErrUnexpectedToken, tok)
	}
}

// parseParenExpr parses a parenthesised expression.
func (p *parser) parseParenExpr() (Expr, error) {
	p.lexer.Next() // consume '('
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.lexer.Peek().Type != TokenRParen {
		return nil, fmt.Errorf("%w: ')' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	return expr, nil
}

// parseNumberLiteral converts a numeric token value to a NumberExpr,
// accepting range errors (overflow → ±Inf, underflow → 0).
func parseNumberLiteral(s string) (NumberExpr, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return NumberExpr{}, fmt.Errorf("invalid number %q: %w", s, err)
	}
	return NumberExpr{Value: v}, nil
}

// parseFunctionCall parses function arguments after the name has been consumed.
// The opening '(' is the next token.
func (p *parser) parseFunctionCall(prefix, name string) (Expr, error) {
	p.lexer.Next() // consume '('
	var args []Expr
	if p.lexer.Peek().Type != TokenRParen {
		for {
			arg, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if p.lexer.Peek().Type != TokenComma {
				break
			}
			p.lexer.Next() // consume ','
		}
	}
	if p.lexer.Peek().Type != TokenRParen {
		displayName := name
		if prefix != "" {
			displayName = prefix + ":" + name
		}
		return nil, fmt.Errorf("%w: ')' in function call %s but got %s", ErrExpectedToken, displayName, p.lexer.Peek())
	}
	p.lexer.Next()
	return FunctionCall{Prefix: prefix, Name: name, Args: args}, nil
}

// parseLocationPath parses → AbsoluteLocationPath | RelativeLocationPath.
func (p *parser) parseLocationPath() (Expr, error) {
	tok := p.lexer.Peek()

	path := &LocationPath{}

	if tok.Type == TokenSlash {
		path.Absolute = true
		p.lexer.Next()
		// Bare "/" is a valid path (selects root)
		if !p.looksLikeStep() {
			return path, nil
		}
		steps, err := p.parseRelativeLocationPath()
		if err != nil {
			return nil, err
		}
		path.Steps = steps
		return path, nil
	}

	if tok.Type == TokenSlashSlash {
		path.Absolute = true
		p.lexer.Next()
		// // is shorthand for /descendant-or-self::node()/
		path.Steps = append(path.Steps, Step{
			Axis:     AxisDescendantOrSelf,
			NodeTest: TypeTest{Type: NodeTestNode},
		})
		steps, err := p.parseRelativeLocationPath()
		if err != nil {
			return nil, err
		}
		path.Steps = append(path.Steps, steps...)
		return path, nil
	}

	// Relative
	steps, err := p.parseRelativeLocationPath()
	if err != nil {
		return nil, err
	}
	path.Steps = steps
	return path, nil
}

// parseRelativeLocationPath parses → Step (('/' | '//') Step)*.
func (p *parser) parseRelativeLocationPath() ([]Step, error) {
	step, err := p.parseStep()
	if err != nil {
		return nil, err
	}
	steps := []Step{step}

loop:
	for {
		tok := p.lexer.Peek()
		switch tok.Type {
		case TokenSlash:
			p.lexer.Next()
			s, err := p.parseStep()
			if err != nil {
				return nil, err
			}
			steps = append(steps, s)
		case TokenSlashSlash:
			p.lexer.Next()
			// Insert descendant-or-self::node() step
			steps = append(steps, Step{
				Axis:     AxisDescendantOrSelf,
				NodeTest: TypeTest{Type: NodeTestNode},
			})
			s, err := p.parseStep()
			if err != nil {
				return nil, err
			}
			steps = append(steps, s)
		default:
			break loop
		}
	}
	return steps, nil
}

// parseStep parses → AxisSpecifier NodeTest Predicate* | AbbreviatedStep.
func (p *parser) parseStep() (Step, error) {
	tok := p.lexer.Peek()

	// Abbreviated step: .
	if tok.Type == TokenDot {
		p.lexer.Next()
		return Step{
			Axis:     AxisSelf,
			NodeTest: TypeTest{Type: NodeTestNode},
		}, nil
	}

	// Abbreviated step: ..
	if tok.Type == TokenDotDot {
		p.lexer.Next()
		return Step{
			Axis:     AxisParent,
			NodeTest: TypeTest{Type: NodeTestNode},
		}, nil
	}

	// @ is short for attribute::
	axis := AxisChild
	switch tok.Type {
	case TokenAt:
		axis = AxisAttribute
		p.lexer.Next()
	case TokenName:
		// Check if this is an axis specifier (Name '::')
		p.lexer.Next()
		if p.lexer.Peek().Type == TokenColonColon {
			if a, ok := ixpath.AxisFromName(tok.Value); ok {
				axis = a
				p.lexer.Next() // consume '::'
			} else {
				return Step{}, fmt.Errorf("%w: %q", ErrUnknownAxis, tok.Value)
			}
		} else {
			p.lexer.Backup() // not an axis, put name back
		}
	}

	nodeTest, err := p.parseNodeTest(axis)
	if err != nil {
		return Step{}, err
	}

	var predicates []Expr
	for p.lexer.Peek().Type == TokenLBracket {
		pred, err := p.parsePredicate()
		if err != nil {
			return Step{}, err
		}
		predicates = append(predicates, pred)
	}

	return Step{
		Axis:       axis,
		NodeTest:   nodeTest,
		Predicates: predicates,
	}, nil
}

// parseNodeTest parses → NameTest | NodeType '(' ')' | 'processing-instruction' '(' Literal ')'.
func (p *parser) parseNodeTest(_ AxisType) (NodeTest, error) {
	tok := p.lexer.Peek()

	// * wildcard
	if tok.Type == TokenStar {
		p.lexer.Next()
		return NameTest{Local: "*"}, nil
	}

	if tok.Type != TokenName {
		return nil, fmt.Errorf("%w: node test but got %s", ErrExpectedToken, tok)
	}

	p.lexer.Next()

	// Check if it's a node type test: node(), text(), comment(), processing-instruction()
	if p.lexer.Peek().Type == TokenLParen {
		if nt, ok, err := p.parseNodeTypeTest(tok.Value); ok || err != nil {
			return nt, err
		}
		// Not a node type — it's a function call, back up
		p.lexer.Backup()
	}

	// Check for QName: prefix:local
	if p.lexer.Peek().Type == TokenColon {
		return p.parseQNameTest(tok.Value)
	}

	return NameTest{Local: tok.Value}, nil
}

// parseNodeTypeTest attempts to parse a node type test (node(), text(), comment(),
// processing-instruction()). It returns (result, true, nil) on success,
// (nil, false, nil) when the name is not a node-type keyword, and
// (nil, false, err) on a parse error.
func (p *parser) parseNodeTypeTest(name string) (NodeTest, bool, error) {
	switch name {
	case "node":
		p.lexer.Next() // (
		if p.lexer.Peek().Type != TokenRParen {
			return nil, true, errExpectedRParenAfterNode
		}
		p.lexer.Next() // )
		return TypeTest{Type: NodeTestNode}, true, nil
	case "text":
		p.lexer.Next()
		if p.lexer.Peek().Type != TokenRParen {
			return nil, true, errExpectedRParenAfterText
		}
		p.lexer.Next()
		return TypeTest{Type: NodeTestText}, true, nil
	case "comment":
		p.lexer.Next()
		if p.lexer.Peek().Type != TokenRParen {
			return nil, true, errExpectedRParenAfterComment
		}
		p.lexer.Next()
		return TypeTest{Type: NodeTestComment}, true, nil
	case "processing-instruction":
		p.lexer.Next()
		target := ""
		if p.lexer.Peek().Type == TokenString {
			target = p.lexer.Next().Value
		}
		if p.lexer.Peek().Type != TokenRParen {
			return nil, true, errExpectedRParenAfterPI
		}
		p.lexer.Next()
		return PITest{Target: target}, true, nil
	}
	return nil, false, nil
}

// parseQNameTest parses a prefix:local or prefix:* name test, given that the
// prefix token has already been consumed and a ':' token has been peeked.
func (p *parser) parseQNameTest(prefix string) (NodeTest, error) {
	p.lexer.Next() // consume ':'
	next := p.lexer.Peek()
	if next.Type == TokenStar {
		p.lexer.Next()
		return NameTest{Prefix: prefix, Local: "*"}, nil
	}
	if next.Type == TokenName {
		p.lexer.Next()
		return NameTest{Prefix: prefix, Local: next.Value}, nil
	}
	return nil, fmt.Errorf("%w: name or '*' after '%s:'", ErrExpectedToken, prefix)
}

// parsePredicate parses → '[' Expr ']'.
func (p *parser) parsePredicate() (Expr, error) {
	if p.lexer.Peek().Type != TokenLBracket {
		return nil, fmt.Errorf("%w: '[' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()

	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if p.lexer.Peek().Type != TokenRBracket {
		return nil, fmt.Errorf("%w: ']' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	return expr, nil
}

// looksLikeStep returns true if the next token(s) look like the start
// of a location step (as opposed to a primary expression).
func (p *parser) looksLikeStep() bool {
	tok := p.lexer.Peek()
	switch tok.Type {
	case TokenDot, TokenDotDot, TokenAt, TokenStar:
		return true
	case TokenName:
		// Could be a step (name test or axis) or a function call.
		// Peek ahead: if followed by '(' it's a function call — unless
		// it's a node-type test (node, text, comment, processing-instruction)
		// or an axis (name '::').
		p.lexer.Next()
		next := p.lexer.Peek()
		p.lexer.Backup()

		if next.Type == TokenColonColon {
			return true // axis::
		}
		if next.Type == TokenLParen {
			// node(), text(), comment(), processing-instruction() are steps
			switch tok.Value {
			case "node", "text", "comment", "processing-instruction":
				return true
			}
			return false // function call
		}
		if next.Type == TokenColon {
			// Could be prefix:local (name test) or prefix:name( (QName function call).
			// Peek further: Name : Name ( → function call, not a step.
			p.lexer.Next() // consume prefix
			p.lexer.Next() // consume ':'
			localTok := p.lexer.Peek()
			if localTok.Type == TokenName {
				p.lexer.Next() // consume local name
				afterLocal := p.lexer.Peek()
				p.lexer.Backup() // local name
				p.lexer.Backup() // ':'
				p.lexer.Backup() // prefix
				if afterLocal.Type == TokenLParen {
					return false // QName function call
				}
				return true // QName step (ns:elem)
			}
			p.lexer.Backup() // ':'
			p.lexer.Backup() // prefix
			return true      // prefix:* or similar
		}
		return true // plain name test
	}
	return false
}
