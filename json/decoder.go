package json

import (
	"bytes"
	"io"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/luxas/deklarative/content"
)

var (
	/*_ content.OptionDisallowUnknownFields    = &DecoderOptions{}
	_ content.OptionUnknownNumberStrategy = &DecoderOptions{}

	_ content.OptionDisallowUnknownFields    = &Decoder{}
	_ content.OptionUnknownNumberStrategy = &Decoder{}*/
	_ content.Decoder      = &Decoder{}
	_ content.FrameDecoder = &Decoder{}
)

type DecoderOption interface {
	applyToDecoder(*DecoderOptions)
}

func DisallowUnknownFields() DecoderOption {
	return &DecoderOptions{UnknownFieldsPolicy: content.UnknownFieldsPolicyError}
}

func defaultDecoderOpts() *DecoderOptions {
	return &DecoderOptions{
		// match encoding/json default
		UnknownFieldsPolicy: content.UnknownFieldsPolicyIgnore,
		// match yaml.v3 default
		DuplicateFieldsPolicy: content.DuplicateFieldsPolicyError,
		// match Kubernetes API machinery default
		UnknownNumberStrategy: content.UnknownNumberStrategyInt64OrFloat64,
	}
}

type DecoderOptions struct {
	// DisallowUnknownFields disallows decoding serialized data into structs where
	// the serialized data contain fields not present in the struct.
	// If the decode target is an interface{}, any field names are allowed.
	//
	// Default: UnknownFieldsPolicyIgnore, i.e. unknown fields in the given data are ignored.
	//
	// TODO: Make an example showing both cases.
	UnknownFieldsPolicy content.UnknownFieldsPolicy

	// DisallowDuplicateFields disallows duplicate fields.
	//
	// Default: DuplicateFieldsPolicyError, i.e. duplicate fields raise an error.
	DuplicateFieldsPolicy content.DuplicateFieldsPolicy

	// UnknownNumberStrategy controls how JSON numbers are decoded into interface{} targets.
	//
	// UnknownNumberStrategyAlwaysFloat64 all JSON numbers where the target type is unknown
	// will be decoded into float64's. This can, however, lead to loss of precision, as
	// not every integer can be expressed as a float64. For example, try decoding the
	// integer 10000000000000001 into a float64, and encoding it again. It will be encoded
	// as 10000000000000000; which examplifies the loss of round-trippability.
	//
	// Hence, there is UnknownNumberStrategyInt64OrFloat64, which first tries to parse a
	// JSON number of unknown type as an int64, and only if that fails, assign it to a
	// float64. The result is that integers that fit into int64 can always be round-tripped.
	//
	// UnknownNumberStrategyJSONNumber is the same as encoding/json.Decoder.UseNumber();
	// it leaves the number as a typed string, json.Number; preserving the exact textual
	// representation.
	//
	// Default: UnknownNumberStrategyInt64OrFloat64, which makes integers round-trippable
	// by default.
	//
	// TODO: Make an example showing all three cases.
	UnknownNumberStrategy content.UnknownNumberStrategy

	// content.OptionCaseSensitiveGetter is always case-sensitive
	// content.OptionDefaultFieldNamingGetter is always fieldName
	// content.OptionDisallowDuplicateFieldsGetter is always true
}

func (o *DecoderOptions) applyToDecoder(target *DecoderOptions) {
	if content.ValidUnknownFieldsPolicy(o.UnknownFieldsPolicy) {
		target.UnknownFieldsPolicy = o.UnknownFieldsPolicy
	}
	if content.ValidDuplicateFieldsPolicy(o.DuplicateFieldsPolicy) {
		target.DuplicateFieldsPolicy = o.DuplicateFieldsPolicy
	}
	if content.ValidUnknownNumberStrategy(o.UnknownNumberStrategy) {
		target.UnknownNumberStrategy = o.UnknownNumberStrategy
	}
}

func (o *DecoderOptions) applyOptions(opts []DecoderOption) *DecoderOptions {
	for _, opt := range opts {
		opt.applyToDecoder(o)
	}
	return o
}

func (o *DecoderOptions) applyToJSONIterConfig(c *jsoniterConfig) {
	c.unknownFieldsPolicy = o.UnknownFieldsPolicy
	c.duplicateFieldsPolicy = o.DuplicateFieldsPolicy
	c.unknownNumberStrategy = o.UnknownNumberStrategy
}

func (o *DecoderOptions) toJSONIter() jsoniter.API {
	c := jsoniterConfig{}
	// apply the default encoding options for the given API
	defaultEncoderOpts().applyToJSONIterConfig(&c)
	// apply all contained decoding options
	o.applyToJSONIterConfig(&c)
	return jsoniterForConfig(c)
}

// NewDecoder creates a new *Decoder
func NewDecoder(r io.Reader, opts ...DecoderOption) *Decoder {
	return &Decoder{
		opts: *defaultDecoderOpts().applyOptions(opts),
		once: &sync.Once{},
		r:    r,
	}
}

// Once the first Decode call is called, the decoder configuration doesn't change.
//
// TODO: Highlight differences between this and encoding/json:
// - Uses json-iter (faster!)
// - Disallows duplicate fields; actually it doesn't!!
// - Can decode maps with int, float64 and bool keys
// - Defaults to decoding int or float for roundtrips
// - Case-sensitive
//
// TODO: Open issues for Kubernetes:
// - The default serializer doesn't apply unknown fields restriction for
//   decode into (applies to both YAML and JSON).
// - The default serializer doesn't apply duplicate field restriction when
//   JSON-only, because json-iter doesn't support this.
type Decoder struct {
	opts DecoderOptions
	once *sync.Once
	r    io.Reader

	d *jsoniter.Decoder
}

func (d *Decoder) SupportedContentTypes() content.ContentTypes {
	return []content.ContentType{content.ContentTypeJSON}
}

func (d *Decoder) DisallowUnknownFields() {
	d.opts.UnknownFieldsPolicy = content.UnknownFieldsPolicyError
}
func (d *Decoder) UseNumber() {
	d.opts.UnknownNumberStrategy = content.UnknownNumberStrategyJSONNumber
}

func (d *Decoder) Decode(obj interface{}) error {
	// Initialize d.d once, given the io.Reader and options.
	d.once.Do(func() {
		d.d = d.opts.toJSONIter().NewDecoder(d.r)
	})

	return d.d.Decode(obj)
}

func (d *Decoder) DecodeFrame() (content.Frame, error) {
	f := &frame{}
	if err := d.Decode(&f.content); err != nil {
		return nil, err
	}
	if err := Unmarshal(f.content, &f.obj, &d.opts); err != nil {
		return nil, err
	}
	return f, nil
}

func Unmarshal(data []byte, v interface{}, opts ...DecoderOption) error {
	o := defaultDecoderOpts().applyOptions(opts)
	return o.toJSONIter().Unmarshal(data, v)
}

func UnmarshalStrict(data []byte, v interface{}, opts ...DecoderOption) error {
	return Unmarshal(data, v, append(opts, DisallowUnknownFields())...)
}

var _ content.Frame = &frame{}

type frame struct {
	content RawMessage
	obj     interface{}
}

func (f *frame) ContentType() content.ContentType { return content.ContentTypeJSON }
func (f *frame) Content() []byte                  { return f.content }
func (f *frame) DecodedGeneric() interface{}      { return f.obj }
func (f *frame) IsEmpty() bool                    { return bytes.Equal(f.content, nullBytes) }

var nullBytes = []byte("null")
