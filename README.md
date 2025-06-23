# helium

[![Build Status](https://travis-ci.org/lestrrat-go/helium.svg?branch=main)](https://travis-ci.org/lestrrat-go/helium)
[![GoDoc](https://godoc.org/github.com/lestrrat-go/helium?status.svg)](https://godoc.org/github.com/lestrrat-go/helium)

# What on earth?

This is an exercise in rewriting libxml2 in its entirety in Go. Why? I've run into
performance blockers while using cgo and libxml2, and I couldn't squeeze out that last
drop of performance out of it. Also, there was also [this](https://github.com/golang/go/issues/13400), which I thought was a shame to not be able to handle XML using pure Go.

So I started -- and I still have a long way to go.

# SYNOPSIS

Parsing XML into a DOM model, then dumping it:

```go
import "github.com/lestrrat-go/helium"

func main() {
    doc, err := helium.Parse(`.... xml string ....`)
    if err != nil {
        panic("failed to parse XML: " + err.Error())
    }

    // Dump this XML
    doc.XML(os.Stdout)
}
```

Using command line `helium-lint` (very under developed right now):

```
helium-lint xmlfile ...
```

```
cat xmlfile | helium-lint
```

# Get it

```
go get github.com/lestrrat-go/helium
```

# Test it

```
go test
```

In order to get helpful debug messages:

```
go test -tags debug
```

# Current status

* Good news: parse/dump basic XML with some DTDs are now working.
* Bad news: I have run out of tuits. I intend to work on this from time to time, but I *REALLY* need people's help. See "Contributing" below.
* While XML declaration is parsed, encoding is ignored, and assumed to be UTF-8

# Contributing

I won't have much time to discuss: let the code talk: Give me PRs that are self-explanatory! :)

The goal for the moment is to "port" libxml2. That means that where possible, we should just steal their code, even if things aren't too go-ish. We'll polish after we have a compatible implementation. So don't debate for optimal go-ness unless it's a really low hanging fruit.

Help in forms for PRs is better, but if you insiste, Amazon gift cards to lestrrat at gmail is appreciated ;)

To get started, see notes on test structure below. Grab some files from libxml2, and see
if things work. If it doesn't work, fix it! :)

# Test structure

As of this writing, `dump_test.go` and `sax_test.go` both look for the presence of XML
files under test directory, and parse+dumps appropriate output after seeing that there's
a corresponding `*.dump` or `*.sax2` files.

For SAX tests, do note that letter casing and such are different from that of libxml2.

# What's with the naming?

I thought it sounded cool. Not set in stone, so we may change it later.
