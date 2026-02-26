package encoding

import "testing"

func TestISO88591(t *testing.T) {
	e := Load("iso-8859-1")
	dec := e.NewDecoder()
	enc := e.NewEncoder()
	for i := 0; i <= 255; i++ {
		v := string([]byte{byte(i)})
		s, err := dec.String(v)
		if err != nil {
			t.Logf("Failed to decode '%#x': %s", v, err)
		} else {
			t.Logf("%#x -> '%s'", v, s)
		}

		if i >= 0x80 && i <= 0x9f {
			continue
		}
		v1, err := enc.String(s)
		if err != nil {
			t.Logf("Failed to encode '%s': %s", s, err)
		} else {
			t.Logf("'%s' -> '%#x'", s, v1)
		}
	}
}
