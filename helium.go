package helium

const xmlTextNoEnc = "textnoenc"

// ClarkName returns the Clark notation "{uri}local" for a namespace URI and
// local name pair.
func ClarkName(uri, local string) string {
	return "{" + uri + "}" + local
}
