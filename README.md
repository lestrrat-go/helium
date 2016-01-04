# helium

[![Build Status](https://travis-ci.org/lestrrat/helium.svg?branch=master)](https://travis-ci.org/lestrrat/helium)
[![GoDoc](https://godoc.org/github.com/lestrrat/helium?status.svg)](https://godoc.org/github.com/lestrrat/helium)

# What on earth?

This is an exercise in rewriting libxml2 in its entirety in Go. Why? I've run into
performance blockers while using cgo and libxml2, and I couldn't squeeze out that last
drop of performance out of it. Also, there was also [this](https://github.com/golang/go/issues/13400), which I thought was a shame to not be able to handle XML using pure Go.

So I started -- and I still have a long way to go.

# SYNOPSIS

```go
import "github.com/lestrrat/helium"

func main() {
    doc, err := helium.Parse(`.... xml string ....`)
    if err != nil {
        panic("failed to parse XML: " + err.Error())
    }

    // Dump this XML
    doc.XML(os.Stdout)
}
```

# Get it

```
go get github.com/lestrrat/helium
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

# What's with the naming?

I thought it sounded cool. Not set in stone, so we may change it later.
