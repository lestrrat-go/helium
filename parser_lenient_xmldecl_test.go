package helium

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLenientXMLDecl(t *testing.T) {
	const content = `<root />`

	tests := []struct {
		name       string
		input      string
		version    string
		encoding   string
		standalone DocumentStandaloneType
	}{
		{
			name:       "standard order: version encoding standalone",
			input:      `<?xml version="1.0" encoding="utf-8" standalone="yes"?>` + content,
			version:    "1.0",
			encoding:   "utf-8",
			standalone: StandaloneExplicitYes,
		},
		{
			name:       "encoding before version",
			input:      `<?xml encoding="utf-8" version="1.0"?>` + content,
			version:    "1.0",
			encoding:   "utf-8",
			standalone: StandaloneImplicitNo,
		},
		{
			name:       "standalone before version",
			input:      `<?xml standalone="no" version="1.0"?>` + content,
			version:    "1.0",
			encoding:   "",
			standalone: StandaloneExplicitNo,
		},
		{
			name:       "encoding standalone version",
			input:      `<?xml encoding="euc-jp" standalone="yes" version="1.0"?>` + content,
			version:    "1.0",
			encoding:   "euc-jp",
			standalone: StandaloneExplicitYes,
		},
		{
			name:       "standalone version encoding",
			input:      `<?xml standalone="no" version="1.1" encoding="cp932"?>` + content,
			version:    "1.1",
			encoding:   "cp932",
			standalone: StandaloneExplicitNo,
		},
		{
			name:       "version only",
			input:      `<?xml version="1.0"?>` + content,
			version:    "1.0",
			encoding:   "",
			standalone: StandaloneImplicitNo,
		},
		{
			name:       "encoding only (no version)",
			input:      `<?xml encoding="utf-8"?>` + content,
			version:    "",
			encoding:   "utf-8",
			standalone: StandaloneImplicitNo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			p.SetOption(ParseLenientXMLDecl)

			doc, err := p.Parse(t.Context(), []byte(tt.input))
			require.NoError(t, err, "Parse should succeed")
			require.Equal(t, tt.version, doc.Version(), "version")

			if tt.encoding != "" {
				require.Equal(t, tt.encoding, doc.Encoding(), "encoding")
			}
			require.Equal(t, int(tt.standalone), int(doc.Standalone()), "standalone")
		})
	}
}

func TestParseLenientXMLDeclRejectsWithoutFlag(t *testing.T) {
	// Without the lenient flag, non-standard order should fail.
	input := `<?xml encoding="utf-8" version="1.0"?><root />`
	p := NewParser()
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "strict parser should reject encoding before version")
}
