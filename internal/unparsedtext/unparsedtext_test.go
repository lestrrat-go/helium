package unparsedtext_test

import (
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/lestrrat-go/helium/internal/unparsedtext"
	"github.com/stretchr/testify/require"
)

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", nil},
		{"no newlines", "hello", []string{"hello"}},
		{"lf ending", "a\nb\n", []string{"a", "b"}},
		{"crlf ending", "a\r\nb\r\n", []string{"a", "b"}},
		{"cr only", "a\rb\r", []string{"a", "b"}},
		{"mixed line endings", "a\r\nb\rc\nd\n", []string{"a", "b", "c", "d"}},
		{"no trailing newline", "a\nb", []string{"a", "b"}},
		{"blank lines", "a\n\nb\n", []string{"a", "", "b"}},
		{"only newline", "\n", []string{""}},
		{"only crlf", "\r\n", []string{""}},
		{"only cr", "\r", []string{""}},
		{"multiple blank lines at end", "a\n\n\n", []string{"a", "", ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unparsedtext.SplitLines(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateXMLChars(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid ascii", "hello world", false},
		{"tab is valid", "a\tb", false},
		{"lf is valid", "a\nb", false},
		{"cr is valid", "a\rb", false},
		{"null byte", "a\x00b", true},
		{"control char 0x01", "a\x01b", true},
		{"control char 0x08", "a\x08b", true},
		{"control char 0x0B", "a\x0Bb", true},
		{"control char 0x0C", "a\x0Cb", true},
		{"control char 0x0E", "a\x0Eb", true},
		{"control char 0x1F", "a\x1Fb", true},
		{"valid BMP", "hello \u4e16\u754c", false},
		{"FFFE", "a\uFFFEb", true},
		{"FFFF", "a\uFFFFb", true},
		{"valid emoji (supplementary)", "hello \U0001F600", false},
		{"empty string", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := unparsedtext.ValidateXMLChars(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				var ue *unparsedtext.Error
				require.ErrorAs(t, err, &ue)
				require.Equal(t, unparsedtext.ErrCodeEncoding, ue.Code)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsValidXMLChar(t *testing.T) {
	require.True(t, unparsedtext.IsValidXMLChar('\t'))
	require.True(t, unparsedtext.IsValidXMLChar('\n'))
	require.True(t, unparsedtext.IsValidXMLChar('\r'))
	require.True(t, unparsedtext.IsValidXMLChar(' '))
	require.True(t, unparsedtext.IsValidXMLChar('A'))
	require.True(t, unparsedtext.IsValidXMLChar(0xD7FF))
	require.True(t, unparsedtext.IsValidXMLChar(0xE000))
	require.True(t, unparsedtext.IsValidXMLChar(0xFFFD))
	require.True(t, unparsedtext.IsValidXMLChar(0x10000))
	require.True(t, unparsedtext.IsValidXMLChar(0x10FFFF))

	require.False(t, unparsedtext.IsValidXMLChar(0x00))
	require.False(t, unparsedtext.IsValidXMLChar(0x01))
	require.False(t, unparsedtext.IsValidXMLChar(0x08))
	require.False(t, unparsedtext.IsValidXMLChar(0x0B))
	require.False(t, unparsedtext.IsValidXMLChar(0xD800))
	require.False(t, unparsedtext.IsValidXMLChar(0xFFFE))
	require.False(t, unparsedtext.IsValidXMLChar(0xFFFF))
}

func TestEncodingsCompatible(t *testing.T) {
	tests := []struct {
		specified string
		detected  string
		want      bool
	}{
		{"utf-8", "utf-8", true},
		{"UTF-8", "utf-8", true},
		{"utf_8", "utf-8", true},
		{"utf-16", "utf-16le", true},
		{"utf-16", "utf-16be", true},
		{"UTF-16", "utf-16le", true},
		{"utf-8", "utf-16le", false},
		{"utf-16le", "utf-16be", false},
		{"iso-8859-1", "utf-8", false},
	}
	for _, tt := range tests {
		t.Run(tt.specified+"_vs_"+tt.detected, func(t *testing.T) {
			require.Equal(t, tt.want, unparsedtext.EncodingsCompatible(tt.specified, tt.detected))
		})
	}
}

func TestDecodeText(t *testing.T) {
	t.Run("plain utf-8", func(t *testing.T) {
		text, err := unparsedtext.DecodeText([]byte("hello"), "")
		require.NoError(t, err)
		require.Equal(t, "hello", text)
	})

	t.Run("utf-8 with BOM", func(t *testing.T) {
		data := append([]byte{0xEF, 0xBB, 0xBF}, []byte("hello")...)
		text, err := unparsedtext.DecodeText(data, "")
		require.NoError(t, err)
		require.Equal(t, "hello", text)
	})

	t.Run("utf-8 with BOM and explicit encoding", func(t *testing.T) {
		data := append([]byte{0xEF, 0xBB, 0xBF}, []byte("hello")...)
		text, err := unparsedtext.DecodeText(data, "utf-8")
		require.NoError(t, err)
		require.Equal(t, "hello", text)
	})

	t.Run("utf-16le with BOM", func(t *testing.T) {
		runes := utf16.Encode([]rune("hello"))
		var buf []byte
		buf = append(buf, 0xFF, 0xFE) // LE BOM
		for _, r := range runes {
			buf = binary.LittleEndian.AppendUint16(buf, r)
		}
		text, err := unparsedtext.DecodeText(buf, "")
		require.NoError(t, err)
		require.Equal(t, "hello", text)
	})

	t.Run("utf-16be with BOM", func(t *testing.T) {
		runes := utf16.Encode([]rune("hello"))
		var buf []byte
		buf = append(buf, 0xFE, 0xFF) // BE BOM
		for _, r := range runes {
			buf = binary.BigEndian.AppendUint16(buf, r)
		}
		text, err := unparsedtext.DecodeText(buf, "")
		require.NoError(t, err)
		require.Equal(t, "hello", text)
	})

	t.Run("invalid utf-8", func(t *testing.T) {
		_, err := unparsedtext.DecodeText([]byte{0x80, 0x81}, "")
		require.Error(t, err)
		var ue *unparsedtext.Error
		require.ErrorAs(t, err, &ue)
		require.Equal(t, unparsedtext.ErrCodeEncoding, ue.Code)
	})

	t.Run("unsupported encoding", func(t *testing.T) {
		_, err := unparsedtext.DecodeText([]byte("hello"), "bogus-encoding")
		require.Error(t, err)
		var ue *unparsedtext.Error
		require.ErrorAs(t, err, &ue)
		require.Equal(t, unparsedtext.ErrCodeEncoding, ue.Code)
	})

	t.Run("encoding conflicts with BOM", func(t *testing.T) {
		data := append([]byte{0xFF, 0xFE}, []byte{0x68, 0x00}...) // LE BOM
		_, err := unparsedtext.DecodeText(data, "utf-8")
		require.Error(t, err)
		var ue *unparsedtext.Error
		require.ErrorAs(t, err, &ue)
		require.Equal(t, unparsedtext.ErrCodeEncoding, ue.Code)
		require.Contains(t, ue.Message, "conflicts with BOM")
	})

	t.Run("iso-8859-1", func(t *testing.T) {
		// 0xE9 is 'é' in ISO-8859-1
		text, err := unparsedtext.DecodeText([]byte{0x68, 0xE9, 0x6C, 0x6C, 0x6F}, "iso-8859-1")
		require.NoError(t, err)
		require.Equal(t, "héllo", text)
	})
}

func TestResolveURI(t *testing.T) {
	t.Run("absolute http URI", func(t *testing.T) {
		resolved, err := unparsedtext.ResolveURI(t.Context(), nil, "http://example.com/file.txt")
		require.NoError(t, err)
		require.Equal(t, "http://example.com/file.txt", resolved)
	})

	t.Run("absolute file URI", func(t *testing.T) {
		resolved, err := unparsedtext.ResolveURI(t.Context(), nil, "file:///tmp/file.txt")
		require.NoError(t, err)
		require.Equal(t, "file:///tmp/file.txt", resolved)
	})

	t.Run("fragment rejected", func(t *testing.T) {
		_, err := unparsedtext.ResolveURI(t.Context(), nil, "http://example.com/file.txt#frag")
		require.Error(t, err)
		var ue *unparsedtext.Error
		require.ErrorAs(t, err, &ue)
		require.Equal(t, unparsedtext.ErrCodeRetrieval, ue.Code)
		require.Contains(t, ue.Message, "fragment")
	})

	t.Run("empty href uses base URI", func(t *testing.T) {
		cfg := &unparsedtext.Config{BaseURI: "http://example.com/base.txt"}
		resolved, err := unparsedtext.ResolveURI(t.Context(), cfg, "")
		require.NoError(t, err)
		require.Equal(t, "http://example.com/base.txt", resolved)
	})

	t.Run("empty href no base URI", func(t *testing.T) {
		_, err := unparsedtext.ResolveURI(t.Context(), &unparsedtext.Config{}, "")
		require.Error(t, err)
		var ue *unparsedtext.Error
		require.ErrorAs(t, err, &ue)
		require.Equal(t, unparsedtext.ErrCodeRetrieval, ue.Code)
	})

	t.Run("relative URI resolved against base", func(t *testing.T) {
		cfg := &unparsedtext.Config{BaseURI: "http://example.com/dir/"}
		resolved, err := unparsedtext.ResolveURI(t.Context(), cfg, "file.txt")
		require.NoError(t, err)
		require.Equal(t, "http://example.com/dir/file.txt", resolved)
	})

	t.Run("relative URI without base", func(t *testing.T) {
		_, err := unparsedtext.ResolveURI(t.Context(), nil, "file.txt")
		require.Error(t, err)
		var ue *unparsedtext.Error
		require.ErrorAs(t, err, &ue)
		require.Equal(t, unparsedtext.ErrCodeRetrieval, ue.Code)
	})

	t.Run("unsupported scheme", func(t *testing.T) {
		_, err := unparsedtext.ResolveURI(t.Context(), nil, "ftp://example.com/file.txt")
		require.Error(t, err)
		var ue *unparsedtext.Error
		require.ErrorAs(t, err, &ue)
		require.Contains(t, ue.Message, "unsupported URI scheme")
	})

	t.Run("windows drive path rejected", func(t *testing.T) {
		_, err := unparsedtext.ResolveURI(t.Context(), nil, "C:\\Users\\file.txt")
		require.Error(t, err)
	})
}

func TestReadURI(t *testing.T) {
	t.Run("file path", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		require.NoError(t, os.WriteFile(path, []byte("content"), 0644))

		data, err := unparsedtext.ReadURI(t.Context(), nil, path)
		require.NoError(t, err)
		require.Equal(t, "content", string(data))
	})

	t.Run("file URI", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		require.NoError(t, os.WriteFile(path, []byte("file-uri"), 0644))

		data, err := unparsedtext.ReadURI(t.Context(), nil, "file://"+path)
		require.NoError(t, err)
		require.Equal(t, "file-uri", string(data))
	})

	t.Run("http URI", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("http-content"))
		}))
		defer srv.Close()

		cfg := &unparsedtext.Config{HTTPClient: srv.Client()}
		data, err := unparsedtext.ReadURI(t.Context(), cfg, srv.URL+"/test.txt")
		require.NoError(t, err)
		require.Equal(t, "http-content", string(data))
	})

	t.Run("http 404", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		cfg := &unparsedtext.Config{HTTPClient: srv.Client()}
		_, err := unparsedtext.ReadURI(t.Context(), cfg, srv.URL+"/missing.txt")
		require.Error(t, err)
	})

	t.Run("custom URI resolver", func(t *testing.T) {
		resolver := &unparsedtext.FileURIResolver{BaseDir: t.TempDir()}
		path := filepath.Join(resolver.BaseDir, "resolved.txt")
		require.NoError(t, os.WriteFile(path, []byte("resolved"), 0644))

		cfg := &unparsedtext.Config{URIResolver: resolver}
		data, err := unparsedtext.ReadURI(t.Context(), cfg, "resolved.txt")
		require.NoError(t, err)
		require.Equal(t, "resolved", string(data))
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := unparsedtext.ReadURI(t.Context(), nil, "/nonexistent/path/file.txt")
		require.Error(t, err)
	})
}

