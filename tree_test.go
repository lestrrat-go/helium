package helium_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/stretchr/testify/require"
)

func TestBaseURI(t *testing.T) {
	t.Parallel()

	// BuildURI across local-path, http, and absolute cases.
	t.Run("BuildURI", func(t *testing.T) {
		tests := []struct {
			name     string
			systemID string
			base     string
			want     string
		}{
			{"absolute system id is returned verbatim", "http://x/a.dtd", "http://y/", "http://x/a.dtd"},
			{"relative against http base", "a.dtd", "http://host/dir/doc.xml", "http://host/dir/a.dtd"},
			{"relative against file path", "a.dtd", "/dir/doc.xml", "/dir/a.dtd"},
			{"absolute local path", "/abs/a.dtd", "/dir/doc.xml", "/abs/a.dtd"},
			// Windows shapes are plain strings, so the Windows behavior below is
			// exercised on any GOOS. A native Windows base must NOT route the drive
			// letter through url.Parse (which would emit "c:///a.dtd"); it resolves
			// with local-path (forward-slash) semantics.
			{"relative against windows backslash base", "child.xml", `C:\dir\main.xml`, "C:/dir/child.xml"},
			{"relative against windows forward-slash base", "a.dtd", "D:/dir/doc.xml", "D:/dir/a.dtd"},
			{"windows-absolute system id returned verbatim", `C:\abs\a.dtd`, `D:\dir\doc.xml`, `C:\abs\a.dtd`},
			{"interior dot-dot against windows base", "../sib/child.xml", `C:\a\b\main.xml`, "C:/a/sib/child.xml"},
			{"unc base resolves relative ref", "child.xml", `\\host\share\main.xml`, "//host/share/child.xml"},
			// An absolute-URI systemID stands on its own even when the base is a
			// native Windows path. Without the scheme check this collapsed "http://"
			// to "http:/" and joined it onto the drive-letter base (Windows-only
			// regression that broke the W3C resolve-uri/base-uri cluster).
			{"absolute http system id against windows drive base", "http://example.com/a/b", `D:\dir\doc.xsl`, "http://example.com/a/b"},
			{"absolute http system id against windows slash base", "http://example.com/a/b", "D:/dir/doc.xsl", "http://example.com/a/b"},
			{"absolute file system id against windows base", "file:///x/y", `C:\dir\doc.xsl`, "file:///x/y"},
			// A RELATIVE Windows base (backslashes, no drive — what filepath.Join
			// yields on Windows for a relative test path) must keep its directory so a
			// sibling entity resolves inside it. Without backslash-aware handling this
			// dropped to a bare "world.txt" and the external entity could not be found.
			{"sibling against relative windows base", "world.txt", `..\d\e\example.xml`, "../d/e/world.txt"},
			// A file: base with a Windows drive letter must yield a proper file: URI
			// (not the drive-rooted "/D:/..." path url.Parse exposes), so file-URI-aware
			// loaders convert it back to a native path. The POSIX file: base below
			// keeps returning a plain path, proving POSIX is unaffected.
			{"sibling against windows drive file uri", "nested.dtd", "file:///D:/tmp/t/inc.xml", "file:///D:/tmp/t/nested.dtd"},
			{"sibling against posix file uri", "nested.dtd", "file:///tmp/t/inc.xml", "/tmp/t/nested.dtd"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				require.Equal(t, tt.want, helium.BuildURI(tt.systemID, tt.base))
			})
		}
	})

	t.Run("ResolveURI", func(t *testing.T) {
		// ResolveURI uses the conventional (base, ref) order and must agree with
		// BuildURI(ref, base).
		tests := []struct {
			name string
			base string
			ref  string
			want string
		}{
			{"relative against http base", "http://host/dir/doc.xml", "sib.dtd", "http://host/dir/sib.dtd"},
			{"relative against file path", "/dir/doc.xml", "sib.dtd", "/dir/sib.dtd"},
			{"nested relative against relative base", "docs/a.xml", "sub/c.xml", "docs/sub/c.xml"},
			{"absolute ref is returned verbatim", "http://y/", "http://x/a.dtd", "http://x/a.dtd"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := helium.ResolveURI(tt.base, tt.ref)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
				// The conventional-order wrapper must match the libxml2-parity
				// primitive with the arguments swapped back.
				require.Equal(t, helium.BuildURI(tt.ref, tt.base), got)
			})
		}

		t.Run("unresolvable reference returns an error", func(t *testing.T) {
			t.Parallel()
			// An invalid percent-escape makes the underlying url.Parse fail, so
			// BuildURI yields an empty string and ResolveURI reports an error.
			_, err := helium.ResolveURI("http://host/dir/x", "%zz")
			require.Error(t, err)
		})
	})

	t.Run("NodeGetBase", func(t *testing.T) {
		t.Run("no base", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			e, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(e))

			base := helium.NodeGetBase(doc, e)
			require.Equal(t, "", base)
		})

		t.Run("direct xml:base", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			e, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(e))

			xmlNS := helium.NewNamespace("xml", lexicon.NamespaceXML)
			err = e.SetAttributeNS("base", "http://example.com/", xmlNS)
			require.NoError(t, err)

			base := helium.NodeGetBase(doc, e)
			require.Equal(t, "http://example.com/", base)
		})

		t.Run("inherited xml:base", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			parent, err := doc.CreateElement("parent")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(parent))

			xmlNS := helium.NewNamespace("xml", lexicon.NamespaceXML)
			err = parent.SetAttributeNS("base", "http://example.com/dir/", xmlNS)
			require.NoError(t, err)

			child, err := doc.CreateElement("child")
			require.NoError(t, err)
			require.NoError(t, parent.AddChild(child))

			base := helium.NodeGetBase(doc, child)
			require.Equal(t, "http://example.com/dir/", base)
		})

		t.Run("relative resolution", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			parent, err := doc.CreateElement("parent")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(parent))

			xmlNS := helium.NewNamespace("xml", lexicon.NamespaceXML)
			err = parent.SetAttributeNS("base", "http://example.com/a/b/", xmlNS)
			require.NoError(t, err)

			child, err := doc.CreateElement("child")
			require.NoError(t, err)
			require.NoError(t, parent.AddChild(child))
			err = child.SetAttributeNS("base", "c/d/", xmlNS)
			require.NoError(t, err)

			base := helium.NodeGetBase(doc, child)
			require.Equal(t, "http://example.com/a/b/c/d/", base)
		})
	})

	// NodeGetBase with xml:base attributes and the
	// SetNodeBaseURI override.
	t.Run("NodeGetBase with xml:base", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		doc.SetURL("http://example.com/dir/doc.xml")

		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		xmlNS := helium.NewNamespace("xml", "http://www.w3.org/XML/1998/namespace")
		require.NoError(t, child.SetAttributeNS("base", "sub/", xmlNS))
		require.NoError(t, root.AddChild(child))

		// The child's effective base resolves its xml:base against the doc URL.
		base := helium.NodeGetBase(doc, child)
		require.Contains(t, base, "sub")

		// A nil node yields an empty base.
		require.Equal(t, "", helium.NodeGetBase(doc, nil))

		// SetNodeBaseURI installs an explicit entity base URI that takes precedence.
		helium.SetNodeBaseURI(child, "http://other.example/")
		base = helium.NodeGetBase(doc, child)
		require.Contains(t, base, "other.example")
	})
}

