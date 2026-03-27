package sink_test

import (
	"context"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium/sink"
)

func FuzzSink(f *testing.F) {
	f.Add([]byte{0, 5, 'h', 'e', 'l', 'l', 'o', 0, 5, 'w', 'o', 'r', 'l', 'd', 1, 2})
	f.Add([]byte{0, 1, 'a', 0, 1, 'b', 0, 1, 'c', 1})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		var mu sync.Mutex
		var got []string

		bufSize := 1
		if len(data) > 0 {
			bufSize += int(data[0] % 32)
		}

		s := sink.New[string](ctx, sink.HandlerFunc[string](func(_ context.Context, v string) {
			mu.Lock()
			got = append(got, v)
			mu.Unlock()
		}), sink.WithBufferSize(bufSize))

		for i := 1; i < len(data); {
			op := data[i] % 3
			i++

			switch op {
			case 0:
				if i >= len(data) {
					continue
				}
				n := int(data[i] % 32)
				i++
				if i+n > len(data) {
					n = len(data) - i
				}
				s.Handle(ctx, string(data[i:i+n]))
				i += n
			case 1:
				_ = s.Close()
			case 2:
				cancel()
			}
		}

		_ = s.Close()
		s.Handle(ctx, "after-close")
	})
}
