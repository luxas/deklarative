package json

import (
	"bytes"
	"encoding/json"
)

func Indent(buf *bytes.Buffer, src []byte, prefix, indent string) error {
	return json.Indent(buf, src, prefix, indent)
}

type RawMessage = json.RawMessage

// Number is a re-export of encoding/json.Number. It can be returned from decoded
// unstructured interface{} targets if DecoderOptions.UnknownNumberStrategy is
// UnknownNumberStrategyJSONNumber.
// TODO: Is this number completely round-trippable?
type Number = json.Number