func TestTreeMutation(t *testing.T) {
	t.Parallel()

	t.Run("unlink", func(t *testing.T) {
		t.Run("unlink middle child", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			parent, err := doc.CreateElement("parent")
			require.NoError(t, err)
			a, err := doc.CreateElement("a")
			require.NoError(t, err)
			b, err := doc.CreateElement("b")
			require.NoError(t, err)
			c, err := doc.CreateElement("c")
			require.NoError(t, err)
			require.NoError(t, parent.AddChild(a))
			require.NoError(t, parent.AddChild(b))
			require.NoError(t, parent.AddChild(c))

			helium.UnlinkNode(b)

			require.Nil(t, b.Parent())
			require.Nil(t, b.PrevSibling())
			require.Nil(t, b.NextSibling())
			require.Equal(t, c, a.NextSibling())
			require.Equal(t, a, c.PrevSibling())
		})

		t.Run("unlink first child", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			parent, err := doc.CreateElement("parent")
			require.NoError(t, err)
			a, err := doc.CreateElement("a")
			require.NoError(t, err)
			b, err := doc.CreateElement("b")
			require.NoError(t, err)
			require.NoError(t, parent.AddChild(a))
			require.NoError(t, parent.AddChild(b))

			helium.UnlinkNode(a)

			require.Equal(t, helium.Node(b), parent.FirstChild())
			require.Nil(t, b.PrevSibling())
			require.Nil(t, a.Parent())
		})

		t.Run("unlink last child", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			parent, err := doc.CreateElement("parent")
			require.NoError(t, err)
			a, err := doc.CreateElement("a")
			require.NoError(t, err)
			b, err := doc.CreateElement("b")
			require.NoError(t, err)
			require.NoError(t, parent.AddChild(a))
			require.NoError(t, parent.AddChild(b))

			helium.UnlinkNode(b)

			require.Equal(t, helium.Node(a), parent.LastChild())
			require.Nil(t, a.NextSibling())
			require.Nil(t, b.Parent())
		})

		t.Run("unlink only child", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			parent, err := doc.CreateElement("parent")
			require.NoError(t, err)
			a, err := doc.CreateElement("a")
			require.NoError(t, err)
			require.NoError(t, parent.AddChild(a))

			helium.UnlinkNode(a)

			require.Nil(t, parent.FirstChild())
			require.Nil(t, parent.LastChild())
			require.Nil(t, a.Parent())
		})

		t.Run("unlink nil is no-op", func(t *testing.T) {
			helium.UnlinkNode(nil) // should not panic
		})
	})

	t.Run("replace detaches the old node", func(t *testing.T) {
		// Build <root><a/><secret/><b/></root>
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))
		a, err := doc.CreateElement("a")
		require.NoError(t, err)
		secret, err := doc.CreateElement("secret")
		require.NoError(t, err)
		b, err := doc.CreateElement("b")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(a))
		require.NoError(t, root.AddChild(secret))
		require.NoError(t, root.AddChild(b))

		repl, err := doc.CreateElement("EncryptedData")
		require.NoError(t, err)
		require.NoError(t, secret.Replace(repl))

		// After replacement the old node must be fully detached.
		require.Nil(t, secret.Parent(), "replaced node parent must be cleared")
		require.Nil(t, secret.PrevSibling(), "replaced node prev must be cleared")
		require.Nil(t, secret.NextSibling(), "replaced node next must be cleared")

		// Tree must read a / EncryptedData / b.
		require.Equal(t, a, root.FirstChild())
		require.Equal(t, repl, a.NextSibling())
		require.Equal(t, b, repl.NextSibling())
		require.Equal(t, b, root.LastChild())

		// A stale UnlinkNode on the old handle must NOT corrupt the tree.
		helium.UnlinkNode(secret)
		require.Equal(t, a, root.FirstChild())
		require.Equal(t, repl, a.NextSibling())
		require.Equal(t, b, repl.NextSibling())
		require.Equal(t, b, root.LastChild())
		require.Equal(t, repl, b.PrevSibling())
	})

	t.Run("replace self", func(t *testing.T) {
		t.Run("exact self-replacement is a no-op", func(t *testing.T) {
			// Build <root><a/><b/></root>
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))
			a, err := doc.CreateElement("a")
			require.NoError(t, err)
			b, err := doc.CreateElement("b")
			require.NoError(t, err)
			require.NoError(t, root.AddChild(a))
			require.NoError(t, root.AddChild(b))

			// Replacing a node with itself must leave the tree intact.
			require.NoError(t, a.Replace(a))

			require.Equal(t, root, a.Parent(), "a.Parent() must remain root")
			require.Equal(t, b, a.NextSibling(), "a.NextSibling() must remain b")
			require.Equal(t, a, root.FirstChild(), "root.FirstChild() must remain a")
			require.Equal(t, b, root.LastChild())
			require.Equal(t, a, b.PrevSibling())
		})

		t.Run("replacement list including the replaced node keeps it live", func(t *testing.T) {
			// Build <root><a/><b/></root>
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))
			a, err := doc.CreateElement("a")
			require.NoError(t, err)
			b, err := doc.CreateElement("b")
			require.NoError(t, err)
			require.NoError(t, root.AddChild(a))
			require.NoError(t, root.AddChild(b))

			// Replace a with [a, c]: a stays live, c is inserted after it.
			c, err := doc.CreateElement("c")
			require.NoError(t, err)
			require.NoError(t, a.Replace(a, c))

			require.Equal(t, root, a.Parent())
			require.Equal(t, a, root.FirstChild())
			require.Equal(t, c, a.NextSibling())
			require.Equal(t, b, c.NextSibling())
			require.Equal(t, b, root.LastChild())
		})
	})

	t.Run("replace with an existing sibling", func(t *testing.T) {
		t.Run("replace node with its own next sibling", func(t *testing.T) {
			// Build <root><a/><b/></root>
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))
			a, err := doc.CreateElement("a")
			require.NoError(t, err)
			b, err := doc.CreateElement("b")
			require.NoError(t, err)
			require.NoError(t, root.AddChild(a))
			require.NoError(t, root.AddChild(b))

			// Replacing a with its own next sibling b must yield a
			// well-formed chain with just b as root's only child.
			require.NoError(t, a.Replace(b))

			require.Equal(t, root, b.Parent())
			require.Equal(t, b, root.FirstChild())
			require.Equal(t, b, root.LastChild())
			require.Nil(t, b.NextSibling(), "b.NextSibling() must be nil (no self-loop)")
			require.Nil(t, b.PrevSibling(), "b.PrevSibling() must be nil (no self-loop)")
		})

		t.Run("replace node with its own previous sibling", func(t *testing.T) {
			// Build <root><a/><b/></root>
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))
			a, err := doc.CreateElement("a")
			require.NoError(t, err)
			b, err := doc.CreateElement("b")
			require.NoError(t, err)
			require.NoError(t, root.AddChild(a))
			require.NoError(t, root.AddChild(b))

			// Replacing b with its own previous sibling a must yield a
			// well-formed chain with just a as root's only child.
			require.NoError(t, b.Replace(a))

			require.Equal(t, root, a.Parent())
			require.Equal(t, a, root.FirstChild())
			require.Equal(t, a, root.LastChild())
			require.Nil(t, a.NextSibling(), "a.NextSibling() must be nil (no self-loop)")
			require.Nil(t, a.PrevSibling(), "a.PrevSibling() must be nil (no self-loop)")
		})

		t.Run("replace middle node with its next sibling", func(t *testing.T) {
			// Build <root><a/><b/><c/></root>
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))
			a, err := doc.CreateElement("a")
			require.NoError(t, err)
			b, err := doc.CreateElement("b")
			require.NoError(t, err)
			c, err := doc.CreateElement("c")
			require.NoError(t, err)
			require.NoError(t, root.AddChild(a))
			require.NoError(t, root.AddChild(b))
			require.NoError(t, root.AddChild(c))

			// Replace b with c: chain becomes a / c with no self-loop.
			require.NoError(t, b.Replace(c))

			require.Equal(t, a, root.FirstChild())
			require.Equal(t, c, a.NextSibling())
			require.Equal(t, a, c.PrevSibling())
			require.Nil(t, c.NextSibling(), "c.NextSibling() must be nil")
			require.Equal(t, c, root.LastChild())
		})
	})

	// the public UnsafeAppendChild helper across the
	// empty-parent and non-empty-parent fast paths.
	t.Run("UnsafeAppendChild", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		parent, err := doc.CreateElement("parent")
		require.NoError(t, err)

		first, err := doc.CreateElement("first")
		require.NoError(t, err)
		require.NoError(t, helium.UnsafeAppendChild(parent, first), "fast-link first child")
		require.Equal(t, helium.Node(first), parent.FirstChild())
		require.Equal(t, helium.Node(first), parent.LastChild())
		require.Equal(t, helium.Node(parent), first.Parent())

		second, err := doc.CreateElement("second")
		require.NoError(t, err)
		require.NoError(t, helium.UnsafeAppendChild(parent, second), "fast-link second child")
		require.Equal(t, helium.Node(second), parent.LastChild())
		require.Equal(t, helium.Node(second), first.NextSibling())
		require.Equal(t, helium.Node(first), second.PrevSibling())
	})

	// documents that raw single-pointer linkage
	// is only reachable through the explicitly-unsafe UnsafeSet* functions, while
	// the ordinary guarded path (AddChild) rejects the same cycle.
	t.Run("raw linkage behind the unsafe surface", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		a, err := doc.CreateElement("a")
		require.NoError(t, err)
		b, err := doc.CreateElement("b")
		require.NoError(t, err)
		require.NoError(t, a.AddChild(b))

		// The guarded path refuses to form a parent cycle.
		require.Error(t, b.AddChild(a), "AddChild must reject a cycle")

		// The unsafe primitive still builds one when a caller explicitly opts in.
		helium.UnsafeSetParent(a, b)
		require.Equal(t, helium.Node(b), a.Parent())
	})
}

