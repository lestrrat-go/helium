package relaxng

import helium "github.com/lestrrat-go/helium"

type compileConfig struct {
	filename     string // RNG filename for error messages
	errorHandler helium.ErrorHandler
}

type validateConfig struct {
	filename     string
	errorHandler helium.ErrorHandler
}
