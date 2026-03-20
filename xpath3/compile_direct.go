package xpath3

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/internal/lexicon"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

func compileFromLexer(l *lexer) (*vmProgram, prefixValidationPlan, error) {
	if program, prefixPlan, ok, err := tryCompileDirectFromLexer(l); ok || err != nil {
		return program, prefixPlan, err
	}

	l.idx = 0
	ast, err := parseWithLexer(l)
	if err != nil {
		return nil, prefixValidationPlan{}, err
	}
	return compileOwnedVMProgram(ast)
}

func parseWithLexer(l *lexer) (Expr, error) {
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

func tryCompileDirectFromLexer(l *lexer) (*vmProgram, prefixValidationPlan, bool, error) {
	p := &parser{lexer: l}
	expr, ok, err := p.tryParseDirectExpr()
	if err != nil || !ok {
		return nil, prefixValidationPlan{}, ok, err
	}
	if tok := l.Peek(); tok.Type != TokenEOF {
		return nil, prefixValidationPlan{}, false, nil
	}

	program, prefixPlan, err := compileOwnedVMProgram(expr)
	if err != nil {
		return nil, prefixValidationPlan{}, false, err
	}
	return program, prefixPlan, true, nil
}

func (p *parser) tryParseDirectExpr() (Expr, bool, error) {
	if l, ok := p.lexer.(*lexer); ok {
		start := l.idx
		expr, ok, err := p.tryParseDirectLocationPath()
		if ok || err != nil {
			return expr, ok, err
		}
		l.idx = start

		expr, ok, err = p.tryParseDirectFunctionCall()
		if ok || err != nil {
			return expr, ok, err
		}
		l.idx = start
	}
	return nil, false, nil
}

func (p *parser) tryParseDirectFunctionCall() (Expr, bool, error) {
	tok := p.lexer.Peek()
	if !isNameLikeToken(tok.Type) {
		return nil, false, nil
	}

	p.lexer.Next()
	prefix := ""
	name := tok.Value
	if p.lexer.Peek().Type == TokenColon {
		p.lexer.Next()
		localTok := p.lexer.Peek()
		if !isNameLikeToken(localTok.Type) {
			return nil, false, nil
		}
		p.lexer.Next()
		prefix = name
		name = localTok.Value
	}

	if p.lexer.Peek().Type != TokenLParen {
		return nil, false, nil
	}
	p.lexer.Next()

	if p.lexer.Peek().Type == TokenRParen {
		p.lexer.Next()
		return FunctionCall{Prefix: prefix, Name: name}, true, nil
	}

	arg, ok, err := p.tryParseDirectLocationPath()
	if err != nil || !ok {
		return nil, ok, err
	}
	if p.lexer.Peek().Type != TokenRParen {
		return nil, false, nil
	}
	p.lexer.Next()
	return FunctionCall{Prefix: prefix, Name: name, Args: []Expr{arg}}, true, nil
}

func (p *parser) tryParseDirectLocationPath() (Expr, bool, error) {
	tok := p.lexer.Peek()
	absolute := false
	steps := make([]vmLocationStep, 0, 4)

	switch tok.Type {
	case TokenSlash:
		absolute = true
		p.lexer.Next()
	case TokenSlashSlash:
		absolute = true
		p.lexer.Next()
		steps = append(steps, vmLocationStep{
			Axis:     AxisDescendantOrSelf,
			NodeTest: TypeTest{Kind: NodeKindNode},
		})
	default:
		if !looksLikeDirectStepStart(tok) {
			return nil, false, nil
		}
	}

	if !looksLikeDirectStepStart(p.lexer.Peek()) {
		if absolute {
			return vmLocationPathExpr{Absolute: true, Steps: steps}, true, nil
		}
		return nil, false, nil
	}

	step, ok, err := p.tryParseDirectStep()
	if err != nil || !ok {
		return nil, ok, err
	}
	steps = append(steps, step)

	for {
		switch p.lexer.Peek().Type {
		case TokenSlash:
			p.lexer.Next()
			step, ok, err := p.tryParseDirectStep()
			if err != nil || !ok {
				return nil, ok, err
			}
			steps = append(steps, step)
		case TokenSlashSlash:
			p.lexer.Next()
			step, ok, err := p.tryParseDirectStep()
			if err != nil || !ok {
				return nil, ok, err
			}
			steps = append(steps, vmLocationStep{
				Axis:     AxisDescendantOrSelf,
				NodeTest: TypeTest{Kind: NodeKindNode},
			})
			steps = append(steps, step)
		default:
			return vmLocationPathExpr{Absolute: absolute, Steps: steps}, true, nil
		}
	}
}

func looksLikeDirectStepStart(tok Token) bool {
	switch tok.Type {
	case TokenDot, TokenDotDot, TokenAt, TokenStar:
		return true
	default:
		return isNameLikeToken(tok.Type)
	}
}

func (p *parser) tryParseDirectStep() (vmLocationStep, bool, error) {
	tok := p.lexer.Peek()

	switch tok.Type {
	case TokenDot:
		p.lexer.Next()
		return vmLocationStep{Axis: AxisSelf, NodeTest: TypeTest{Kind: NodeKindNode}}, true, nil
	case TokenDotDot:
		p.lexer.Next()
		return vmLocationStep{Axis: AxisParent, NodeTest: TypeTest{Kind: NodeKindNode}}, true, nil
	}

	axis := AxisChild
	if tok.Type == TokenAt {
		axis = AxisAttribute
		p.lexer.Next()
	}

	if axis == AxisChild && isNameLikeToken(tok.Type) {
		if p.lexer.PeekAt(1).Type == TokenColonColon {
			a, ok := ixpath.AxisFromName(tok.Value)
			if !ok {
				return vmLocationStep{}, false, fmt.Errorf("%w: %q", ErrUnknownAxis, tok.Value)
			}
			axis = a
			p.lexer.Next()
			p.lexer.Next()
		}
	}

	nodeTest, ok, err := p.tryParseDirectNodeTest(axis)
	if err != nil || !ok {
		return vmLocationStep{}, ok, err
	}

	predicates, err := p.parseDirectPredicates()
	if err != nil {
		return vmLocationStep{}, true, err
	}

	return vmLocationStep{
		Axis:       axis,
		NodeTest:   nodeTest,
		Predicates: predicates,
	}, true, nil
}

func (p *parser) tryParseDirectNodeTest(axis AxisType) (NodeTest, bool, error) {
	tok := p.lexer.Peek()

	switch tok.Type {
	case TokenStar:
		if p.lexer.PeekAt(1).Type == TokenColon {
			localTok := p.lexer.PeekAt(2)
			if !isNameLikeToken(localTok.Type) {
				return nil, false, nil
			}
			p.lexer.Next()
			p.lexer.Next()
			p.lexer.Next()
			return NameTest{Prefix: "*", Local: localTok.Value}, true, nil
		}
		p.lexer.Next()
		return NameTest{Local: "*"}, true, nil
	default:
		if !isNameLikeToken(tok.Type) {
			return nil, false, nil
		}
	}

	if p.lexer.PeekAt(1).Type == TokenLParen {
		return nil, false, nil
	}

	if strings.HasPrefix(tok.Value, "Q{") {
		idx := strings.Index(tok.Value, "}")
		if idx < 0 {
			return nil, false, fmt.Errorf("%w: malformed URIQualifiedName %q", ErrUnexpectedToken, tok.Value)
		}
		uri := tok.Value[2:idx]
		local := tok.Value[idx+1:]
		if uri == lexicon.XMLNS {
			return nil, false, &XPathError{Code: errCodeXPST0081, Message: "the xmlns namespace URI cannot be used in name tests"}
		}
		p.lexer.Next()
		return NameTest{URI: uri, Local: local}, true, nil
	}

	if p.lexer.PeekAt(1).Type == TokenColon {
		next := p.lexer.PeekAt(2)
		if next.Type == TokenStar {
			p.lexer.Next()
			p.lexer.Next()
			p.lexer.Next()
			return NameTest{Prefix: tok.Value, Local: "*"}, true, nil
		}
		if !isNameLikeToken(next.Type) {
			return nil, false, nil
		}
		p.lexer.Next()
		p.lexer.Next()
		p.lexer.Next()
		return NameTest{Prefix: tok.Value, Local: next.Value}, true, nil
	}

	p.lexer.Next()
	if axis == AxisAttribute {
		return NameTest{Local: tok.Value}, true, nil
	}
	return NameTest{Local: tok.Value}, true, nil
}

func (p *parser) parseDirectPredicates() ([]Expr, error) {
	var predicates []Expr
	if p.lexer.Peek().Type == TokenLBracket {
		predicates = make([]Expr, 0, 2)
	}
	for p.lexer.Peek().Type == TokenLBracket {
		p.lexer.Next()
		predStart := -1
		if l, ok := p.lexer.(*lexer); ok {
			predStart = l.idx
		}

		pred, ok, err := p.tryParseDirectPredicateExpr()
		if err != nil {
			return nil, err
		}
		if !ok || p.lexer.Peek().Type != TokenRBracket {
			if predStart >= 0 {
				p.lexer.(*lexer).idx = predStart
			}
			pred, err = p.parseExpression()
			if err != nil {
				return nil, err
			}
		}

		if p.lexer.Peek().Type != TokenRBracket {
			return nil, fmt.Errorf("%w: ']' but got %s", ErrExpectedToken, p.lexer.Peek())
		}
		p.lexer.Next()
		predicates = append(predicates, pred)
	}
	return predicates, nil
}

func (p *parser) tryParseDirectPredicateExpr() (Expr, bool, error) {
	l, ok := p.lexer.(*lexer)
	if !ok {
		return nil, false, nil
	}

	start := l.idx

	left, ok, err := p.tryParseDirectPredicateOperand()
	if err != nil || !ok {
		l.idx = start
		return nil, ok, err
	}

	op := p.lexer.Peek().Type
	if !isDirectPredicateOp(op) {
		return left, true, nil
	}
	p.lexer.Next()

	right, ok, err := p.tryParseDirectPredicateOperand()
	if err != nil || !ok {
		l.idx = start
		return nil, ok, err
	}
	return BinaryExpr{Op: op, Left: left, Right: right}, true, nil
}

func (p *parser) tryParseDirectPredicateOperand() (Expr, bool, error) {
	tok := p.lexer.Peek()
	switch tok.Type {
	case TokenString:
		p.lexer.Next()
		return LiteralExpr{Value: tok.Value}, true, nil
	case TokenNumber:
		p.lexer.Next()
		n, err := parseNumberLiteral(tok.Value)
		if err != nil {
			return nil, false, err
		}
		return n, true, nil
	case TokenVariableRef:
		p.lexer.Next()
		return variableExprFromToken(tok.Value), true, nil
	}
	return p.tryParseDirectLocationPath()
}

func isDirectPredicateOp(t TokenType) bool {
	return isGeneralComp(t) || isValueComp(t)
}
