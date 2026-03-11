package xpath3

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

const maxParseDepth = 200

// parser builds an AST from a token stream.
type parser struct {
	lexer *lexer
	depth int
}

// Parse parses an XPath 3.1 expression string into an AST.
func Parse(expr string) (Expr, error) {
	l, err := newLexer(expr)
	if err != nil {
		return nil, err
	}
	p := &parser{lexer: l}
	e, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if tok := p.lexer.Peek(); tok.Type != TokenEOF {
		return nil, fmt.Errorf("%w: %s after expression", ErrUnexpectedToken, tok)
	}
	return e, nil
}

// parseExpression parses → ExprSingle (',' ExprSingle)* (sequence constructor).
func (p *parser) parseExpression() (Expr, error) {
	p.depth++
	if p.depth > maxParseDepth {
		return nil, ErrExprTooDeep
	}
	defer func() { p.depth-- }()

	first, err := p.parseExprSingle()
	if err != nil {
		return nil, err
	}

	if p.lexer.Peek().Type != TokenComma {
		return first, nil
	}

	items := []Expr{first}
	for p.lexer.Peek().Type == TokenComma {
		p.lexer.Next()
		item, err := p.parseExprSingle()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return SequenceExpr{Items: items}, nil
}

// parseExprSingle parses → ForExpr | LetExpr | QuantifiedExpr | IfExpr | TryCatchExpr | OrExpr.
func (p *parser) parseExprSingle() (Expr, error) {
	tok := p.lexer.Peek()
	switch tok.Type {
	case TokenFor:
		return p.parseFLWOR()
	case TokenLet:
		return p.parseFLWOR()
	case TokenSome:
		return p.parseQuantifiedExpr(true)
	case TokenEvery:
		return p.parseQuantifiedExpr(false)
	case TokenIf:
		return p.parseIfExpr()
	case TokenTry:
		return p.parseTryCatchExpr()
	}
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

// parseAndExpr parses → ComparisonExpr ('and' ComparisonExpr)*.
func (p *parser) parseAndExpr() (Expr, error) {
	left, err := p.parseComparisonExpr()
	if err != nil {
		return nil, err
	}
	for p.lexer.Peek().Type == TokenAnd {
		p.lexer.Next()
		right, err := p.parseComparisonExpr()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: TokenAnd, Left: left, Right: right}
	}
	return left, nil
}

// parseComparisonExpr parses → ConcatExpr ((GeneralComp | ValueComp) ConcatExpr)?.
// General: = != < <= > >=
// Value: eq ne lt le gt ge
func (p *parser) parseComparisonExpr() (Expr, error) {
	left, err := p.parseConcatExpr()
	if err != nil {
		return nil, err
	}
	tok := p.lexer.Peek()
	if isGeneralComp(tok.Type) || isValueComp(tok.Type) || isNodeComp(tok.Type) {
		p.lexer.Next()
		right, err := p.parseConcatExpr()
		if err != nil {
			return nil, err
		}
		return BinaryExpr{Op: tok.Type, Left: left, Right: right}, nil
	}
	return left, nil
}

// parseConcatExpr parses → RangeExpr ('||' RangeExpr)*.
func (p *parser) parseConcatExpr() (Expr, error) {
	left, err := p.parseRangeExpr()
	if err != nil {
		return nil, err
	}
	for p.lexer.Peek().Type == TokenConcat {
		p.lexer.Next()
		right, err := p.parseRangeExpr()
		if err != nil {
			return nil, err
		}
		left = ConcatExpr{Left: left, Right: right}
	}
	return left, nil
}

// parseRangeExpr parses → AdditiveExpr ('to' AdditiveExpr)?.
func (p *parser) parseRangeExpr() (Expr, error) {
	left, err := p.parseAdditiveExpr()
	if err != nil {
		return nil, err
	}
	if p.lexer.Peek().Type == TokenTo {
		p.lexer.Next()
		right, err := p.parseAdditiveExpr()
		if err != nil {
			return nil, err
		}
		return RangeExpr{Start: left, End: right}, nil
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

// parseMultiplicativeExpr parses → UnionExpr (('*' | 'div' | 'idiv' | 'mod') UnionExpr)*.
func (p *parser) parseMultiplicativeExpr() (Expr, error) {
	left, err := p.parseUnionExpr()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.lexer.Peek()
		if tok.Type != TokenStar && tok.Type != TokenDiv && tok.Type != TokenIdiv && tok.Type != TokenMod {
			break
		}
		p.lexer.Next()
		right, err := p.parseUnionExpr()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: tok.Type, Left: left, Right: right}
	}
	return left, nil
}

// parseUnionExpr parses → IntersectExceptExpr (('|' | 'union') IntersectExceptExpr)*.
func (p *parser) parseUnionExpr() (Expr, error) {
	left, err := p.parseIntersectExceptExpr()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.lexer.Peek()
		if tok.Type != TokenPipe && tok.Type != TokenUnion {
			break
		}
		p.lexer.Next()
		right, err := p.parseIntersectExceptExpr()
		if err != nil {
			return nil, err
		}
		left = UnionExpr{Left: left, Right: right}
	}
	return left, nil
}

// parseIntersectExceptExpr parses → InstanceOfExpr (('intersect' | 'except') InstanceOfExpr)*.
func (p *parser) parseIntersectExceptExpr() (Expr, error) {
	left, err := p.parseInstanceOfExpr()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.lexer.Peek()
		if tok.Type != TokenIntersect && tok.Type != TokenExcept {
			break
		}
		p.lexer.Next()
		right, err := p.parseInstanceOfExpr()
		if err != nil {
			return nil, err
		}
		left = IntersectExceptExpr{Op: tok.Type, Left: left, Right: right}
	}
	return left, nil
}

