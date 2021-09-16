package json

import (
	"bytes"
	"io"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/luxas/deklarative/content"
)

type EncoderOption interface {
	applyToEncoder(*EncoderOptions)
}

func defaultEncoderOpts() *EncoderOptions {
	return &EncoderOptions{
		// match encoding/json default
		EscapeHTML: boolVar(true),
		// match yaml.v3 default
		ZeroEncodePolicy: content.CheckIsZero |
			content.CheckIsZeroPointer |
			content.CheckIsZeroStructRecursive,
	}
}
func boolVar(b bool) *bool { return &b }

type EncoderOptions struct {
	Indent           string
	Prefix           string
	EscapeHTML       *bool
	ZeroEncodePolicy content.ZeroEncodePolicy
}

func (o *EncoderOptions) applyToEncoder(target *EncoderOptions) {
	if len(o.Indent) != 0 {
		target.Indent = o.Indent
	}
	if len(o.Prefix) != 0 {
		target.Prefix = o.Prefix
	}
	if o.EscapeHTML != nil {
		target.EscapeHTML = o.EscapeHTML
	}
	if content.IsValidZeroEncodePolicy(o.ZeroEncodePolicy) {
		target.ZeroEncodePolicy = o.ZeroEncodePolicy
	}
}

func (o *EncoderOptions) applyOptions(opts []EncoderOption) *EncoderOptions {
	for _, opt := range opts {
		opt.applyToEncoder(o)
	}
	return o
}

func (o *EncoderOptions) applyToJSONIterConfig(c *jsoniterConfig) {
	c.escapeHTML = *o.EscapeHTML
	c.zeroEncodePolicy = o.ZeroEncodePolicy
}

func (o *EncoderOptions) toJSONIter() jsoniter.API {
	c := jsoniterConfig{}
	// apply the default decoding options for the given API
	defaultDecoderOpts().applyToJSONIterConfig(&c)
	// apply all contained encoding options
	o.applyToJSONIterConfig(&c)
	return jsoniterForConfig(c)
}

type Encoder struct {
	opts EncoderOptions
	once *sync.Once
	w    io.Writer

	e *jsoniter.Encoder
}

func (e *Encoder) SetIndent(prefix, indent string) {
	e.opts.Prefix = prefix
	e.opts.Indent = indent
}

func (e *Encoder) SetEscapeHTML(escape bool) {
	e.opts.EscapeHTML = &escape
}

func (e *Encoder) Encode(obj interface{}) error {
	e.once.Do(func() {
		e.e = e.opts.toJSONIter().NewEncoder(e.w)
	})

	// Quick path: encode without any indentation
	if len(e.opts.Prefix) == 0 && len(e.opts.Indent) == 0 {
		return e.e.Encode(obj)
	}

	// Slower path, marshal, then
	out, err := MarshalIndent(obj, e.opts.Prefix, e.opts.Indent, &e.opts)
	if err != nil {
		return err
	}

	// TODO: Is this safe to do? Shall a newline be appended?
	_, err = e.w.Write(out)
	return err
	//return e.e.Encode(RawMessage(out))
}

func Marshal(obj interface{}, opts ...EncoderOption) ([]byte, error) {
	o := defaultEncoderOpts().applyOptions(opts)
	return o.toJSONIter().Marshal(obj)
}
func MarshalIndent(obj interface{}, prefix, indent string, opts ...EncoderOption) ([]byte, error) {
	// Marshal "the normal way" first; without any indentation
	// (even though opts might include indentation settings)
	// TODO: If json-iter would marshal e.g. RawMessages correctly, we wouldn't
	// need to do this. TODO: Open issues.
	out, err := Marshal(obj, opts...)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if _, err := buf.WriteString(prefix); err != nil {
		return nil, err
	}
	if err := Indent(&buf, out, prefix, indent); err != nil {
		return nil, err
	}
	// Return the indented bytes
	return buf.Bytes(), nil
}
