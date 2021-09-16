package content

import "io"

type FrameDecoder interface {
	DecodeFrame() (Frame, error)

	//SupportsContentType(ct ContentType) bool
}

type Decoder interface {
	Decode(into interface{}) error
}

/*type RecognizingDecoder interface {
	RecognizesData() bool
}*/

type ReadAllFrameDecoderCreator struct {
	ContentType ContentType
}

func (c ReadAllFrameDecoderCreator) NewFrameDecoder(r io.Reader) FrameDecoder {
	return ReadAllFrameDecoder(r, c.ContentType)
}

func (ReadAllFrameDecoderCreator) SupportsContentType(ct ContentType) bool { return true }

func ReadAllFrameDecoder(r io.Reader, ct ContentType) FrameDecoder {
	return &readAllFrameDecoder{r, ct, false}
}

type readAllFrameDecoder struct {
	r           io.Reader
	ct          ContentType
	hasBeenRead bool
}

func (r *readAllFrameDecoder) DecodeFrame() (Frame, error) {
	if r.hasBeenRead {
		return nil, io.EOF
	}
	r.hasBeenRead = true

	data, err := io.ReadAll(r.r)
	if err != nil {
		return nil, err
	}

	return NewFrame(r.ct, data, nil, len(data) == 0), nil
}

/*
type DecoderOption interface {
	ApplyToDecoder(target DecoderOptionTarget)
}*/

type OptionDisallowUnknownFields interface {
	// ApplyDisallowUnknownFields applies the DisallowUnknownFields option.
	// If DisallowUnknownFields == true, only known fields will be decoded, and if there are
	// unknown fields in the serialized bytes, the decoder will error. For
	// DisallowUnknownFields == false, the decoder will silently ignore unknown fields.
	ApplyDisallowUnknownFields(DisallowUnknownFields bool)
}

type OptionDisallowUnknownFieldsGetter interface {
	// GetDisallowUnknownFields gets whether the implementer disallows unknown fields or not.
	//
	// Decoders defaulting this to false, but allows overriding:
	//
	//	encoding/json
	//	json-iterator
	//	yaml.v3
	//	kyaml
	//
	// Decoders defaulting this to false, without an option to override:
	//
	//	k8s.io/apimachinery/pkg/util/json
	//
	// Decoders defaulting this to false, but allows overriding to the same value as
	// the value of GetDisallowDuplicateFields() (i.e. either strict or non-strict):
	//
	//	yaml.v2
	//	sigs.k8s.io/yaml
	//	k8s.io/apimachinery/pkg/runtime/serializer/json
	//
	GetDisallowUnknownFields() bool
}

// TODO: Maybe make a OptionDisallowDuplicateFields type

type OptionDisallowDuplicateFieldsGetter interface {
	// GetDisallowUnknownFields gets whether the implementer disallows duplicate fields or not.
	//
	// Decoders hardcoding this to false (i.e. duplicate fields are allowed):
	//
	//	encoding/json
	//	k8s.io/apimachinery/pkg/util/json
	//
	// Decoders hardcoding this to true:
	//
	// 	yaml.v3
	//	kyaml
	//	json-iterator
	//
	// Decoders defaulting this to false, but allows overriding to the same value as
	// the value of GetDisallowUnknownFields() (i.e. either strict or non-strict):
	//
	//	yaml.v2
	//	sigs.k8s.io/yaml
	//	k8s.io/apimachinery/pkg/runtime/serializer/json
	//
	GetDisallowDuplicateFields() bool
}

type OptionCaseSensitiveGetter interface {
	// OptionCaseSensitiveGetter gets whether the implementer is case-sensitive or not.
	//
	// Decoders hardcoding this to false (i.e. are in-case-sensitive):
	//
	//	encoding/json
	//	k8s.io/apimachinery/pkg/util/json
	//	sigs.k8s.io/yaml
	//
	// Decoders defaulting to false, but allows configuration:
	//
	//	json-iterator
	//
	// Decoders hardcoding this to true:
	//
	//	yaml.v2
	// 	yaml.v3
	//	kyaml
	//	k8s.io/apimachinery/pkg/runtime/serializer/json
	//
	GetCaseSensitive() bool
}

type OptionDefaultFieldNamingGetter interface {
	// GetDefaultFieldNaming returns what the default naming policy is for fields of typed
	// structs if no field tag is set.
	//
	// Decoders and encoders defaulting this to NamingConventionLowercase:
	//
	//	yaml.v2
	// 	yaml.v3
	//	kyaml
	//
	// Decoders and encoders defaulting this to NamingConventionFieldName:
	//
	//	encoding/json
	//	json-iter?
	//	k8s.io/apimachinery/pkg/runtime/serializer/json
	//	k8s.io/apimachinery/pkg/util/json
	//	sigs.k8s.io/yaml
	//
	GetDefaultFieldNaming() NamingConvention
}

