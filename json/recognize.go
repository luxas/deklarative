package json

import (
	"bytes"
	"unicode"

	"github.com/luxas/deklarative/content"
)

const Recognizer = recognizer(0)

var _ content.PeekRecognizer = Recognizer

type recognizer int

func (recognizer) FromPeekBytes(peek []byte) content.ContentType {
	if isJSONBuffer(peek) {
		return content.ContentTypeJSON
	}
	return ""
}

func (recognizer) SupportedContentTypes() content.ContentTypes {
	return []content.ContentType{content.ContentTypeJSON}
}

// isJSONBuffer scans the provided buffer, looking
// for an open brace indicating this is JSON.
func isJSONBuffer(buf []byte) bool {
	return hasJSONPrefix(buf)
}

// hasJSONPrefix returns true if the provided buffer appears to start with
// a JSON open brace.
func hasJSONPrefix(buf []byte) bool {
	return hasPrefix(buf, []byte("{"))
}

// Return true if the first non-whitespace bytes in buf is
// prefix.
func hasPrefix(buf []byte, prefix []byte) bool {
	trim := bytes.TrimLeftFunc(buf, unicode.IsSpace)
	return bytes.HasPrefix(trim, prefix)
}