func TestWalk(t *testing.T) {
	t.Run("sees sibling replacement during traversal", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		a, err := doc.CreateElement("a")
		require.NoError(t, err)
		c, err := doc.CreateElement("c")
		require.NoError(t, err)

		require.NoError(t, doc.AddChild(root))
		require.NoError(t, root.AddChild(a))
		require.NoError(t, root.AddChild(c))

		var visited []string
		err = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
			if n.Type() != helium.ElementNode {
				return nil
			}

			visited = append(visited, n.Name())
			if n == a {
				b, err := doc.CreateElement("b")
				require.NoError(t, err)
				require.NoError(t, c.Replace(b))
			}
			return nil
		}))
		require.NoError(t, err)
		require.Equal(t, []string{"root", "a", "b"}, visited)
	})

	t.Run("skips sibling removed during traversal", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		a, err := doc.CreateElement("a")
		require.NoError(t, err)
		c, err := doc.CreateElement("c")
		require.NoError(t, err)

		require.NoError(t, doc.AddChild(root))
		require.NoError(t, root.AddChild(a))
		require.NoError(t, root.AddChild(c))

		var visited []string
		err = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
			if n.Type() != helium.ElementNode {
				return nil
			}

			visited = append(visited, n.Name())
			if n == a {
				helium.UnlinkNode(c)
			}
			return nil
		}))
		require.NoError(t, err)
		require.Equal(t, []string{"root", "a"}, visited)
	})
}

