package xpath

import (
	"fmt"
	"strconv"
)

// Parser builds an AST from a token stream.
type Parser struct {
	lexer *Lexer
}

// Parse parses an XPath expression string into an AST.
func Parse(expr string) (Expr, error) {
	l, err := NewLexer(expr)
	if err != nil {
		return nil, err
	}
	p := &Parser{lexer: l}
	e, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if tok := p.lexer.Peek(); tok.Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token %s after expression", tok)
	}
	return e, nil
}

// parseExpr → OrExpr
func (p *Parser) parseExpr() (Expr, error) {
	return p.parseOrExpr()
}

// parseOrExpr → AndExpr ('or' AndExpr)*
func (p *Parser) parseOrExpr() (Expr, error) {
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

// parseAndExpr → EqualityExpr ('and' EqualityExpr)*
func (p *Parser) parseAndExpr() (Expr, error) {
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

// parseEqualityExpr → RelationalExpr (('=' | '!=') RelationalExpr)*
func (p *Parser) parseEqualityExpr() (Expr, error) {
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

// parseRelationalExpr → AdditiveExpr (('<'|'>'|'<='|'>=') AdditiveExpr)*
func (p *Parser) parseRelationalExpr() (Expr, error) {
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

// parseAdditiveExpr → MultiplicativeExpr (('+' | '-') MultiplicativeExpr)*
func (p *Parser) parseAdditiveExpr() (Expr, error) {
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

// parseMultiplicativeExpr → UnaryExpr (('*' | 'div' | 'mod') UnaryExpr)*
func (p *Parser) parseMultiplicativeExpr() (Expr, error) {
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

// parseUnaryExpr → '-'* UnionExpr
func (p *Parser) parseUnaryExpr() (Expr, error) {
	negate := 0
	for p.lexer.Peek().Type == TokenMinus {
		p.lexer.Next()
		negate++
	}
	expr, err := p.parseUnionExpr()
	if err != nil {
		return nil, err
	}
	for i := 0; i < negate; i++ {
		expr = UnaryExpr{Operand: expr}
	}
	return expr, nil
}

// parseUnionExpr → PathExpr ('|' PathExpr)*
func (p *Parser) parseUnionExpr() (Expr, error) {
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
func (p *Parser) parsePathExpr() (Expr, error) {
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

// parseFilterExpr → PrimaryExpr Predicate*
func (p *Parser) parseFilterExpr() (Expr, error) {
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

// parsePrimaryExpr → VariableRef | '(' Expr ')' | Literal | Number | FunctionCall
func (p *Parser) parsePrimaryExpr() (Expr, error) {
	tok := p.lexer.Peek()

	switch tok.Type {
	case TokenVariableRef:
		p.lexer.Next()
		return VariableExpr{Name: tok.Value}, nil

	case TokenLParen:
		p.lexer.Next()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.lexer.Peek().Type != TokenRParen {
			return nil, fmt.Errorf("expected ')' but got %s", p.lexer.Peek())
		}
		p.lexer.Next()
		return expr, nil

	case TokenString:
		p.lexer.Next()
		return LiteralExpr{Value: tok.Value}, nil

	case TokenNumber:
		p.lexer.Next()
		v, err := strconv.ParseFloat(tok.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q: %w", tok.Value, err)
		}
		return NumberExpr{Value: v}, nil

	case TokenName:
		// Check if it's a function call (Name followed by '(')
		p.lexer.Next()
		if p.lexer.Peek().Type == TokenLParen {
			return p.parseFunctionCall(tok.Value)
		}
		// Not a function call — put the name back and let the caller handle it
		p.lexer.Backup()
		return nil, fmt.Errorf("unexpected token %s in primary expression", tok)

	default:
		return nil, fmt.Errorf("unexpected token %s in primary expression", tok)
	}
}

// parseFunctionCall parses function arguments after the name has been consumed.
// The opening '(' is the next token.
func (p *Parser) parseFunctionCall(name string) (Expr, error) {
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
		return nil, fmt.Errorf("expected ')' in function call %s but got %s", name, p.lexer.Peek())
	}
	p.lexer.Next()
	return FunctionCall{Name: name, Args: args}, nil
}

// parseLocationPath → AbsoluteLocationPath | RelativeLocationPath
func (p *Parser) parseLocationPath() (Expr, error) {
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

// parseRelativeLocationPath → Step (('/' | '//') Step)*
func (p *Parser) parseRelativeLocationPath() ([]Step, error) {
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

// parseStep → AxisSpecifier NodeTest Predicate* | AbbreviatedStep
func (p *Parser) parseStep() (Step, error) {
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
			if a, ok := axisFromName(tok.Value); ok {
				axis = a
				p.lexer.Next() // consume '::'
			} else {
				return Step{}, fmt.Errorf("unknown axis %q", tok.Value)
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

// parseNodeTest → NameTest | NodeType '(' ')' | 'processing-instruction' '(' Literal ')'
func (p *Parser) parseNodeTest(axis AxisType) (NodeTest, error) {
	tok := p.lexer.Peek()

	// * wildcard
	if tok.Type == TokenStar {
		p.lexer.Next()
		return NameTest{Local: "*"}, nil
	}

	if tok.Type == TokenName {
		p.lexer.Next()

		// Check if it's a node type test: node(), text(), comment(), processing-instruction()
		if p.lexer.Peek().Type == TokenLParen {
			switch tok.Value {
			case "node":
				p.lexer.Next() // (
				if p.lexer.Peek().Type != TokenRParen {
					return nil, fmt.Errorf("expected ')' after node(")
				}
				p.lexer.Next() // )
				return TypeTest{Type: NodeTestNode}, nil
			case "text":
				p.lexer.Next()
				if p.lexer.Peek().Type != TokenRParen {
					return nil, fmt.Errorf("expected ')' after text(")
				}
				p.lexer.Next()
				return TypeTest{Type: NodeTestText}, nil
			case "comment":
				p.lexer.Next()
				if p.lexer.Peek().Type != TokenRParen {
					return nil, fmt.Errorf("expected ')' after comment(")
				}
				p.lexer.Next()
				return TypeTest{Type: NodeTestComment}, nil
			case "processing-instruction":
				p.lexer.Next()
				target := ""
				if p.lexer.Peek().Type == TokenString {
					target = p.lexer.Next().Value
				}
				if p.lexer.Peek().Type != TokenRParen {
					return nil, fmt.Errorf("expected ')' after processing-instruction(")
				}
				p.lexer.Next()
				return PITest{Target: target}, nil
			}
			// Not a node type — it's a function call, back up
			p.lexer.Backup()
		}

		// Check for QName: prefix:local
		if p.lexer.Peek().Type == TokenColon {
			prefix := tok.Value
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
			return nil, fmt.Errorf("expected name or '*' after '%s:'", prefix)
		}

		return NameTest{Local: tok.Value}, nil
	}

	return nil, fmt.Errorf("expected node test but got %s", tok)
}

// parsePredicate → '[' Expr ']'
func (p *Parser) parsePredicate() (Expr, error) {
	if p.lexer.Peek().Type != TokenLBracket {
		return nil, fmt.Errorf("expected '[' but got %s", p.lexer.Peek())
	}
	p.lexer.Next()

	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if p.lexer.Peek().Type != TokenRBracket {
		return nil, fmt.Errorf("expected ']' but got %s", p.lexer.Peek())
	}
	p.lexer.Next()
	return expr, nil
}

// looksLikeStep returns true if the next token(s) look like the start
// of a location step (as opposed to a primary expression).
func (p *Parser) looksLikeStep() bool {
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
		return true // plain name test
	}
	return false
}
