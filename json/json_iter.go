package json

import (
	"reflect"
	"strconv"
	"sync"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/luxas/deklarative/content"
	"github.com/modern-go/reflect2"
)

// NOTE: This is in part copied from
// https://github.com/kubernetes/apimachinery/blob/v0.22.0/pkg/runtime/serializer/json/json.go#L113-L184

type customExtension struct {
	jsoniter.DummyExtension

	decodeInt64OrFloat64 bool
	zeroEncodePolicy     content.ZeroEncodePolicy
}

func (e *customExtension) CreateDecoder(typ reflect2.Type) jsoniter.ValDecoder {
	// Only return something custom if enabled
	if !e.decodeInt64OrFloat64 {
		return nil
	}

	if typ.String() == "interface {}" {
		return customNumberDecoder{}
	}
	// This really feels like a hack, but allows users to decode into a named
	// interface{} alias, e.g. type Generic interface{} in this package.
	type1 := typ.Type1()
	if typ.Kind() == reflect.Interface && type1 != nil && type1.NumMethod() == 0 {
		return customNumberDecoder{}
	}
	return nil
}

var isZeroerType = reflect2.TypeOfPtr((*content.IsZeroer)(nil)).Elem()

func (e *customExtension) DecorateEncoder(typ reflect2.Type, encoder jsoniter.ValEncoder) jsoniter.ValEncoder {
	// Opt-in-only
	if !content.IsValidZeroEncodePolicy(e.zeroEncodePolicy) {
		return encoder
	}

	enableIsZeroer := e.zeroEncodePolicy&content.CheckIsZero != 0
	if enableIsZeroer && typ.Implements(isZeroerType) {
		return &isZeroValEncoder{encoder, typ}
	}

	enableIsZeroerPtr := e.zeroEncodePolicy&content.CheckIsZeroPointer != 0
	ptrType := reflect2.PtrTo(typ)
	if enableIsZeroerPtr && ptrType.Implements(isZeroerType) {
		return &referenceEncoder{&isZeroValEncoder{encoder, ptrType}}
	}

	// TODO: Make this check recursive for structs; somehow.
	// Maybe pass the jsoniter.API to the composite ValEncoder implementation
	// such that, at runtime, the (cached) ValEncoder for the fields can be
	// looked up from the struct's type fields. The logic would be along the lines of:
	/*
		- If the type here is of kind Struct; return a ValEncoder that
		  contains the fields of the struct (statically); then at runtime
		  uses the jsoniter.API to get the right ValEncoder for the right
		  type (these can even be cached), and then finally uses a
		  field.UnsafeGet before calling the ValEncoder.IsEmpty function.
	*/
	return encoder
}

type referenceEncoder struct{ underlying jsoniter.ValEncoder }

func (e *referenceEncoder) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	e.underlying.Encode(ptr, stream)
}

func (e *referenceEncoder) IsEmpty(ptr unsafe.Pointer) bool {
	return e.underlying.IsEmpty(unsafe.Pointer(&ptr))
}

type isZeroValEncoder struct {
	underlying jsoniter.ValEncoder
	valType    reflect2.Type
}

func (e *isZeroValEncoder) IsEmpty(ptr unsafe.Pointer) bool {
	obj := e.valType.UnsafeIndirect(ptr)
	if e.valType.IsNullable() && reflect2.IsNil(obj) {
		return true
	}
	return (obj).(content.IsZeroer).IsZero()
}
func (e *isZeroValEncoder) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	e.underlying.Encode(ptr, stream)
}

// TODO: Maybe make this a composite/"decorated" encoder? That might
// be faster/more efficient/correct?
type customNumberDecoder struct{}

func (customNumberDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() { //nolint:exhaustive
	case jsoniter.NumberValue:
		var number jsoniter.Number
		iter.ReadVal(&number)
		i64, err := strconv.ParseInt(string(number), 10, 64)
		if err == nil {
			*(*interface{})(ptr) = i64
			return
		}
		f64, err := strconv.ParseFloat(string(number), 64)
		if err == nil {
			*(*interface{})(ptr) = f64
			return
		}
		iter.ReportError("DecodeNumber", err.Error())
	default:
		*(*interface{})(ptr) = iter.Read()
	}
}

func buildJSONIterAPI(c *jsoniterConfig) jsoniter.API {
	useNumber := c.unknownNumberStrategy == content.UnknownNumberStrategyJSONNumber
	decodeInt64OrFloat := c.unknownNumberStrategy == content.UnknownNumberStrategyInt64OrFloat64
	disallowUnknownFields := c.unknownFieldsPolicy == content.UnknownFieldsPolicyError

	config := jsoniter.Config{
		EscapeHTML:             c.escapeHTML,
		SortMapKeys:            true,
		ValidateJsonRawMessage: true,
		CaseSensitive:          true,

		DisallowUnknownFields: disallowUnknownFields,
		UseNumber:             useNumber,
	}.Froze()

	config.RegisterExtension(&customExtension{
		decodeInt64OrFloat64: decodeInt64OrFloat,
		zeroEncodePolicy:     c.zeroEncodePolicy,
	})
	return config
}

var (
	jsoniterPool   = map[jsoniterConfig]jsoniter.API{}
	jsoniterPoolMu = &sync.Mutex{}
)

// This struct MUST be comparable by value with other structs of the same type,
// and hence, not contain any pointers, maps or slices ("reference types").
type jsoniterConfig struct {
	unknownFieldsPolicy   content.UnknownFieldsPolicy
	duplicateFieldsPolicy content.DuplicateFieldsPolicy
	unknownNumberStrategy content.UnknownNumberStrategy

	escapeHTML       bool
	zeroEncodePolicy content.ZeroEncodePolicy
}

func jsoniterForConfig(c jsoniterConfig) jsoniter.API {
	jsoniterPoolMu.Lock()
	defer jsoniterPoolMu.Unlock()

	// Return cached API if exists for the config
	if api, ok := jsoniterPool[c]; ok {
		return api
	}

	jsoniterPool[c] = buildJSONIterAPI(&c)
	return jsoniterPool[c]
}