func TestLoadText(t *testing.T) {
	t.Run("basic file load", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		require.NoError(t, os.WriteFile(path, []byte("hello world"), 0644))

		cfg := &unparsedtext.Config{BaseURI: "file://" + dir + "/"}
		text, err := unparsedtext.LoadText(t.Context(), cfg, "test.txt", "")
		require.NoError(t, err)
		require.Equal(t, "hello world", text)
	})

	t.Run("rejects non-XML characters", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.txt")
		require.NoError(t, os.WriteFile(path, []byte("hello\x00world"), 0644))

		text, err := unparsedtext.LoadText(t.Context(), nil, "file://"+path, "")
		require.Error(t, err)
		require.Empty(t, text)
		var ue *unparsedtext.Error
		require.ErrorAs(t, err, &ue)
		require.Equal(t, unparsedtext.ErrCodeEncoding, ue.Code)
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := unparsedtext.LoadText(t.Context(), nil, "file:///nonexistent/file.txt", "")
		require.Error(t, err)
		var ue *unparsedtext.Error
		require.ErrorAs(t, err, &ue)
		require.Equal(t, unparsedtext.ErrCodeRetrieval, ue.Code)
	})
}

func TestLoadTextLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lines.txt")
	require.NoError(t, os.WriteFile(path, []byte("line1\r\nline2\nline3"), 0644))

	lines, err := unparsedtext.LoadTextLines(t.Context(), nil, "file://"+path, "")
	require.NoError(t, err)
	require.Equal(t, []string{"line1", "line2", "line3"}, lines)
}

