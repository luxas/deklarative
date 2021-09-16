package yaml

import (
	"io"
	"sync"

	"github.com/luxas/deklarative/content"
	"github.com/luxas/deklarative/json"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Node = yaml.Node

//type RNode = yaml.RNode

//func NewRNode(n *Node) *RNode { return yaml.NewRNode(n) }

// TODO: This should maybe be a bit more robust, and check for if the DocumentNode's
// children is zero, and not just check if it null.
func IsEmptyDoc(n *Node) bool { return yaml.IsYNodeEmptyDoc(n) }

var (
//_ content.OptionDisallowUnknownFields = &DecoderOptions{}
)

func defaultDecoderOpts() *DecoderOptions {
	return &DecoderOptions{
		// match yaml.v3 default
		UnknownFieldsPolicy: content.UnknownFieldsPolicyIgnore,
	}
}

type DecoderOption interface {
	applyToDecoder(*DecoderOptions)
}

type DecoderOptions struct {
	UnknownFieldsPolicy content.UnknownFieldsPolicy // default: Ignore

	// AutoRecognizeSeqIndent *bool: TODO

	// SupportNonStringKeys *bool needed?

	// content.OptionCaseSensitiveGetter is always case-sensitive
	// content.OptionDefaultFieldNamingGetter is always lowercase
	// content.OptionDisallowDuplicateFieldsGetter is always true
	// content.OptionUnknownNumberStrategyGetter is always int64 then float64
}

func (o *DecoderOptions) applyToDecoder(target *DecoderOptions) {
	if content.ValidUnknownFieldsPolicy(o.UnknownFieldsPolicy) {
		target.UnknownFieldsPolicy = o.UnknownFieldsPolicy
	}
}

func (o *DecoderOptions) applyOptions(opts []DecoderOption) *DecoderOptions {
	for _, opt := range opts {
		opt.applyToDecoder(o)
	}
	return o
}

func (o *DecoderOptions) toJSONOpts() *json.DecoderOptions {
	return &json.DecoderOptions{
		UnknownFieldsPolicy:   o.UnknownFieldsPolicy,
		DuplicateFieldsPolicy: content.DuplicateFieldsPolicyError,
		UnknownNumberStrategy: content.UnknownNumberStrategyInt64OrFloat64,
	}
}

func NewDecoder(r io.Reader, opts []DecoderOption) *Decoder {
	return &Decoder{
		opts: *defaultDecoderOpts().applyOptions(opts),
		once: &sync.Once{},
		r:    r,
	}
}

// Once the first Decode call is called, the decoder configuration doesn't change.
type Decoder struct {
	opts DecoderOptions

	once *sync.Once
	r    io.Reader

	d *yaml.Decoder
}

func (d *Decoder) KnownFields(knownFieldsOnly bool) {
	if knownFieldsOnly {
		d.opts.UnknownFieldsPolicy = content.UnknownFieldsPolicyError
	}
	d.opts.UnknownFieldsPolicy = content.UnknownFieldsPolicyIgnore
}

func (d *Decoder) Decode(into interface{}) error {
	d.once.Do(func() {
		d.d = yaml.NewDecoder(d.r)
	})

	// If this is a YAML node; fast-path decode it directly.
	if isYAMLNode(into) {
		return d.d.Decode(into)
	}

	// Convert the YAML to an JSON-able object.
	jsonObj, err := yamlUnmarshal(func(into interface{}) error {
		return d.d.Decode(into)
	})

	j, err := json.Marshal(jsonObj)
	if err != nil {
		return err
	}

	return json.Unmarshal(j, into, d.opts.toJSONOpts())
}

func (d *Decoder) DecodeFrame() (content.Frame, error) {
	n := &Node{}
	if err := d.Decode(n); err != nil {
		return nil, err
	}
	// TODO: Detect the sequence indentation style, and use it when
	// marshalling
	seqIndent := yaml.CompactSequenceStyle

	content, err := Marshal(n) // TODO: Apply seqIndent
	if err != nil {
		return nil, err
	}

	return &frame{content, n, seqIndent}, nil
}

type Frame interface {
	content.Frame

	SequenceIndentStyle() SequenceIndentStyle
	YAMLNode() *Node
}

var _ content.Frame = &frame{}

type frame struct {
	content   []byte
	n         *Node
	seqIndent SequenceIndentStyle
}

func (f *frame) ContentType() content.ContentType { return content.ContentTypeJSON }
func (f *frame) Content() []byte                  { return f.content }
func (f *frame) DecodedGeneric() interface{}      { return f.n }
func (f *frame) IsEmpty() bool                    { return IsEmptyDoc(f.n) }

func (f *frame) SequenceIndentStyle() SequenceIndentStyle { return f.seqIndent }
func (f *frame) YAMLNode() *Node                          { return f.n }

type SequenceIndentStyle = yaml.SequenceIndentStyle
