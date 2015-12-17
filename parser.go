package helium

func (p *Parser) Parse(b []byte) (*Document, error) {
	ctx := &parserCtx{}
	ctx.init(b)
	defer ctx.release()
	if err := ctx.parse(); err != nil {
		return nil, err
	}

	return ctx.doc, nil
}

