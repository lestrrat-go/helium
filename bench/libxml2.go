//go:build cgo && libxml2bench

package bench

/*
#cgo pkg-config: libxml-2.0
#include <libxml/parser.h>
#include <libxml/tree.h>

// parseAndFree parses XML from memory, frees the document, and returns 0
// on success or -1 on failure. Keeping parse+free in a single C function
// avoids paying ~100ns cgo-crossing overhead twice per iteration.
static int parseAndFree(const char *buf, int len) {
	xmlDocPtr doc = xmlParseMemory(buf, len);
	if (doc == NULL) {
		return -1;
	}
	xmlFreeDoc(doc);
	return 0;
}
*/
import "C"

import "unsafe"

// Libxml2Init initializes the libxml2 parser. Call once before benchmarking.
func Libxml2Init() {
	C.xmlInitParser()
}

// Libxml2Cleanup frees libxml2 global state.
func Libxml2Cleanup() {
	C.xmlCleanupParser()
}

// Libxml2ParseAndFree parses XML from a byte slice using libxml2 and
// immediately frees the resulting document. Returns an error if parsing fails.
func Libxml2ParseAndFree(data []byte) bool {
	cBuf := (*C.char)(unsafe.Pointer(&data[0]))
	cLen := C.int(len(data))
	return C.parseAndFree(cBuf, cLen) == 0
}