// parseInstanceOfExpr parses → TreatAsExpr ('instance' 'of' SequenceType)?.
func (p *parser) parseInstanceOfExpr() (Expr, error) {
	left, err := p.parseTreatAsExpr()
	if err != nil {
		return nil, err
	}
	if p.lexer.Peek().Type == TokenInstanceOf {
		p.lexer.Next() // consume 'instance'
		if p.lexer.Peek().Type != TokenOf {
			return nil, fmt.Errorf("%w: 'of' after 'instance' but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next() // consume 'of'
		st, err := p.parseSequenceType()
		if err != nil {
			return nil, err
		}
		return InstanceOfExpr{Expr: left, Type: st}, nil
	}
	return left, nil
}

// parseTreatAsExpr parses → CastableAsExpr ('treat' 'as' SequenceType)?.
func (p *parser) parseTreatAsExpr() (Expr, error) {
	left, err := p.parseCastableAsExpr()
	if err != nil {
		return nil, err
	}
	if p.lexer.Peek().Type == TokenTreatAs {
		p.lexer.Next() // consume 'treat'
		if p.lexer.Peek().Type != TokenAs {
			return nil, fmt.Errorf("%w: 'as' after 'treat' but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next() // consume 'as'
		st, err := p.parseSequenceType()
		if err != nil {
			return nil, err
		}
		return TreatAsExpr{Expr: left, Type: st}, nil
	}
	return left, nil
}

// parseCastableAsExpr parses → CastAsExpr ('castable' 'as' SingleType)?.
func (p *parser) parseCastableAsExpr() (Expr, error) {
	left, err := p.parseCastAsExpr()
	if err != nil {
		return nil, err
	}
	if p.lexer.Peek().Type == TokenCastableAs {
		p.lexer.Next() // consume 'castable'
		if p.lexer.Peek().Type != TokenAs {
			return nil, fmt.Errorf("%w: 'as' after 'castable' but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next() // consume 'as'
		typeName, allowEmpty, err := p.parseSingleType()
		if err != nil {
			return nil, err
		}
		return CastableExpr{Expr: left, Type: typeName, AllowEmpty: allowEmpty}, nil
	}
	return left, nil
}

// parseCastAsExpr parses → ArrowExpr ('cast' 'as' SingleType)?.
func (p *parser) parseCastAsExpr() (Expr, error) {
	left, err := p.parseArrowExpr()
	if err != nil {
		return nil, err
	}
	if p.lexer.Peek().Type == TokenCastAs {
		p.lexer.Next() // consume 'cast'
		if p.lexer.Peek().Type != TokenAs {
			return nil, fmt.Errorf("%w: 'as' after 'cast' but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next() // consume 'as'
		typeName, allowEmpty, err := p.parseSingleType()
		if err != nil {
			return nil, err
		}
		return CastExpr{Expr: left, Type: typeName, AllowEmpty: allowEmpty}, nil
	}
	return left, nil
}

// parseArrowExpr parses → UnaryExpr ('=>' ArrowFunctionSpecifier ArgumentList)*.
// Desugared: $x => f(a, b) becomes f($x, a, b).
func (p *parser) parseArrowExpr() (Expr, error) {
	left, err := p.parseUnaryExpr()
	if err != nil {
		return nil, err
	}
	for p.lexer.Peek().Type == TokenArrow {
		p.lexer.Next() // consume '=>'
		// ArrowFunctionSpecifier: Name or VarRef or ParenthesizedExpr
		funcSpec, prefix, name, err := p.parseArrowTarget()
		if err != nil {
			return nil, err
		}
		// Parse argument list
		if p.lexer.Peek().Type != TokenLParen {
			return nil, fmt.Errorf("%w: '(' after arrow target but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		args, err := p.parseArgumentList()
		if err != nil {
			return nil, err
		}
		// Desugar: prepend left as first argument
		allArgs := make([]Expr, 0, len(args)+1)
		allArgs = append(allArgs, left)
		allArgs = append(allArgs, args...)

		if funcSpec != nil {
			// Dynamic call: ($expr)(args...)
			left = DynamicFunctionCall{Func: funcSpec, Args: allArgs}
		} else {
			left = FunctionCall{Prefix: prefix, Name: name, Args: allArgs}
		}
	}
	return left, nil
}

// parseArrowTarget parses the function specifier after =>.
// Returns either (expr, "", "", nil) for dynamic targets or (nil, prefix, name, nil) for static.
func (p *parser) parseArrowTarget() (Expr, string, string, error) {
	tok := p.lexer.Peek()
	if tok.Type == TokenVariableRef {
		p.lexer.Next()
		return VariableExpr{Name: tok.Value}, "", "", nil
	}
	if tok.Type == TokenLParen {
		expr, err := p.parseParenExpr()
		if err != nil {
			return nil, "", "", err
		}
		return expr, "", "", nil
	}
	if tok.Type == TokenName || tok.Type == TokenMap || tok.Type == TokenArray {
		p.lexer.Next()
		prefix := ""
		name := tok.Value
		if p.lexer.Peek().Type == TokenColon {
			p.lexer.Next() // consume ':'
			localTok := p.lexer.Next()
			if localTok.Type != TokenName {
				return nil, "", "", fmt.Errorf("%w: name after '%s:' in arrow but got %s", ErrExpectedToken, name, localTok)
			}
			prefix = name
			name = localTok.Value
		}
		return nil, prefix, name, nil
	}
	return nil, "", "", fmt.Errorf("%w: arrow function target but got %s", ErrExpectedToken, tok)
}

// parseUnaryExpr parses → ('-' | '+')* ValueExpr.
// ValueExpr ::= SimpleMapExpr.
func (p *parser) parseUnaryExpr() (Expr, error) {
	negate := 0
	hasUnary := false
loop:
	for {
		tok := p.lexer.Peek()
		switch tok.Type {
		case TokenMinus:
			p.lexer.Next()
			negate++
			hasUnary = true
		case TokenPlus:
			p.lexer.Next()
			hasUnary = true
			// unary + is a no-op for negate count but still marks as having unary
		default:
			break loop
		}
	}
	expr, err := p.parseSimpleMapExpr()
	if err != nil {
		return nil, err
	}
	if hasUnary && negate == 0 {
		// Pure unary plus: wrap to enforce numeric type check
		expr = UnaryExpr{Operand: expr, Negate: false}
	}
	for i := 0; i < negate; i++ {
		expr = UnaryExpr{Operand: expr, Negate: true}
	}
	return expr, nil
}

// parseSimpleMapExpr parses → PathExpr ('!' PathExpr)*.
func (p *parser) parseSimpleMapExpr() (Expr, error) {
	left, err := p.parsePathExpr()
	if err != nil {
		return nil, err
	}
	for p.lexer.Peek().Type == TokenBang {
		p.lexer.Next()
		right, err := p.parsePathExpr()
		if err != nil {
			return nil, err
		}
		left = SimpleMapExpr{Left: left, Right: right}
	}
	return left, nil
}

// parsePathExpr → LocationPath | PostfixExpr (('/' | '//') RelativeLocationPath)?
func (p *parser) parsePathExpr() (Expr, error) {
	tok := p.lexer.Peek()

	var expr Expr

	// Absolute location path starts with / or //
	if tok.Type == TokenSlash || tok.Type == TokenSlashSlash {
		if tok.Type == TokenSlash {
			p.lexer.Next()
			if p.looksLikeStep() {
				steps, err := p.parseRelativeLocationPath()
				if err != nil {
					return nil, err
				}
				expr = &LocationPath{Absolute: true, Steps: steps}
			} else if p.canStartPostfixExpr() {
				// /functionCall(...) — root then non-step expression
				root := &LocationPath{Absolute: true}
				right, err := p.parsePostfixExpr()
				if err != nil {
					return nil, err
				}
				expr = PathStepExpr{Left: root, Right: right}
			} else {
				// Bare "/" selects document root
				expr = &LocationPath{Absolute: true}
			}
		} else {
			// //
			p.lexer.Next()
			descStep := Step{
				Axis:     AxisDescendantOrSelf,
				NodeTest: TypeTest{Kind: NodeKindNode},
			}
			if p.looksLikeStep() {
				steps, err := p.parseRelativeLocationPath()
				if err != nil {
					return nil, err
				}
				allSteps := append([]Step{descStep}, steps...)
				expr = &LocationPath{Absolute: true, Steps: allSteps}
			} else {
				// //functionCall(...)
				root := &LocationPath{Absolute: true, Steps: []Step{descStep}}
				right, err := p.parsePostfixExpr()
				if err != nil {
					return nil, err
				}
				expr = PathStepExpr{Left: root, Right: right}
			}
		}
	} else if p.looksLikeStep() {
		// Relative location path
		steps, err := p.parseRelativeLocationPath()
		if err != nil {
			return nil, err
		}
		expr = &LocationPath{Steps: steps}
	} else {
		// Primary/postfix expression
		postfix, err := p.parsePostfixExpr()
		if err != nil {
			return nil, err
		}
		expr = postfix
	}

	// Apply postfix operations
	var err error
	expr, err = p.parsePostfixOps(expr)
	if err != nil {
		return nil, err
	}

	// Handle path continuation: / or // followed by more steps or expressions
	return p.parsePathContinuation(expr)
}

// parsePathContinuation handles trailing / or // after the initial part of a path.
// It chains additional axis steps (via PathExpr) or non-step expressions (via PathStepExpr).
func (p *parser) parsePathContinuation(expr Expr) (Expr, error) {
	for {
		tok := p.lexer.Peek()
		if tok.Type != TokenSlash && tok.Type != TokenSlashSlash {
			return expr, nil
		}
		descOrSelf := tok.Type == TokenSlashSlash
		p.lexer.Next()

		if p.looksLikeStep() {
			// Collect consecutive axis steps into a LocationPath
			path := &LocationPath{}
			if descOrSelf {
				path.Steps = append(path.Steps, Step{
					Axis:     AxisDescendantOrSelf,
					NodeTest: TypeTest{Kind: NodeKindNode},
				})
			}
			steps, err := p.parseRelativeLocationPath()
			if err != nil {
				return nil, err
			}
			path.Steps = append(path.Steps, steps...)
			expr = PathExpr{Filter: expr, Path: path}
		} else {
			// Non-step expression: function call, variable, etc.
			if descOrSelf {
				// E1 // nonStep → E1 / descendant-or-self::node() / nonStep
				expr = PathExpr{
					Filter: expr,
					Path: &LocationPath{Steps: []Step{{
						Axis:     AxisDescendantOrSelf,
						NodeTest: TypeTest{Kind: NodeKindNode},
					}}},
				}
			}
			right, err := p.parsePostfixExpr()
			if err != nil {
				return nil, err
			}
			expr = PathStepExpr{Left: expr, Right: right}
		}

		// Apply postfix operations on the result (predicates, lookups)
		var err error
		expr, err = p.parsePostfixOps(expr)
		if err != nil {
			return nil, err
		}
	}
}

// canStartPostfixExpr returns true if the next token can begin a postfix expression.
func (p *parser) canStartPostfixExpr() bool {
	tok := p.lexer.Peek()
	switch tok.Type {
	case TokenVariableRef, TokenLParen, TokenString, TokenNumber,
		TokenDot, TokenQMark, TokenLBracket:
		return true
	case TokenName, TokenFunction, TokenMap, TokenArray:
		return true
	}
	return false
}

// parsePostfixExpr parses → PrimaryExpr (Predicate | ArgumentList | Lookup)*.
func (p *parser) parsePostfixExpr() (Expr, error) {
	primary, err := p.parsePrimaryExpr()
	if err != nil {
		return nil, err
	}
	return p.parsePostfixOps(primary)
}

// parsePostfixOps applies predicates, argument lists (dynamic calls), and lookups.
func (p *parser) parsePostfixOps(expr Expr) (Expr, error) {
	for {
		tok := p.lexer.Peek()
		switch tok.Type {
		case TokenLBracket:
			pred, err := p.parsePredicate()
			if err != nil {
				return nil, err
			}
			expr = FilterExpr{Expr: expr, Predicates: []Expr{pred}}
		case TokenLParen:
			// Dynamic function call: expr(args)
			args, err := p.parseArgumentList()
			if err != nil {
				return nil, err
			}
			expr = DynamicFunctionCall{Func: expr, Args: args}
		case TokenQMark:
			// Lookup: expr ? key
			p.lexer.Next()
			key, all, err := p.parseLookupKey()
			if err != nil {
				return nil, err
			}
			expr = LookupExpr{Expr: expr, Key: key, All: all}
		default:
			return expr, nil
		}
	}
}

// parsePrimaryExpr parses the primary expression productions.
func (p *parser) parsePrimaryExpr() (Expr, error) {
	tok := p.lexer.Peek()

	switch tok.Type {
	case TokenVariableRef:
		p.lexer.Next()
		return VariableExpr{Name: tok.Value}, nil

	case TokenLParen:
		return p.parseParenExprOrEmpty()

	case TokenString:
		p.lexer.Next()
		return LiteralExpr{Value: tok.Value}, nil

	case TokenNumber:
		p.lexer.Next()
		return parseNumberLiteral(tok.Value)

	case TokenDot:
		p.lexer.Next()
		return ContextItemExpr{}, nil

	case TokenQMark:
		// Unary lookup: ? key
		p.lexer.Next()
		key, all, err := p.parseLookupKey()
		if err != nil {
			return nil, err
		}
		return UnaryLookupExpr{Key: key, All: all}, nil

	case TokenFunction:
		// Could be inline function or function(*)
		return p.parseFunctionKeyword()

	case TokenMap:
		// map:func(...) is a namespace-prefixed function call, not a constructor
		if p.lexer.PeekAt(1).Type == TokenColon {
			return p.parseNamePrimary()
		}
		return p.parseMapConstructor()

	case TokenArray:
		// array:func(...) is a namespace-prefixed function call, not a constructor
		if p.lexer.PeekAt(1).Type == TokenColon {
			return p.parseNamePrimary()
		}
		return p.parseArrayCurlyConstructor()

	case TokenLBracket:
		return p.parseArraySquareConstructor()

	case TokenName:
		return p.parseNamePrimary()

	case TokenIf:
		return p.parseIfExpr()

	default:
		return nil, fmt.Errorf("%w: %s in primary expression", ErrUnexpectedToken, tok)
	}
}

// parseNamePrimary handles a Name at the start of a primary expression:
// function call, named function ref, or error.
func (p *parser) parseNamePrimary() (Expr, error) {
	tok := p.lexer.Next()
	prefix := ""
	name := tok.Value

	// Check for QName: prefix:name
	if p.lexer.Peek().Type == TokenColon {
		p.lexer.Next() // consume ':'
		localTok := p.lexer.Peek()
		if localTok.Type == TokenName {
			p.lexer.Next()
			prefix = name
			name = localTok.Value
		} else {
			p.lexer.Backup() // put ':' back
			p.lexer.Backup() // put name back
			return nil, fmt.Errorf("%w: %s in primary expression", ErrUnexpectedToken, tok)
		}
	}

	// Named function ref: name#arity
	if p.lexer.Peek().Type == TokenHash {
		p.lexer.Next() // consume '#'
		arityTok := p.lexer.Next()
		if arityTok.Type != TokenNumber {
			return nil, fmt.Errorf("%w: arity number after '#' but got %s", ErrExpectedToken, arityTok)
		}
		arity, err := strconv.Atoi(arityTok.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid arity %q: %w", arityTok.Value, err)
		}
		return NamedFunctionRef{Prefix: prefix, Name: name, Arity: arity}, nil
	}

	// Function call: name(args)
	if p.lexer.Peek().Type == TokenLParen {
		args, err := p.parseArgumentList()
		if err != nil {
			return nil, err
		}
		return FunctionCall{Prefix: prefix, Name: name, Args: args}, nil
	}

	// Not a primary — put back
	if prefix != "" {
		p.lexer.Backup() // local name
		p.lexer.Backup() // ':'
	}
	p.lexer.Backup() // name
	return nil, fmt.Errorf("%w: %s in primary expression", ErrUnexpectedToken, tok)
}

// parseParenExprOrEmpty handles '(' which could be an empty sequence () or a parenthesized expression.
func (p *parser) parseParenExprOrEmpty() (Expr, error) {
	p.lexer.Next() // consume '('
	if p.lexer.Peek().Type == TokenRParen {
		p.lexer.Next() // consume ')'
		return SequenceExpr{Items: nil}, nil
	}
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.lexer.Peek().Type != TokenRParen {
		return nil, fmt.Errorf("%w: ')' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	return expr, nil
}

// parseParenExpr parses '(' Expr ')'.
func (p *parser) parseParenExpr() (Expr, error) {
	p.lexer.Next() // consume '('
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.lexer.Peek().Type != TokenRParen {
		return nil, fmt.Errorf("%w: ')' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	return expr, nil
}

// parseNumberLiteral converts a numeric token value to a LiteralExpr.
// Per XPath 3.1: no dot and no e/E → xs:integer (*big.Int),
// dot but no e/E → xs:decimal (*big.Rat),
// e/E → xs:double (float64).
func parseNumberLiteral(s string) (LiteralExpr, error) {
	hasE := strings.ContainsAny(s, "eE")
	hasDot := strings.Contains(s, ".")
	if !hasDot && !hasE {
		// Integer literal → *big.Int (arbitrary precision)
		v, ok := new(big.Int).SetString(s, 10)
		if !ok {
			return LiteralExpr{}, fmt.Errorf("invalid integer %q", s)
		}
		return LiteralExpr{Value: v}, nil
	}
	if !hasE {
		// Decimal literal → *big.Rat (exact rational)
		v, ok := new(big.Rat).SetString(s)
		if !ok {
			return LiteralExpr{}, fmt.Errorf("invalid decimal %q", s)
		}
		return LiteralExpr{Value: v}, nil
	}
	// Double literal → float64
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return LiteralExpr{}, fmt.Errorf("invalid number %q: %w", s, err)
	}
	return LiteralExpr{Value: v}, nil
}

// parseArgumentList parses '(' (ExprSingle (',' ExprSingle)*)? ')'.
// ? in argument position produces PlaceholderExpr (partial application).
func (p *parser) parseArgumentList() ([]Expr, error) {
	if p.lexer.Peek().Type != TokenLParen {
		return nil, fmt.Errorf("%w: '(' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next() // consume '('
	var args []Expr
	if p.lexer.Peek().Type != TokenRParen {
		for {
			if p.lexer.Peek().Type == TokenQMark {
				// Disambiguate: ? followed by a key specifier (NCName, integer, *, '(')
				// is a unary lookup expression, not a placeholder.
				next := p.lexer.PeekAt(1).Type
				if next == TokenName || next == TokenNumber || next == TokenStar || next == TokenLParen {
					arg, err := p.parseExprSingle()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
				} else {
					p.lexer.Next()
					args = append(args, PlaceholderExpr{})
				}
			} else {
				arg, err := p.parseExprSingle()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
			}
			if p.lexer.Peek().Type != TokenComma {
				break
			}
			p.lexer.Next() // consume ','
		}
	}
	if p.lexer.Peek().Type != TokenRParen {
		return nil, fmt.Errorf("%w: ')' in argument list but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	return args, nil
}

// parseLookupKey parses the key after '?': NCName, integer, '*', or '(' expr ')'.
func (p *parser) parseLookupKey() (Expr, bool, error) {
	tok := p.lexer.Peek()
	switch tok.Type {
	case TokenStar:
		p.lexer.Next()
		return nil, true, nil
	case TokenNumber:
		// Only integer literals are valid lookup keys (no dot, no exponent)
		if strings.ContainsAny(tok.Value, ".eE") {
			return nil, false, fmt.Errorf("%w: integer lookup key but got %s", ErrExpectedToken, tok)
		}
		p.lexer.Next()
		n := new(big.Int)
		if _, ok := n.SetString(tok.Value, 10); !ok {
			return nil, false, fmt.Errorf("invalid lookup index %q", tok.Value)
		}
		return LiteralExpr{Value: n}, false, nil
	case TokenLParen:
		expr, err := p.parseParenExprOrEmpty()
		if err != nil {
			return nil, false, err
		}
		return expr, false, nil
	default:
		// Accept any keyword or name token as an NCName key
		if name := tokenAsNCName(tok); name != "" {
			p.lexer.Next()
			return LiteralExpr{Value: name}, false, nil
		}
		return nil, false, fmt.Errorf("%w: lookup key but got %s", ErrExpectedToken, tok)
	}
}

// tokenAsNCName returns the NCName string for a token that can be used as an
// NCName in lookup keys and other non-keyword contexts. Returns "" if the
// token is not a valid NCName.
func tokenAsNCName(tok Token) string {
	switch tok.Type {
	case TokenName:
		// Reject URIQualifiedNames (Q{uri}local) — they are not NCNames
		if strings.HasPrefix(tok.Value, "Q{") {
			return ""
		}
		// Reject prefixed names (prefix:local) — they are QNames, not NCNames
		if strings.Contains(tok.Value, ":") {
			return ""
		}
		return tok.Value
	case TokenDiv:
		return "div"
	case TokenMod:
		return "mod"
	case TokenAnd:
		return "and"
	case TokenOr:
		return "or"
	case TokenReturn:
		return "return"
	case TokenElse:
		return "else"
	case TokenEq:
		return "eq"
	case TokenNe:
		return "ne"
	case TokenLt:
		return "lt"
	case TokenLe:
		return "le"
	case TokenGt:
		return "gt"
	case TokenGe:
		return "ge"
	case TokenIdiv:
		return "idiv"
	case TokenIf:
		return "if"
	case TokenThen:
		return "then"
	case TokenFor:
		return "for"
	case TokenLet:
		return "let"
	case TokenIn:
		return "in"
	case TokenSome:
		return "some"
	case TokenEvery:
		return "every"
	case TokenSatisfies:
		return "satisfies"
	case TokenIs:
		return "is"
	case TokenTo:
		return "to"
	case TokenUnion:
		return "union"
	case TokenIntersect:
		return "intersect"
	case TokenExcept:
		return "except"
	case TokenInstanceOf:
		return "instance"
	case TokenTreatAs:
		return "treat"
	case TokenCastableAs:
		return "castable"
	case TokenCastAs:
		return "cast"
	case TokenAs:
		return "as"
	case TokenOf:
		return "of"
	}
	return ""
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
			if !p.looksLikeStep() {
				p.lexer.Backup() // put / back — caller handles non-step continuation
				break loop
			}
			s, err := p.parseStep()
			if err != nil {
				return nil, err
			}
			steps = append(steps, s)
		case TokenSlashSlash:
			p.lexer.Next()
			if !p.looksLikeStep() {
				p.lexer.Backup() // put // back — caller handles non-step continuation
				break loop
			}
			steps = append(steps, Step{
				Axis:     AxisDescendantOrSelf,
				NodeTest: TypeTest{Kind: NodeKindNode},
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
			NodeTest: TypeTest{Kind: NodeKindNode},
		}, nil
	}

	// Abbreviated step: ..
	if tok.Type == TokenDotDot {
		p.lexer.Next()
		return Step{
			Axis:     AxisParent,
			NodeTest: TypeTest{Kind: NodeKindNode},
		}, nil
	}

	// @ is short for attribute::
	axis := AxisChild
	switch tok.Type {
	case TokenAt:
		axis = AxisAttribute
		p.lexer.Next()
	case TokenName:
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

// parseNodeTest parses → NameTest | KindTest.
func (p *parser) parseNodeTest(_ AxisType) (NodeTest, error) {
	tok := p.lexer.Peek()

	// * wildcard or *:local
	if tok.Type == TokenStar {
		p.lexer.Next()
		if p.lexer.Peek().Type == TokenColon {
			p.lexer.Next() // consume ':'
			local := p.lexer.Peek()
			if local.Type == TokenName {
				p.lexer.Next()
				return NameTest{Prefix: "*", Local: local.Value}, nil
			}
			return nil, fmt.Errorf("expected local name after *")
		}
		return NameTest{Local: "*"}, nil
	}

	if tok.Type != TokenName {
		return nil, fmt.Errorf("%w: node test but got %s", ErrExpectedToken, tok)
	}

	p.lexer.Next()

	// Check if it's a kind test: node(), text(), element(), etc.
	if p.lexer.Peek().Type == TokenLParen {
		if nt, ok, err := p.parseKindTest(tok.Value); ok || err != nil {
			return nt, err
		}
		// Not a kind test — back up past the name
		p.lexer.Backup()
	}

	// Check for URIQualifiedName: Q{uri}local
	if strings.HasPrefix(tok.Value, "Q{") {
		if idx := strings.Index(tok.Value, "}"); idx >= 0 {
			uri := tok.Value[2:idx]
			local := tok.Value[idx+1:]
			if uri == "http://www.w3.org/2000/xmlns/" {
				return nil, &XPathError{Code: errCodeXPST0081, Message: "the xmlns namespace URI cannot be used in name tests"}
			}
			return NameTest{URI: uri, Local: local}, nil
		}
	}

	// Check for QName: prefix:local or prefix:*
	if p.lexer.Peek().Type == TokenColon {
		return p.parseQNameTest(tok.Value)
	}

	return NameTest{Local: tok.Value}, nil
}

// parseKindTest attempts to parse a kind test function.
// Returns (result, true, nil) on success, (nil, false, nil) when name is not a kind test keyword.
func (p *parser) parseKindTest(name string) (NodeTest, bool, error) {
	switch name {
	case "node":
		p.lexer.Next() // consume '('
		if err := p.expectToken(TokenRParen); err != nil {
			return nil, true, fmt.Errorf("expected ')' after node(")
		}
		return TypeTest{Kind: NodeKindNode}, true, nil
	case "text":
		p.lexer.Next()
		if err := p.expectToken(TokenRParen); err != nil {
			return nil, true, fmt.Errorf("expected ')' after text(")
		}
		return TypeTest{Kind: NodeKindText}, true, nil
	case "comment":
		p.lexer.Next()
		if err := p.expectToken(TokenRParen); err != nil {
			return nil, true, fmt.Errorf("expected ')' after comment(")
		}
		return TypeTest{Kind: NodeKindComment}, true, nil
	case "processing-instruction":
		p.lexer.Next()
		target := ""
		if p.lexer.Peek().Type == TokenString || p.lexer.Peek().Type == TokenName {
			target = p.lexer.Next().Value
		}
		if err := p.expectToken(TokenRParen); err != nil {
			return nil, true, fmt.Errorf("expected ')' after processing-instruction(")
		}
		return PITest{Target: target}, true, nil
	case "element":
		return p.parseElementOrAttributeTest(true)
	case "attribute":
		return p.parseElementOrAttributeTest(false)
	case "document-node":
		return p.parseDocumentNodeTest()
	case "schema-element":
		return p.parseSchemaTest(true)
	case "schema-attribute":
		return p.parseSchemaTest(false)
	case "namespace-node":
		p.lexer.Next()
		if err := p.expectToken(TokenRParen); err != nil {
			return nil, true, fmt.Errorf("expected ')' after namespace-node(")
		}
		return NamespaceNodeTest{}, true, nil
	case "item":
		p.lexer.Next()
		if err := p.expectToken(TokenRParen); err != nil {
			return nil, true, fmt.Errorf("expected ')' after item(")
		}
		return AnyItemTest{}, true, nil
	}
	return nil, false, nil
}

// parseElementOrAttributeTest parses element(name?, type?) or attribute(name?, type?).
func (p *parser) parseElementOrAttributeTest(isElement bool) (NodeTest, bool, error) {
	p.lexer.Next() // consume '('
	name := ""
	typeName := ""
	nillable := false

	if p.lexer.Peek().Type != TokenRParen {
		// Parse optional name
		if p.lexer.Peek().Type == TokenStar {
			p.lexer.Next()
			name = "*"
		} else if p.lexer.Peek().Type == TokenName {
			name = p.scanQName()
		}
		// Parse optional type
		if p.lexer.Peek().Type == TokenComma {
			p.lexer.Next()
			typeName = p.scanQName()
			// element(name, type?) — trailing ? means nillable
			if isElement && p.lexer.Peek().Type == TokenQMark {
				p.lexer.Next()
				nillable = true
			}
		}
	}
	if err := p.expectToken(TokenRParen); err != nil {
		if isElement {
			return nil, true, fmt.Errorf("expected ')' after element(")
		}
		return nil, true, fmt.Errorf("expected ')' after attribute(")
	}

	if isElement {
		return ElementTest{Name: name, TypeName: typeName, Nillable: nillable}, true, nil
	}
	return AttributeTest{Name: name, TypeName: typeName}, true, nil
}

// parseDocumentNodeTest parses document-node(element(...)?) or document-node().
func (p *parser) parseDocumentNodeTest() (NodeTest, bool, error) {
	p.lexer.Next() // consume '('
	var inner NodeTest
	if p.lexer.Peek().Type == TokenName && p.lexer.Peek().Value == "element" {
		p.lexer.Next() // consume 'element'
		nt, _, err := p.parseElementOrAttributeTest(true)
		if err != nil {
			return nil, true, err
		}
		inner = nt
	}
	if err := p.expectToken(TokenRParen); err != nil {
		return nil, true, fmt.Errorf("expected ')' after document-node(")
	}
	return DocumentTest{Inner: inner}, true, nil
}

// parseSchemaTest parses schema-element(name) or schema-attribute(name).
func (p *parser) parseSchemaTest(isElement bool) (NodeTest, bool, error) {
	p.lexer.Next() // consume '('
	name := p.scanQName()
	if err := p.expectToken(TokenRParen); err != nil {
		if isElement {
			return nil, true, fmt.Errorf("expected ')' after schema-element(")
		}
		return nil, true, fmt.Errorf("expected ')' after schema-attribute(")
	}
	if isElement {
		return SchemaElementTest{Name: name}, true, nil
	}
	return SchemaAttributeTest{Name: name}, true, nil
}

// parseQNameTest parses a prefix:local or prefix:* name test.
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

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.lexer.Peek().Type != TokenRBracket {
		return nil, fmt.Errorf("%w: ']' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	return expr, nil
}

// --- XPath 3.1 Expression Extensions ---

// parseFLWOR parses for/let expressions (XPath 3.1 simple for/let only).
// XPath 3.1 does not support full FLWOR — only a single for or let clause
// (with comma-separated bindings) followed by "return".
func (p *parser) parseFLWOR() (Expr, error) {
	var clauses []FLWORClause
	tok := p.lexer.Peek()
	switch tok.Type {
	case TokenFor:
		p.lexer.Next()
		fc, err := p.parseForBindings()
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, fc...)
	case TokenLet:
		p.lexer.Next()
		lc, err := p.parseLetBindings()
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, lc...)
	}
	if p.lexer.Peek().Type != TokenReturn {
		return nil, fmt.Errorf("%w: 'return' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	ret, err := p.parseExprSingle()
	if err != nil {
		return nil, err
	}
	return FLWORExpr{Clauses: clauses, Return: ret}, nil
}

// parseForBindings parses "for $var in expr (, $var in expr)*".
func (p *parser) parseForBindings() ([]FLWORClause, error) {
	var clauses []FLWORClause
	for {
		if p.lexer.Peek().Type != TokenVariableRef {
			return nil, fmt.Errorf("%w: variable after 'for' but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		varName := p.lexer.Next().Value
		// Optional positional variable: "at $pos"
		var posVar string
		if tok := p.lexer.Peek(); tok.Type == TokenName && tok.Value == "at" {
			p.lexer.Next()
			if p.lexer.Peek().Type != TokenVariableRef {
				return nil, fmt.Errorf("%w: variable after 'at' but got %s", ErrExpectedToken, p.lexer.Peek())
			}
			posVar = p.lexer.Next().Value
		}
		if p.lexer.Peek().Type != TokenIn {
			return nil, fmt.Errorf("%w: 'in' after variable but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next()
		expr, err := p.parseExprSingle()
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, ForClause{Var: varName, PosVar: posVar, Expr: expr})
		if p.lexer.Peek().Type != TokenComma {
			break
		}
		p.lexer.Next()
	}
	return clauses, nil
}

// parseLetBindings parses "let $var := expr (, $var := expr)*".
func (p *parser) parseLetBindings() ([]FLWORClause, error) {
	var clauses []FLWORClause
	for {
		if p.lexer.Peek().Type != TokenVariableRef {
			return nil, fmt.Errorf("%w: variable after 'let' but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		varName := p.lexer.Next().Value
		// Expect ':=' — colon then equals
		if p.lexer.Peek().Type != TokenColon {
			return nil, fmt.Errorf("%w: ':=' after variable but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next()
		if p.lexer.Peek().Type != TokenEquals {
			return nil, fmt.Errorf("%w: '=' after ':' in let binding but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next()
		expr, err := p.parseExprSingle()
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, LetClause{Var: varName, Expr: expr})
		if p.lexer.Peek().Type != TokenComma {
			break
		}
		p.lexer.Next()
	}
	return clauses, nil
}

// parseQuantifiedExpr parses "some/every $var in domain satisfies test".
func (p *parser) parseQuantifiedExpr(some bool) (Expr, error) {
	p.lexer.Next() // consume 'some' or 'every'
	var bindings []QuantifiedBinding
	for {
		if p.lexer.Peek().Type != TokenVariableRef {
			return nil, fmt.Errorf("%w: variable after 'some'/'every' but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		varName := p.lexer.Next().Value
		if p.lexer.Peek().Type != TokenIn {
			return nil, fmt.Errorf("%w: 'in' after variable but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next()
		domain, err := p.parseExprSingle()
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, QuantifiedBinding{Var: varName, Domain: domain})
		if p.lexer.Peek().Type != TokenComma {
			break
		}
		p.lexer.Next()
	}
	if p.lexer.Peek().Type != TokenSatisfies {
		return nil, fmt.Errorf("%w: 'satisfies' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	satisfies, err := p.parseExprSingle()
	if err != nil {
		return nil, err
	}
	return QuantifiedExpr{Some: some, Bindings: bindings, Satisfies: satisfies}, nil
}

// parseIfExpr parses "if (cond) then thenExpr else elseExpr".
func (p *parser) parseIfExpr() (Expr, error) {
	p.lexer.Next() // consume 'if'
	if p.lexer.Peek().Type != TokenLParen {
		return nil, fmt.Errorf("%w: '(' after 'if' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.lexer.Peek().Type != TokenRParen {
		return nil, fmt.Errorf("%w: ')' after if condition but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	if p.lexer.Peek().Type != TokenThen {
		return nil, fmt.Errorf("%w: 'then' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	thenExpr, err := p.parseExprSingle()
	if err != nil {
		return nil, err
	}
	if p.lexer.Peek().Type != TokenElse {
		return nil, fmt.Errorf("%w: 'else' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	elseExpr, err := p.parseExprSingle()
	if err != nil {
		return nil, err
	}
	return IfExpr{Cond: cond, Then: thenExpr, Else: elseExpr}, nil
}

// parseTryCatchExpr parses "try { expr } catch code { expr }".
func (p *parser) parseTryCatchExpr() (Expr, error) {
	p.lexer.Next() // consume 'try'
	if p.lexer.Peek().Type != TokenLBrace {
		return nil, fmt.Errorf("%w: '{' after 'try' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	tryExpr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.lexer.Peek().Type != TokenRBrace {
		return nil, fmt.Errorf("%w: '}' after try body but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()

	var catches []CatchClause
	for p.lexer.Peek().Type == TokenCatch {
		p.lexer.Next()
		// Parse error codes (NameTest ("|" NameTest)*)
		// NameTest ::= EQName | Wildcard
		// Wildcard ::= "*" | NCName ":*" | "*:" NCName | "BracedURILiteral" "*"
		var codes []string
		for {
			code, err := p.parseCatchCode()
			if err != nil {
				return nil, err
			}
			codes = append(codes, code)
			if p.lexer.Peek().Type != TokenPipe {
				break
			}
			p.lexer.Next()
		}
		if p.lexer.Peek().Type != TokenLBrace {
			return nil, fmt.Errorf("%w: '{' after catch but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next()
		catchExpr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.lexer.Peek().Type != TokenRBrace {
			return nil, fmt.Errorf("%w: '}' after catch body but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next()
		catches = append(catches, CatchClause{Codes: codes, Expr: catchExpr})
	}
	if len(catches) == 0 {
		return nil, fmt.Errorf("%w: at least one 'catch' clause", ErrExpectedToken)
	}
	return TryCatchExpr{Try: tryExpr, Catches: catches}, nil
}

// parseCatchCode parses a single catch error code.
// Supports: "*", "prefix:local", "prefix:*", "*:local", "Q{uri}local", "Q{uri}*"
func (p *parser) parseCatchCode() (string, error) {
	tok := p.lexer.Peek()

	// * or *:local
	if tok.Type == TokenStar {
		p.lexer.Next()
		if p.lexer.Peek().Type == TokenColon {
			p.lexer.Next()
			localTok := p.lexer.Peek()
			if localTok.Type == TokenName {
				p.lexer.Next()
				return "*:" + localTok.Value, nil
			}
			p.lexer.Backup() // put ':' back
		}
		return "*", nil
	}

	// Q{uri}local or Q{uri}* (already scanned as single TokenName by lexer)
	if tok.Type == TokenName && len(tok.Value) > 2 && tok.Value[0] == 'Q' && tok.Value[1] == '{' {
		p.lexer.Next()
		// Check if next token is * for Q{uri}*
		if p.lexer.Peek().Type == TokenStar {
			p.lexer.Next()
			return tok.Value + "*", nil
		}
		return tok.Value, nil
	}

	// NCName or NCName:NCName or NCName:*
	if tok.Type == TokenName {
		p.lexer.Next()
		name := tok.Value
		if p.lexer.Peek().Type == TokenColon {
			p.lexer.Next()
			next := p.lexer.Peek()
			if next.Type == TokenStar {
				p.lexer.Next()
				return name + ":*", nil
			}
			if next.Type == TokenName {
				p.lexer.Next()
				return name + ":" + next.Value, nil
			}
			p.lexer.Backup() // put ':' back
		}
		return name, nil
	}

	// "catch" keyword may appear as a name in some contexts
	if tok.Type == TokenCatch {
		p.lexer.Next()
		return tok.Value, nil
	}

	return "", fmt.Errorf("%w: error code in catch clause but got %s", ErrExpectedToken, tok)
}

// --- Constructor Parsing ---

// parseFunctionKeyword handles "function" which could be:
// - inline function: function($x) { ... }
// - function test: function(*) in sequence types
func (p *parser) parseFunctionKeyword() (Expr, error) {
	p.lexer.Next() // consume 'function'
	if p.lexer.Peek().Type != TokenLParen {
		return nil, fmt.Errorf("%w: '(' after 'function' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next() // consume '('

	// Check for function(*) — wildcard function test used in sequence types
	if p.lexer.Peek().Type == TokenStar {
		p.lexer.Next()
		if err := p.expectToken(TokenRParen); err != nil {
			return nil, fmt.Errorf("expected ')' after function(*")
		}
		// This is a FunctionTest, not an expression — should only appear in sequence types.
		// Return a dummy expression; the parser will handle this in sequence type context.
		return nil, fmt.Errorf("%w: function(*) outside sequence type", ErrUnexpectedToken)
	}

	// Inline function: function($param as type, ...) as returnType { body }
	var params []FunctionParam
	if p.lexer.Peek().Type != TokenRParen {
		for {
			if p.lexer.Peek().Type != TokenVariableRef {
				return nil, fmt.Errorf("%w: parameter variable but got %s", ErrExpectedToken, p.lexer.Peek())
			}
			paramName := p.lexer.Next().Value
			param := FunctionParam{Name: paramName}
			if p.lexer.Peek().Type == TokenAs {
				p.lexer.Next()
				st, err := p.parseSequenceType()
				if err != nil {
					return nil, err
				}
				param.TypeHint = &st
			}
			params = append(params, param)
			if p.lexer.Peek().Type != TokenComma {
				break
			}
			p.lexer.Next()
		}
	}
	if err := p.expectToken(TokenRParen); err != nil {
		return nil, fmt.Errorf("expected ')' in inline function parameters")
	}

	// Check for duplicate parameter names (XQST0039)
	seen := make(map[string]bool, len(params))
	for _, param := range params {
		if seen[param.Name] {
			return nil, fmt.Errorf("XQST0039: duplicate parameter name $%s in inline function", param.Name)
		}
		seen[param.Name] = true
	}

	var returnType *SequenceType
	if p.lexer.Peek().Type == TokenAs {
		p.lexer.Next()
		st, err := p.parseSequenceType()
		if err != nil {
			return nil, err
		}
		returnType = &st
	}

	if p.lexer.Peek().Type != TokenLBrace {
		return nil, fmt.Errorf("%w: '{' for function body but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()

	var body Expr
	if p.lexer.Peek().Type == TokenRBrace {
		// Empty function body: function(){} returns empty sequence
		body = SequenceExpr{Items: nil}
	} else {
		var err error
		body, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}
	if p.lexer.Peek().Type != TokenRBrace {
		return nil, fmt.Errorf("%w: '}' after function body but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()

	return InlineFunctionExpr{Params: params, ReturnType: returnType, Body: body}, nil
}

// parseMapConstructor parses "map { key: value, ... }".
func (p *parser) parseMapConstructor() (Expr, error) {
	p.lexer.Next() // consume 'map'
	if p.lexer.Peek().Type != TokenLBrace {
		return nil, fmt.Errorf("%w: '{' after 'map' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()

	var pairs []MapConstructorPair
	if p.lexer.Peek().Type != TokenRBrace {
		for {
			key, err := p.parseExprSingle()
			if err != nil {
				return nil, err
			}
			if p.lexer.Peek().Type != TokenColon {
				return nil, fmt.Errorf("%w: ':' in map entry but got %s", ErrExpectedToken, p.lexer.Peek())
			}
			p.lexer.Next()
			value, err := p.parseExprSingle()
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, MapConstructorPair{Key: key, Value: value})
			if p.lexer.Peek().Type != TokenComma {
				break
			}
			p.lexer.Next()
		}
	}
	if p.lexer.Peek().Type != TokenRBrace {
		return nil, fmt.Errorf("%w: '}' after map constructor but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	return MapConstructorExpr{Pairs: pairs}, nil
}

// parseArrayCurlyConstructor parses "array { expr }".
func (p *parser) parseArrayCurlyConstructor() (Expr, error) {
	p.lexer.Next() // consume 'array'
	if p.lexer.Peek().Type != TokenLBrace {
		return nil, fmt.Errorf("%w: '{' after 'array' but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	var items []Expr
	if p.lexer.Peek().Type != TokenRBrace {
		// array { expr } — the expr is a single sequence expression
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		items = []Expr{expr}
	}
	if p.lexer.Peek().Type != TokenRBrace {
		return nil, fmt.Errorf("%w: '}' after array constructor but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	return ArrayConstructorExpr{Items: items, SquareBracket: false}, nil
}

// parseArraySquareConstructor parses "[item1, item2, ...]".
func (p *parser) parseArraySquareConstructor() (Expr, error) {
	p.lexer.Next() // consume '['
	var items []Expr
	if p.lexer.Peek().Type != TokenRBracket {
		for {
			item, err := p.parseExprSingle()
			if err != nil {
				return nil, err
			}
			items = append(items, item)
			if p.lexer.Peek().Type != TokenComma {
				break
			}
			p.lexer.Next()
		}
	}
	if p.lexer.Peek().Type != TokenRBracket {
		return nil, fmt.Errorf("%w: ']' after array constructor but got %s", ErrExpectedToken, p.lexer.Peek())
	}
	p.lexer.Next()
	return ArrayConstructorExpr{Items: items, SquareBracket: true}, nil
}

// --- Sequence Type Parsing ---

// parseSequenceType parses a sequence type (used in instance-of, treat-as, etc.).
func (p *parser) parseSequenceType() (SequenceType, error) {
	tok := p.lexer.Peek()

	// empty-sequence()
	if tok.Type == TokenName && tok.Value == "empty-sequence" {
		p.lexer.Next()
		if err := p.expectToken(TokenLParen); err != nil {
			return SequenceType{}, fmt.Errorf("expected '(' after empty-sequence")
		}
		if err := p.expectToken(TokenRParen); err != nil {
			return SequenceType{}, fmt.Errorf("expected ')' after empty-sequence(")
		}
		return SequenceType{Void: true}, nil
	}

	// Parse item type
	itemTest, err := p.parseItemType()
	if err != nil {
		return SequenceType{}, err
	}

	// Parse optional occurrence indicator
	occ := OccurrenceExactlyOne
	switch p.lexer.Peek().Type {
	case TokenQMark:
		p.lexer.Next()
		occ = OccurrenceZeroOrOne
	case TokenStar:
		p.lexer.Next()
		occ = OccurrenceZeroOrMore
	case TokenPlus:
		p.lexer.Next()
		occ = OccurrenceOneOrMore
	}

	return SequenceType{ItemTest: itemTest, Occurrence: occ}, nil
}

// parseItemType parses an item type within a sequence type.
func (p *parser) parseItemType() (NodeTest, error) {
	tok := p.lexer.Peek()

	if tok.Type == TokenName {
		// Could be: item(), node(), element(), attribute(), xs:integer, etc.
		if p.lexer.PeekAt(1).Type == TokenLParen {
			p.lexer.Next() // consume name
			nt, ok, err := p.parseKindTest(tok.Value)
			if err != nil {
				return nil, err
			}
			if ok {
				return nt, nil
			}
			// Not a kind test — treat as function test or atomic type
			p.lexer.Backup()
		}
		// Atomic or union type name (possibly prefixed)
		return p.parseAtomicOrUnionType()
	}

	if tok.Type == TokenFunction {
		p.lexer.Next()
		if p.lexer.Peek().Type != TokenLParen {
			return nil, fmt.Errorf("%w: '(' after 'function' but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next()
		// function(*) - any function test
		if p.lexer.Peek().Type == TokenStar {
			p.lexer.Next()
			if err := p.expectToken(TokenRParen); err != nil {
				return nil, fmt.Errorf("expected ')' after function(*")
			}
			return FunctionTest{AnyFunction: true}, nil
		}
		// function() as ReturnType or function(ParamTypes) as ReturnType
		var paramTypes []SequenceType
		if p.lexer.Peek().Type != TokenRParen {
			for {
				st, err := p.parseSequenceType()
				if err != nil {
					return nil, err
				}
				paramTypes = append(paramTypes, st)
				if p.lexer.Peek().Type != TokenComma {
					break
				}
				p.lexer.Next()
			}
		}
		if err := p.expectToken(TokenRParen); err != nil {
			return nil, fmt.Errorf("expected ')' in function type test")
		}
		var returnType SequenceType
		if p.lexer.Peek().Type == TokenAs {
			p.lexer.Next()
			rt, err := p.parseSequenceType()
			if err != nil {
				return nil, err
			}
			returnType = rt
		}
		return FunctionTest{ParamTypes: paramTypes, ReturnType: returnType}, nil
	}

	if tok.Type == TokenMap {
		p.lexer.Next()
		if p.lexer.Peek().Type != TokenLParen {
			return nil, fmt.Errorf("%w: '(' after 'map' but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next()
		// map(*) or map(KeyType, ValueType)
		if p.lexer.Peek().Type == TokenStar {
			p.lexer.Next() // consume *
			if err := p.expectToken(TokenRParen); err != nil {
				return nil, fmt.Errorf("expected ')' after map(*")
			}
			return MapTest{AnyType: true}, nil
		}
		// Parse key type (must be an atomic type)
		keyType, err := p.parseAtomicOrUnionType()
		if err != nil {
			return nil, err
		}
		if p.lexer.Peek().Type != TokenComma {
			return nil, fmt.Errorf("%w: ',' in map(K, V) type test", ErrExpectedToken)
		}
		p.lexer.Next() // consume comma
		// Parse value sequence type
		valType, err := p.parseSequenceType()
		if err != nil {
			return nil, err
		}
		if err := p.expectToken(TokenRParen); err != nil {
			return nil, fmt.Errorf("expected ')' after map type test")
		}
		return MapTest{KeyType: keyType, ValType: valType}, nil
	}

	if tok.Type == TokenArray {
		p.lexer.Next()
		if p.lexer.Peek().Type != TokenLParen {
			return nil, fmt.Errorf("%w: '(' after 'array' but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next()
		// array(*) or array(MemberType)
		if p.lexer.Peek().Type == TokenStar {
			p.lexer.Next() // consume *
			if err := p.expectToken(TokenRParen); err != nil {
				return nil, fmt.Errorf("expected ')' after array(*")
			}
			return ArrayTest{AnyType: true}, nil
		}
		// Parse member sequence type
		memberType, err := p.parseSequenceType()
		if err != nil {
			return nil, err
		}
		if err := p.expectToken(TokenRParen); err != nil {
			return nil, fmt.Errorf("expected ')' after array type test")
		}
		return ArrayTest{MemberType: memberType}, nil
	}

	return nil, fmt.Errorf("%w: item type but got %s", ErrExpectedToken, tok)
}

// parseAtomicOrUnionType parses an atomic/union type name like xs:integer.
func (p *parser) parseAtomicOrUnionType() (NodeTest, error) {
	name := p.scanQName()
	if name == "" {
		return nil, fmt.Errorf("%w: type name", ErrExpectedToken)
	}
	// Split prefix:local
	prefix, local := splitQName(name)
	return AtomicOrUnionType{Prefix: prefix, Name: local}, nil
}

// parseSingleType parses an atomic type name with optional '?' (for cast/castable).
func (p *parser) parseSingleType() (AtomicTypeName, bool, error) {
	name := p.scanQName()
	if name == "" {
		return AtomicTypeName{}, false, fmt.Errorf("%w: type name in cast", ErrExpectedToken)
	}
	prefix, local := splitQName(name)
	allowEmpty := false
	if p.lexer.Peek().Type == TokenQMark {
		p.lexer.Next()
		allowEmpty = true
	}
	return AtomicTypeName{Prefix: prefix, Name: local}, allowEmpty, nil
}

// --- Helpers ---

// scanQName scans a possibly-prefixed QName (prefix:local) from the token stream.
func (p *parser) scanQName() string {
	tok := p.lexer.Peek()
	if tok.Type != TokenName {
		return ""
	}
	p.lexer.Next()
	name := tok.Value
	if p.lexer.Peek().Type == TokenColon {
		p.lexer.Next()
		localTok := p.lexer.Peek()
		if localTok.Type == TokenName {
			p.lexer.Next()
			return name + ":" + localTok.Value
		}
		p.lexer.Backup() // put ':' back
	}
	return name
}

// splitQName splits "prefix:local" into (prefix, local) or ("", name).
func splitQName(qname string) (string, string) {
	for i, c := range qname {
		if c == ':' {
			return qname[:i], qname[i+1:]
		}
	}
	return "", qname
}

// expectToken consumes the next token if it matches the expected type, or returns an error.
func (p *parser) expectToken(expected TokenType) error {
	if p.lexer.Peek().Type != expected {
		return fmt.Errorf("%w: %s but got %s", ErrExpectedToken, expected, p.lexer.Peek())
	}
	p.lexer.Next()
	return nil
}

// looksLikeStep returns true if the next token(s) look like the start of a location step.
func (p *parser) looksLikeStep() bool {
	tok := p.lexer.Peek()
	switch tok.Type {
	case TokenDotDot, TokenAt, TokenStar:
		return true
	case TokenName:
		p.lexer.Next()
		next := p.lexer.Peek()
		p.lexer.Backup()

		if next.Type == TokenColonColon {
			return true // axis::
		}
		if next.Type == TokenLParen {
			// node(), text(), comment(), processing-instruction(),
			// element(), attribute(), document-node(), schema-element(),
			// schema-attribute(), namespace-node() are steps
			switch tok.Value {
			case "node", "text", "comment", "processing-instruction",
				"element", "attribute", "document-node", "schema-element",
				"schema-attribute", "namespace-node":
				return true
			}
			return false // function call
		}
		if next.Type == TokenColon {
			// prefix:local (name test) vs prefix:name( (QName function call)
			// vs prefix:name# (named function ref)
			p.lexer.Next() // consume prefix
			p.lexer.Next() // consume ':'
			localTok := p.lexer.Peek()
			if localTok.Type == TokenName {
				p.lexer.Next()
				afterLocal := p.lexer.Peek()
				p.lexer.Backup() // local name
				p.lexer.Backup() // ':'
				p.lexer.Backup() // prefix
				if afterLocal.Type == TokenLParen {
					return false // QName function call
				}
				if afterLocal.Type == TokenHash {
					return false // named function ref
				}
				return true // QName step
			}
			p.lexer.Backup() // ':'
			p.lexer.Backup() // prefix
			return true
		}
		if next.Type == TokenHash {
			return false // named function ref: name#arity
		}
		return true // plain name test
	}
	return false
}

func isGeneralComp(t TokenType) bool {
	return t == TokenEquals || t == TokenNotEquals ||
		t == TokenLess || t == TokenLessEq ||
		t == TokenGreater || t == TokenGreaterEq
}

func isValueComp(t TokenType) bool {
	return t == TokenEq || t == TokenNe ||
		t == TokenLt || t == TokenLe ||
		t == TokenGt || t == TokenGe
}

func isNodeComp(t TokenType) bool {
	return t == TokenIs || t == TokenNodePre || t == TokenNodeFol
}
