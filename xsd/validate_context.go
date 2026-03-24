package xsd

import (
	"context"
	"strings"
)

type validationContext struct {
	ctx      context.Context
	schema   *Schema
	cfg      *validateConfig
	filename string
	out      *strings.Builder
}

func newValidationContext(ctx context.Context, schema *Schema, cfg *validateConfig, filename string, out *strings.Builder) *validationContext {
	if ctx == nil {
		ctx = context.Background()
	}
	return &validationContext{
		ctx:      ctx,
		schema:   schema,
		cfg:      cfg,
		filename: filename,
		out:      out,
	}
}
