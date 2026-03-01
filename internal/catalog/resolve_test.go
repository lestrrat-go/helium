package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveURIUnwrapsURN(t *testing.T) {
	// Catalog with a public entry for "-//OASIS//DTD DocBook XML V4.1.2//EN".
	cat := &Catalog{
		Entries: []Entry{
			{
				Typ:  EntryPublic,
				Name: "-//OASIS//DTD DocBook XML V4.1.2//EN",
				URL:  "file:///usr/share/xml/docbook.dtd",
			},
		},
		Pref: PreferPublic,
	}

	// The URN encoding of "-//OASIS//DTD DocBook XML V4.1.2//EN" per RFC 3151:
	//   -  → -
	//   // → :
	//   (space) → +
	urn := "urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN"

	// ResolveURI should unwrap the URN and resolve via the public entry.
	got := cat.ResolveURI(urn)
	require.Equal(t, "file:///usr/share/xml/docbook.dtd", got)
}

func TestResolveURINonURNUnchanged(t *testing.T) {
	cat := &Catalog{
		Entries: []Entry{
			{
				Typ:  EntryURI,
				Name: "http://example.com/schema.xsd",
				URL:  "file:///local/schema.xsd",
			},
		},
	}

	// Normal URI (not a URN) should resolve via URI entry as before.
	got := cat.ResolveURI("http://example.com/schema.xsd")
	require.Equal(t, "file:///local/schema.xsd", got)
}

func TestResolveURIURNNotFound(t *testing.T) {
	cat := &Catalog{
		Entries: []Entry{
			{
				Typ:  EntryPublic,
				Name: "-//Other//DTD//EN",
				URL:  "file:///other.dtd",
			},
		},
		Pref: PreferPublic,
	}

	// URN that unwraps to a public ID not in the catalog.
	urn := "urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN"
	got := cat.ResolveURI(urn)
	require.Equal(t, "", got)
}