type NamingConvention int

const (
	NamingConventionLowercase NamingConvention = iota
	NamingConventionFieldName
)

/*type OptionUseNumber interface {
	ApplyUseNumber(useNumber bool)
}

type OptionUseNumberGetter interface {
	GetUseNumber() bool
}

type OptionConvertUnstructuredNumbers interface {
	ApplyConvertUnstructuredNumbers(convert bool)
}

type OptionConvertUnstructuredNumbersGetter interface {
	GetConvertUnstructuredNumbers(convert bool)
}*/

type OptionUnknownNumberStrategy interface {
	// ApplyUnknownNumberStrategy applies the given UnknownNumberStrategy option.
	// If strategy is UnknownNumberStrategyAlwaysFloat64, any number without a given
	// Go number type will be assigned float64. This can, however, lead to loss of
	// precision, as not every integer can be expressed as a float64. Hence, there is
	// UnknownNumberStrategyInt64OrFloat64, which first tries to parse the number as
	// an int64, and only if that fails, assign it to a float64. This leads to that
	// integers can always be round-tripped losslessly. The third option,
	// UnknownNumberStrategyJSONNumber leaves the value as a string, with Int64() and
	// Float64() methods (see encoding/json.Number).
	ApplyUnknownNumberStrategy(strategy UnknownNumberStrategy)
}

type OptionUnknownNumberStrategyGetter interface {
	// GetUnknownNumberStrategy returns what unknown number strategy is used.
	//
	// Decoders that default to UnknownNumberStrategyAlwaysFloat64, but allow
	// use of the json.Number method:
	//
	//	encoding/json
	//	json-iterator
	//
	// Decoders that default to UnknownNumberStrategyAlwaysFloat64, but do not
	// allow decoding into json.Number:
	//
	//	sigs.k8s.io/yaml
	//
	// Decoders that default to UnknownNumberStrategyInt64OrFloat64, and do not
	// allow configuring:
	//
	//	yaml.v2
	// 	yaml.v3
	//	kyaml
	//	k8s.io/apimachinery/pkg/util/json
	//	k8s.io/apimachinery/pkg/runtime/serializer/json
	//
	GetUnknownNumberStrategy() UnknownNumberStrategy
}

type UnknownNumberStrategy int

const (
	UnknownNumberStrategyAlwaysFloat64 UnknownNumberStrategy = 1 + iota
	UnknownNumberStrategyInt64OrFloat64
	UnknownNumberStrategyJSONNumber
)

func (s UnknownNumberStrategy) String() string {
	switch s {
	case UnknownNumberStrategyAlwaysFloat64:
		return "AlwaysFloat64"
	case UnknownNumberStrategyInt64OrFloat64:
		return "Int64OrFloat64"
	case UnknownNumberStrategyJSONNumber:
		return "JSONNumber"
	default:
		return ""
	}
}

func ValidUnknownNumberStrategy(strategy UnknownNumberStrategy) bool {
	return UnknownNumberStrategyAlwaysFloat64 <= strategy &&
		strategy <= UnknownNumberStrategyJSONNumber
}

type DuplicateFieldsPolicy int

const (
	DuplicateFieldsPolicyIgnore DuplicateFieldsPolicy = 1 + iota
	DuplicateFieldsPolicyError
	// DuplicateFieldsPolicyWarn
)

func (p DuplicateFieldsPolicy) String() string {
	switch p {
	case DuplicateFieldsPolicyIgnore:
		return "Ignore"
	case DuplicateFieldsPolicyError:
		return "Error"
	default:
		return ""
	}
}

func ValidDuplicateFieldsPolicy(policy DuplicateFieldsPolicy) bool {
	return DuplicateFieldsPolicyIgnore <= policy &&
		policy <= DuplicateFieldsPolicyError
}

type UnknownFieldsPolicy int

const (
	UnknownFieldsPolicyIgnore UnknownFieldsPolicy = 1 + iota
	UnknownFieldsPolicyError
	// UnknownFieldsPolicyWarn
)

func (p UnknownFieldsPolicy) String() string {
	switch p {
	case UnknownFieldsPolicyIgnore:
		return "Ignore"
	case UnknownFieldsPolicyError:
		return "Error"
	default:
		return ""
	}
}

func ValidUnknownFieldsPolicy(policy UnknownFieldsPolicy) bool {
	return UnknownFieldsPolicyIgnore <= policy &&
		policy <= UnknownFieldsPolicyError
}

/*type DecoderOptionTarget interface {
	UseNumber(bool)
	AllowNonStringKeys(bool) // TODO: Figure this out
	CaseSensitive(bool)      // Always true?
}*/
