package xmldsig1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReferenceCoversElement(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  parsedReference
		want bool
	}{
		{
			name: "canonicalized element",
			ref: parsedReference{
				uri:        "#payload",
				transforms: []parsedTransform{{algorithm: ExcC14N10}},
			},
			want: true,
		},
		{
			name: "xpath filtered element",
			ref: parsedReference{
				uri:        "#payload",
				transforms: []parsedTransform{{algorithm: TransformXPath}},
			},
		},
		{
			name: "enveloped element",
			ref: parsedReference{
				uri:        "#payload",
				transforms: []parsedTransform{{algorithm: TransformEnvelopedSignature}},
			},
		},
		{
			name: "manifest",
			ref: parsedReference{
				uri:        "#payload",
				refType:    TypeManifest,
				transforms: []parsedTransform{{algorithm: ExcC14N10}},
			},
		},
		{
			name: "external reference",
			ref:  parsedReference{uri: "https://example.test/payload"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, referenceCoversElement(tc.ref))
		})
	}
}
