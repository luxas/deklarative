package yaml

import (
	"fmt"

	"github.com/luxas/deklarative/content"
	"github.com/luxas/deklarative/json"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

type MarshalOptions struct {
	ZeroEncodePolicy content.ZeroEncodePolicy
}

// Marshal marshals the object into JSON then converts JSON to YAML and returns the
// YAML.
func Marshal(obj interface{}) ([]byte, error) {
	// Be compatible with e.g. kyaml.Marshal.
	// TODO: What to do here if a Node is embedded within the object?
	if isYAMLNode(obj) {
		return yamlMarshal(obj)
	}

	// Marshal using JSON, to keep struct tags.
	j, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("error marshaling into JSON: %v", err)
	}

	y, err := JSONToYAML(j)
	if err != nil {
		return nil, fmt.Errorf("error converting JSON to YAML: %v", err)
	}

	return y, nil
}

type JSONToYAMLOptions struct {
	// DuplicateFieldsPolicy is always Error; it's invalid YAML

	// TODO: Would the UseNumber option even be valid here?
	// UnknownNumberStrategy is always Int64OrFloat64; yaml.v3 default

	EncoderOptions
}

type EncoderOptions struct {
	// Do not expose the indent option now; to avoid fragmentation atm.

	kyaml.EncoderOptions

	// TODO: Option for auto-detecting the indent style?
	// It should probably be an enum then, with an extra option
}

// JSONToYAML Converts JSON to YAML.
// If there are duplicate fields in the input JSON, an error will be returned.
func JSONToYAML(j []byte) ([]byte, error) {
	// Convert the JSON to an object.
	var obj interface{}
	// The disallow known fields option here doesn't matter as we're marshalling
	// into an interface{}.
	// TODO: Always error on duplicate fields
	// TODO: Always UnknownNumberStrategyInt64OrFloat64
	if err := json.Unmarshal(j, &obj); err != nil {
		return nil, err
	}

	// Marshal this object into YAML.
	// TODO: Marshal using kyaml's sequence indent option
	return yamlMarshal(obj)
}

func isYAMLNode(obj interface{}) bool {
	_, isNode := obj.(Node)
	_, isNodePtr := obj.(*Node)
	return isNode || isNodePtr
}

func yamlMarshal(obj interface{}) ([]byte, error) {
	return kyaml.MarshalWithOptions(obj, &kyaml.EncoderOptions{
		SeqIndent: kyaml.CompactSequenceStyle,
	})
}

type UnmarshalOptions struct {
	UnknownFieldsPolicy content.UnknownFieldsPolicy
}

// Unmarshal converts YAML to JSON then uses JSON to unmarshal into an object,
// optionally configuring the behavior of the JSON unmarshal.
func Unmarshal(y []byte, into interface{}) error {
	return unmarshal(y, into, false)
}

// UnmarshalStrict strictly converts YAML to JSON then uses JSON to unmarshal
// into an object, optionally configuring the behavior of the JSON unmarshal.
func UnmarshalStrict(y []byte, into interface{}) error {
	return unmarshal(y, into, true)
}

// unmarshal unmarshals the given YAML byte stream into the given interface,
// optionally performing the unmarshalling strictly
func unmarshal(y []byte, into interface{}, strict bool) error {
	// If this is a YAML node; fast-path decode it directly.
	if isYAMLNode(into) {
		return kyaml.Unmarshal(y, into)
	}

	// No need to provide any options here, not for the time being at least,
	// as cosmetic changes aren't needed anyways.
	j, err := YAMLToJSON(y)
	if err != nil {
		return fmt.Errorf("error converting YAML to JSON: %v", err)
	}

	// TODO: json decoder options here!
	// Always disallow duplicate fields, and enable int64orfloat64
	if err := json.Unmarshal(j, into); err != nil {
		return fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	return nil
}

type YAMLToJSONOptions struct {
	JSONEncoderOpts json.EncoderOptions
}

// YAMLToJSON converts YAML to JSON. Since JSON is a subset of YAML,
// passing JSON through this method should be a no-op.
//
// Things YAML can do that are not supported by JSON:
// * In YAML you can have binary and null keys in your maps. These are invalid
//   in JSON. (int and float keys are converted to strings.)
// * Binary data in YAML with the !!binary tag is not supported. If you want to
//   use binary data with this library, encode the data as base64 as usual but do
//   not use the !!binary tag in your YAML. This will ensure the original base64
//   encoded data makes it all the way through to the JSON.
//
// If there are duplicate fields, an error is always returned.
// TODO: Rewrite this spec.
func YAMLToJSON(y []byte) ([]byte, error) {
	// The disallow unknown fields option doesn't matter here as the decode
	// target is an interface{}.
	jsonObj, err := yamlUnmarshalBytes(y)
	if err != nil {
		return nil, err
	}

	// Convert this object to JSON and return the data.
	// TODO: JSON encode options here
	return json.Marshal(jsonObj)
}

func yamlUnmarshal(decodeFn func(into interface{}) error) (interface{}, error) {
	var yamlObj interface{}
	if err := decodeFn(yamlObj); err != nil {
		return nil, err
	}

	return convertNonStringMapKeys(yamlObj)
}

func yamlUnmarshalBytes(y []byte) (interface{}, error) {
	return yamlUnmarshal(func(into interface{}) error {
		return kyaml.Unmarshal(y, into)
	})
}

func ToJSONGeneric(n *Node) (interface{}, error) {
	return yamlUnmarshal(func(into interface{}) error {
		return n.Decode(into)
	})
}

// convertNonStringMapKeys traverses an unstructured object
// to find any occurrences of map[interface{}]interface{} that it
// can convert to map[string]interface{}, as required by the JSON
// specification. YAML allows non-string keys, hence this "conversion"
// is required. It automatically disregards NaN and Infinity values,
// as per the JSON spec.
//
// This is still round-trippable, as the json.Decoder allows mapping
// string-encoded map keys into ints, floats and booleans at
// decode-time, hence, we're not losing data.
//
// TODO: Consider forcing obj to be a pointer, such that the signature
// for this function can be convertNonStringMapKeys(obj interface{}) error
// (this would probably?) save some memory and time.
func convertNonStringMapKeys(obj interface{}) (interface{}, error) {
	var err error
	switch m := obj.(type) {
	case []interface{}:
		for i, v := range m {
			m[i], err = convertNonStringMapKeys(v)
			if err != nil {
				return nil, err
			}
		}
		return m, nil
	case map[string]interface{}:
		for k, v := range m {
			m[k], err = convertNonStringMapKeys(v)
			if err != nil {
				return nil, err
			}
		}
		return m, nil
	case map[interface{}]interface{}:
		// json-iter.*Stream.WriteFloat64 has the following logic:
		// if math.IsInf(val, 0) || math.IsNaN(val) { // then error }
		// which means that unsupported float values in JSON will be
		// caught and reported.
		marshalled, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		var retval map[string]interface{}
		if err := json.Unmarshal(marshalled, &retval); err != nil {
			return nil, err
		}
		return retval, nil
	default:
		return obj, nil
	}
}
