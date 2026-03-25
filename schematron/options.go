package schematron

import (
	helium "github.com/lestrrat-go/helium"
)

type compileConfig struct {
	filename     string
	errorHandler helium.ErrorHandler
}

type validateConfig struct {
	filename     string
	quiet        bool
	errorHandler ErrorHandler
}

// ErrorHandler receives individual schematron validation errors.
type ErrorHandler interface {
	HandleError(e ValidationError)
}

// ErrorHandlerFunc is a function adapter for ErrorHandler.
type ErrorHandlerFunc func(ValidationError)

// HandleError implements ErrorHandler.
func (f ErrorHandlerFunc) HandleError(e ValidationError) { f(e) }

// ValidationError represents a single schematron validation error.
type ValidationError struct {
	Filename string
	Line     int
	Element  string
	Path     string
	Message  string
}
