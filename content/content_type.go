package content

import (
	"fmt"

	"github.com/luxas/deklarative/content/metadata"
)

var _ fmt.Stringer = ContentType("")

// ContentType specifies the content type of some stream.
// Ideally, a standard MIME notation like "application/json" shall be used.
type ContentType string //nolint:revive

const (
	// TODO: Maybe consider moving these to their respective package?
	ContentTypeYAML ContentType = "application/yaml"
	ContentTypeJSON ContentType = "application/json"
)

func (ct ContentType) ContentType() ContentType { return ct }
func (ct ContentType) String() string           { return string(ct) }

type ContentTypes []ContentType //nolint:revive

func (cts ContentTypes) Has(want ContentType) bool {
	for _, ct := range cts {
		if ct == want {
			return true
		}
	}
	return false
}

func WithContentType(ct ContentType) metadata.HeaderOption {
	return metadata.SetOption(metadata.ContentTypeKey, ct.String())
}

/*type ContentTypeMetadataRecognizer interface {
	metadata.HeaderOption
}*/

// ContentTyped is an interface that contains and/or supports one content type.
//nolint:revive
type ContentTyped interface {
	ContentType() ContentType
}

// ContentTypeSupporter supports potentially multiple content types.
//nolint:revive
type ContentTypeSupporter interface {
	// Order _might_ carry a meaning
	SupportedContentTypes() ContentTypes
}
