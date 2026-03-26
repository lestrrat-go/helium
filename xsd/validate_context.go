package xsd

import (
	"context"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

type validationContext struct {
	ctx      context.Context
	schema   *Schema
	cfg      *validateConfig
	filename string
	out      *strings.Builder
	errors   []ValidationError
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

// addValidityError writes a formatted validity error to the output buffer,
// delivers it to the ErrorHandler (if set), and appends a structured error.
func (vc *validationContext) addValidityError(file string, line int, elemName, msg string) {
	errStr := validityError(file, line, elemName, msg)
	vc.out.WriteString(errStr)
	vc.errors = append(vc.errors, ValidationError{
		Filename: file,
		Line:     line,
		Element:  elemName,
		Message:  msg,
	})
	if vc.cfg != nil && vc.cfg.errorHandler != nil {
		vc.cfg.errorHandler.Handle(vc.ctx, helium.NewLeveledError(errStr, helium.ErrorLevelError))
	}
}

// addValidityErrorAttr writes a formatted attribute validity error to the
// output buffer, delivers it to the ErrorHandler (if set), and appends a
// structured error.
func (vc *validationContext) addValidityErrorAttr(file string, line int, elemName, attrName, msg string) {
	errStr := validityErrorAttr(file, line, elemName, attrName, msg)
	vc.out.WriteString(errStr)
	vc.errors = append(vc.errors, ValidationError{
		Filename:  file,
		Line:      line,
		Element:   elemName,
		Attribute: attrName,
		Message:   msg,
	})
	if vc.cfg != nil && vc.cfg.errorHandler != nil {
		vc.cfg.errorHandler.Handle(vc.ctx, helium.NewLeveledError(errStr, helium.ErrorLevelError))
	}
}
