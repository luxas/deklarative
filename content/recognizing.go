package content

import (
	"path/filepath"

	"github.com/luxas/deklarative/content/metadata"
)

type MetadataRecognizer interface {
	FromMetadata(h metadata.Header) ContentType

	//metadata.HeaderOption

	// SupportedContentTypes() tells about what ContentTypes are supported by this recognizer
	//ContentTypeSupporter
}

type PeekRecognizer interface {
	FromPeekBytes(peek []byte) ContentType

	// SupportedContentTypes() tells about what ContentTypes are supported by this recognizer
	//ContentTypeSupporter
}

var _ MetadataRecognizer = ExtToContentTypeMap{}

// ExtToContentTypeMap is a metadata.HeaderOption implementation that
// based on the X-Content-Location header sets the content type. The
// map maps the extension to the content type.
type ExtToContentTypeMap map[string]ContentType

func (m ExtToContentTypeMap) FromMetadata(h metadata.Header) ContentType {
	loc, ok := metadata.GetString(h, metadata.XContentLocationKey)
	if !ok {
		return ""
	}
	ext := filepath.Ext(loc)
	ct, ok := m[ext]
	if !ok {
		return ""
	}

	return ct
}

func HeaderOptionFromMetaRecognizer(rec MetadataRecognizer) metadata.HeaderOption {
	return &metadataRecognizerOption{rec}
}

type metadataRecognizerOption struct{ MetadataRecognizer }

func (o *metadataRecognizerOption) ApplyToHeader(h metadata.Header) {
	ct, _ := metadata.GetString(h, metadata.ContentTypeKey)
	if len(ct) != 0 {
		return // no need to recognize
	}

	ct = string(o.FromMetadata(h))
	if len(ct) == 0 {
		return // no recognition result
	}

	h.Set(metadata.ContentTypeKey, ct)
}

/*func (m ExtToContentTypeMap) SupportedContentTypes() ContentTypes {
	cts := ContentTypes{}
	for _, ct := range m {
		if !cts.Has(ct) {
			cts = append(cts, ct)
		}
	}
	return cts
}*/
