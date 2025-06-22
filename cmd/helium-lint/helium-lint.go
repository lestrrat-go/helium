package main

import (
	"fmt"
	"io"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/cliutil"
)

type cmdopts struct {
	Format   bool `long:"format"`
	NoBlanks bool `long:"noblanks"`
	Version  bool `long:"version"`
}

func main() {
	os.Exit(_main())
}

func showVersion() {
	fmt.Printf("helium-lint: using helium version %s\n", helium.Version)
}

func showUsage() {
	fmt.Printf(`Usage : helium-lint [options] XMLfiles ...
	Parse the XML files and output the result of the parsing
	--version : display the version of the XML library used
`)
}

func _main() int {
	opts := cmdopts{}
	args, err := flags.ParseArgs(&opts, os.Args[1:])
	if err != nil {
		showUsage()
		return 1
	}

	if opts.Version {
		showVersion()
		return 0
	}

	inputCh := make(chan io.Reader)
	errCh := make(chan error)
	switch {
	case len(args) > 0: // filename present
		go func() {
			defer close(inputCh)
			for _, f := range args {
				fh, err := os.Open(f)
				if err != nil {
					errCh <- err
					return
				}
				inputCh <- fh
			}
		}()
	case !cliutil.IsTty(os.Stdin.Fd()):
		go func() {
			defer close(inputCh)
			inputCh <- os.Stdin
		}()
	default:
		showUsage()
		return 1
	}

	p := helium.NewParser()
	for in := range inputCh {
		buf, err := io.ReadAll(in)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return 1
		}
		doc, err := p.Parse(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return 1
		}

		d := helium.Dumper{}
		d.DumpDoc(os.Stdout, doc)
	}

	select {
	case err := <-errCh:
		fmt.Fprintf(os.Stderr, "%s", err)
		return 1
	default:
	}

	return 0
}
