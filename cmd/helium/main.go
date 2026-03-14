package main

import (
	"context"
	"os"

	"github.com/lestrrat-go/helium/internal/cli/heliumcmd"
)

func main() {
	os.Exit(heliumcmd.Execute(context.Background(), os.Args[1:]))
}
