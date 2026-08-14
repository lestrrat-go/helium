package xmlenc1

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUnpadConstantTime pins the two CBC unpadding rules against each other.
// Both read the padding length from the final octet and both require it to
// name between one octet and one whole block. They part company over the
// octets ahead of it: xmlenc-core1 §5.2.1 leaves those arbitrary and
// cbcPaddingXMLEnc does not read them, while PKCS#7 fixes every one of them to
// the length and cbcPaddingPKCS7 checks them.
//
// So cbcPaddingPKCS7 accepts a strict subset. Every row below states what BOTH
// rules answer, which is what makes the subset relation visible: no row may
// have PKCS#7 accept something XML Encryption rejects.
func TestUnpadConstantTime(t *testing.T) {
	const blockSize = 16

	// block builds one 16-octet block from a plaintext prefix, N-1 filler
	// octets, and a final octet naming N.
	block := func(prefix string, filler byte, n int) []byte {
		out := make([]byte, blockSize)
		copy(out, prefix)
		for i := len(prefix); i < blockSize-1; i++ {
			out[i] = filler
		}
		out[blockSize-1] = byte(n)
		return out
	}

	tests := []struct {
		name string
		data []byte
		// wantXMLEnc and wantPKCS7 are the recovered plaintext under each
		// rule; a nil value means the rule rejects the input.
		wantXMLEnc []byte
		wantPKCS7  []byte
	}{
		{
			name:       "pkcs7 padding is valid under both rules",
			data:       block("hello", 11, 11),
			wantXMLEnc: []byte("hello"),
			wantPKCS7:  []byte("hello"),
		},
		{
			name:       "arbitrary filler is valid xmlenc padding and not pkcs7",
			data:       block("hello", 0xAB, 11),
			wantXMLEnc: []byte("hello"),
			wantPKCS7:  nil,
		},
		{
			name:       "zero filler is valid xmlenc padding and not pkcs7",
			data:       block("hello", 0x00, 11),
			wantXMLEnc: []byte("hello"),
			wantPKCS7:  nil,
		},
		{
			name:       "a single padding octet needs no filler at all",
			data:       block("123456789012345", 0x00, 1),
			wantXMLEnc: []byte("123456789012345"),
			wantPKCS7:  []byte("123456789012345"),
		},
		{
			name:       "a whole block of arbitrary padding is valid xmlenc padding",
			data:       block("", 0x7F, blockSize),
			wantXMLEnc: []byte{},
			wantPKCS7:  nil,
		},
		{
			name:       "a whole block of pkcs7 padding is valid under both rules",
			data:       block("", blockSize, blockSize),
			wantXMLEnc: []byte{},
			wantPKCS7:  []byte{},
		},
		{
			name:       "a length of zero is refused by both rules",
			data:       block("hello", 0xAB, 0),
			wantXMLEnc: nil,
			wantPKCS7:  nil,
		},
		{
			name:       "a length past the block size is refused by both rules",
			data:       block("hello", 0xAB, blockSize+1),
			wantXMLEnc: nil,
			wantPKCS7:  nil,
		},
		{
			name:       "a length of 255 is refused by both rules",
			data:       block("hello", 0xAB, 255),
			wantXMLEnc: nil,
			wantPKCS7:  nil,
		},
		{
			name:       "empty input is refused by both rules",
			data:       []byte{},
			wantXMLEnc: nil,
			wantPKCS7:  nil,
		},
		{
			name:       "input that is not a whole number of blocks is refused by both rules",
			data:       bytes.Repeat([]byte{1}, blockSize+1),
			wantXMLEnc: nil,
			wantPKCS7:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotXMLEnc, okXMLEnc := unpadConstantTime(tc.data, cbcPaddingXMLEnc)
			gotPKCS7, okPKCS7 := unpadConstantTime(tc.data, cbcPaddingPKCS7)

			if tc.wantXMLEnc == nil {
				require.False(t, okXMLEnc, "xmlenc rule must reject")
			}
			if tc.wantXMLEnc != nil {
				require.True(t, okXMLEnc, "xmlenc rule must accept")
				require.Equal(t, tc.wantXMLEnc, gotXMLEnc)
			}

			if tc.wantPKCS7 == nil {
				require.False(t, okPKCS7, "pkcs7 rule must reject")
			}
			if tc.wantPKCS7 != nil {
				require.True(t, okPKCS7, "pkcs7 rule must accept")
				require.Equal(t, tc.wantPKCS7, gotPKCS7)
			}

			// The subset relation itself: PKCS#7 never accepts what the
			// XML Encryption rule refuses.
			if okPKCS7 {
				require.True(t, okXMLEnc, "pkcs7 accepted padding the xmlenc rule refused")
			}
		})
	}
}

// TestUnpadConstantTimeDefaultIsXMLEnc pins the zero value of cbcPadding to the
// conforming rule, so a decrypt path that forgets to thread the option through
// stays interoperable rather than silently becoming strict.
func TestUnpadConstantTimeDefaultIsXMLEnc(t *testing.T) {
	var zero cbcPadding
	require.Equal(t, cbcPaddingXMLEnc, zero)

	data := make([]byte, 16)
	copy(data, "hello")
	for i := 5; i < 15; i++ {
		data[i] = 0xAB
	}
	data[15] = 11

	got, ok := unpadConstantTime(data, zero)
	require.True(t, ok)
	require.Equal(t, []byte("hello"), got)
}