func TestNodeAccessors(t *testing.T) {
	t.Parallel()

	t.Run("text", func(t *testing.T) {
		t.Run("AppendText", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			n := doc.CreateText([]byte("Hello "))
			require.NoError(t, n.AppendText([]byte("World!")), "AppendText succeeds")
			require.Equal(t, []byte("Hello World!"), n.Content(), "Content matches")
		})

		t.Run("AddChild merges text", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			n1 := doc.CreateText([]byte("Hello "))
			n2 := doc.CreateText([]byte("World!"))

			require.NoError(t, n1.AddChild(n2), "AddChild succeeds")
			require.Equal(t, []byte("Hello World!"), n1.Content(), "Content matches")
		})

		t.Run("AddChild rejects non-text", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			n1 := doc.CreateText([]byte("Hello "))
			n2 := &helium.ProcessingInstruction{}

			require.ErrorIs(t, n1.AddChild(n2), helium.ErrInvalidOperation, "AddChild fails")
			require.Equal(t, []byte("Hello "), n1.Content(), "Content matches")
		})
	})

	// docnode.Line via a parsed node that carries line info.
	t.Run("line", func(t *testing.T) {
		const src = "<root>\n  <child/>\n</root>"
		doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		root := doc.DocumentElement()
		require.NotNil(t, root)
		// Line() returns the recorded line number; it must be a non-negative int and
		// not panic. We assert it is callable and consistent.
		require.GreaterOrEqual(t, root.Line(), 0)
		for c := root.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Type() == helium.ElementNode {
				require.GreaterOrEqual(t, c.Line(), 0)
			}
		}
	})

	// the XIncludeMarker node type and its methods.
	t.Run("XInclude marker", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		m := helium.NewXIncludeMarker(doc, helium.XIncludeStartNode, "include")
		require.Equal(t, helium.XIncludeStartNode, m.Type())
		require.Equal(t, "include", m.Name())

		child := doc.CreateText([]byte("hello"))
		require.NoError(t, m.AddChild(child))
		require.NoError(t, m.AppendText([]byte(" world")))
		m.SetTreeDoc(doc)
	})
}