func TestIsAvailable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.txt")
	require.NoError(t, os.WriteFile(path, []byte("ok"), 0644))

	require.True(t, unparsedtext.IsAvailable(t.Context(), nil, "file://"+path, ""))
	require.False(t, unparsedtext.IsAvailable(t.Context(), nil, "file://"+filepath.Join(dir, "nope.txt"), ""))
}

func TestFileURIResolver(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.txt"), []byte("resolved-data"), 0644))

	r := &unparsedtext.FileURIResolver{BaseDir: dir}

	t.Run("relative path", func(t *testing.T) {
		rc, err := r.ResolveURI("data.txt")
		require.NoError(t, err)
		defer func() { _ = rc.Close() }()
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, "resolved-data", string(data))
	})

	t.Run("absolute path", func(t *testing.T) {
		rc, err := r.ResolveURI(filepath.Join(dir, "data.txt"))
		require.NoError(t, err)
		defer func() { _ = rc.Close() }()
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, "resolved-data", string(data))
	})

	t.Run("unsupported scheme", func(t *testing.T) {
		_, err := r.ResolveURI("ftp://example.com/file.txt")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported URI scheme")
	})
}

func TestErrorType(t *testing.T) {
	err := &unparsedtext.Error{Code: "FOUT1170", Message: "test error"}
	require.Equal(t, "FOUT1170: test error", err.Error())
	require.Contains(t, err.Error(), "FOUT1170")

	// Verify it satisfies the error interface and can be matched with errors.As
	var target *unparsedtext.Error
	require.ErrorAs(t, err, &target)
	require.Equal(t, "FOUT1170", target.Code)
}

func TestDecodeTextUTF16WithExplicitEncoding(t *testing.T) {
	// UTF-16LE without BOM but with explicit encoding
	runes := utf16.Encode([]rune("test"))
	var buf []byte
	for _, r := range runes {
		buf = binary.LittleEndian.AppendUint16(buf, r)
	}
	text, err := unparsedtext.DecodeText(buf, "utf-16le")
	require.NoError(t, err)
	require.Equal(t, "test", text)
}

func TestLoadTextHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/hello.txt") {
			_, _ = w.Write([]byte("hello from http"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := &unparsedtext.Config{HTTPClient: srv.Client()}
	text, err := unparsedtext.LoadText(t.Context(), cfg, srv.URL+"/hello.txt", "")
	require.NoError(t, err)
	require.Equal(t, "hello from http", text)
}
