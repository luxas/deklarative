package yaml

import (
	"bufio"
	"bytes"

	"github.com/luxas/deklarative/content"
	"github.com/luxas/deklarative/json"
	"gopkg.in/yaml.v3"
)

const Recognizer = recognizer(0)

var _ content.PeekRecognizer = Recognizer

type recognizer int

func (recognizer) FromPeekBytes(peek []byte) content.ContentType {
	if ct := json.Recognizer.FromPeekBytes(peek); len(ct) != 0 {
		return ct
	}
	if isYAML(peek) {
		return content.ContentTypeYAML
	}
	return ""
}

func (recognizer) SupportedContentTypes() content.ContentTypes {
	cts := json.Recognizer.SupportedContentTypes()
	cts = append(cts, content.ContentTypeYAML)
	return cts
}

// TODO: Use the approach of reading as many lines as we can, and then putting
// that into yaml.Unmarshal into a *Node.
func isYAML(peek []byte) bool {
	line, err := getLine(peek)
	if err != nil {
		return false
	}

	o := map[string]interface{}{}
	err = yaml.Unmarshal(line, &o)
	return err == nil && len(o) != 0
}

// TODO: Use yaml.LineReader instead?
func getLine(peek []byte) ([]byte, error) {
	s := bufio.NewScanner(bytes.NewReader(peek))
	// TODO: Support very long lines? (over 65k bytes?) Probably not
	for s.Scan() {
		t := bytes.TrimSpace(s.Bytes())
		if len(t) == 0 || bytes.Equal(t, []byte("---")) || bytes.HasPrefix(t, []byte{'#'}) {
			continue
		}
		return t, nil
	}
	// Return a possible scanning error
	if err := s.Err(); err != nil {
		return nil, err
	}
	//
	return nil, nil
}
