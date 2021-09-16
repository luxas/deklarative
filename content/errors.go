package content

import (
	"fmt"

	"github.com/luxas/deklarative/content/structerr"
)

// RecognizeError is returned from NewRecognizingReader and
// NewRecognizingWriter if recognition failed.
type RecognizeError struct {
	Metadata   Metadata
	Peek       []byte
	Underlying error
}

// RecognizeError may carry an underlying error cause.
var _ structerr.UnwrapError = &RecognizeError{}

func (e *RecognizeError) Error() string {
	msg := "couldn't recognize content type"
	if e.Underlying != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Underlying)
	}
	return msg
}

func (e *RecognizeError) Is(target error) bool {
	//nolint:errorlint
	_, ok := target.(*RecognizeError)
	return ok
}

func (e *RecognizeError) Unwrap() error { return e.Underlying }

// Enforce all struct errors implementing structerr.StructError.
var _ structerr.StructError = &UnsupportedContentTypeError{}

// ErrUnsupportedContentType creates a new *UnsupportedContentTypeError.
func ErrUnsupportedContentType(unsupported ContentType, supported ...ContentType) *UnsupportedContentTypeError {
	return &UnsupportedContentTypeError{Unsupported: unsupported, Supported: supported}
}

// UnsupportedContentTypeError describes that the supplied content type is not supported by an
// implementation handling different content types.
//
// This error can be checked for equality using errors.Is(err, &UnsupportedContentTypeError{}).
type UnsupportedContentTypeError struct {
	// Unsupported is the content type that was given but not supported
	// +required
	Unsupported ContentType
	// Supported is optional; if len(Supported) != 0, it lists the content types that indeed
	// are supported by the implementation. If len(Supported) == 0, it should not be used
	// as an indicator.
	// +optional
	Supported []ContentType
}

func (e *UnsupportedContentTypeError) Error() string {
	msg := fmt.Sprintf("unsupported content type: %q", e.Unsupported)
	if len(e.Supported) != 0 {
		msg = fmt.Sprintf("%s. supported content types: %v", msg, e.Supported)
	}
	return msg
}

func (e *UnsupportedContentTypeError) Is(target error) bool {
	//nolint:errorlint
	_, ok := target.(*UnsupportedContentTypeError)
	return ok
}
