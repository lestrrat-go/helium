package relaxng

import helium "github.com/lestrrat-go/helium"

type compilerCfg struct {
	filename     string // RNG filename for error messages
	errorHandler helium.ErrorHandler
}

type validatorCfg struct {
	filename string
}
