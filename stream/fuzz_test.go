package stream_test

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/stream"
)

func FuzzWriter(f *testing.F) {
	f.Add([]byte{0, 1, 4, 'r', 'o', 'o', 't', 7, 5, 'h', 'e', 'l', 'l', 'o', 2, 3})
	f.Add([]byte{0, 8, 4, 'r', 'o', 'o', 't', 4, 2, 'i', 'd', 1, 'x', 7, 4, 't', 'e', 'x', 't', 2, 3})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}

		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		if len(data) > 0 && data[0]&1 == 1 {
			w = w.Indent("  ")
		}
		if len(data) > 0 && data[0]&2 == 2 {
			w = w.QuoteChar('\'')
		}

		for i := 1; i < len(data); {
			op := data[i] % 14
			i++

			switch op {
			case 0:
				_ = w.StartDocument("", "", "")
			case 1:
				_ = w.StartElement(fuzzChunk(data, &i, 24))
			case 2:
				_ = w.EndElement()
			case 3:
				_ = w.EndDocument()
			case 4:
				name := fuzzChunk(data, &i, 16)
				value := fuzzChunk(data, &i, 24)
				_ = w.WriteAttribute(name, value)
			case 5:
				prefix := fuzzChunk(data, &i, 8)
				local := fuzzChunk(data, &i, 16)
				uri := fuzzChunk(data, &i, 24)
				_ = w.StartElementNS(prefix, local, uri)
			case 6:
				prefix := fuzzChunk(data, &i, 8)
				local := fuzzChunk(data, &i, 16)
				uri := fuzzChunk(data, &i, 24)
				value := fuzzChunk(data, &i, 24)
				_ = w.WriteAttributeNS(prefix, local, uri, value)
			case 7:
				_ = w.WriteString(fuzzChunk(data, &i, 48))
			case 8:
				_ = w.StartComment()
			case 9:
				_ = w.EndComment()
			case 10:
				_ = w.StartPI(fuzzChunk(data, &i, 16))
			case 11:
				_ = w.EndPI()
			case 12:
				_ = w.StartCDATA()
			case 13:
				_ = w.EndCDATA()
			}
		}

		_ = w.EndDocument()
		if w.Error() == nil && buf.Len() > 0 {
			_, _ = helium.NewParser().Parse(t.Context(), buf.Bytes())
		}
	})
}

func fuzzChunk(data []byte, index *int, max int) string {
	if *index >= len(data) {
		return ""
	}

	n := int(data[*index] % byte(max+1))
	*index++
	if *index+n > len(data) {
		n = len(data) - *index
	}
	s := string(data[*index : *index+n])
	*index += n
	return s
}
