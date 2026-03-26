package xsd

import (
	"context"

	helium "github.com/lestrrat-go/helium"
)

type validationContext struct {
	ctx           context.Context
	schema        *Schema
	cfg           *validateConfig
	filename      string
	errorHandler  helium.ErrorHandler
	suppressDepth int
}

func newValidationContext(ctx context.Context, schema *Schema, cfg *validateConfig, filename string, handler helium.ErrorHandler) *validationContext {
	if ctx == nil {
		ctx = context.Background()
	}
	if handler == nil {
		handler = helium.NilErrorHandler{}
	}
	return &validationContext{
		ctx:          ctx,
		schema:       schema,
		cfg:          cfg,
		filename:     filename,
		errorHandler: handler,
	}
}

// validationErrors is a synchronous ErrorHandler that accumulates error
// strings in order. Used internally by ValidateElement and tests.
type validationErrors struct {
	errors []string
}

func (ve *validationErrors) Handle(_ context.Context, err error) {
	ve.errors = append(ve.errors, err.Error())
}

// reportValidityError formats a validation error and sends it to the ErrorHandler.
func (vc *validationContext) reportValidityError(file string, line int, elemName, msg string) {
	if vc.suppressDepth > 0 {
		return
	}
	errStr := validityError(file, line, elemName, msg)
	vc.errorHandler.Handle(vc.ctx, helium.NewLeveledError(errStr, helium.ErrorLevelError))
}

// reportValidityErrorAttr formats an attribute validation error and sends it to the ErrorHandler.
func (vc *validationContext) reportValidityErrorAttr(file string, line int, elemName, attrName, msg string) {
	if vc.suppressDepth > 0 {
		return
	}
	errStr := validityErrorAttr(file, line, elemName, attrName, msg)
	vc.errorHandler.Handle(vc.ctx, helium.NewLeveledError(errStr, helium.ErrorLevelError))
}
