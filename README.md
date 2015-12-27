# helium

[![Build Status](https://travis-ci.org/lestrrat/helium.svg?branch=master)](https://travis-ci.org/lestrrat/helium)
[![GoDoc](https://godoc.org/github.com/lestrrat/helium?status.svg)](https://godoc.org/github.com/lestrrat/helium)

# What on earth?

This is an excercise in rewriting libxml2 in its entirety in Go.

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

* Still very broken, but basic XML parsing (no DTDs yet) is functional, so you can probably write a SAX2 parser that generates the correct DOM structure
* While XML declaration is parsed, encoding is ignored, and assumed to be UTF-8

# Important Notice:

I'm only going to work on this full-throttle until Jan 4, 2016. After that, I need to get back to life for a while again. Help in forms for PRs is better, but if you insiste, Amazon gift cards to lestrrat at gmail is appreciated ;)