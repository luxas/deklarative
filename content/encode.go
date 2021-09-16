package content

import "strings"

type Encoder interface {
	Encode(obj interface{}) error
}

type EncoderOption interface {
	ApplyToEncoder(target EncoderOptionTarget)
}

type EncoderOptionTarget interface {
	SetIndent(string)                   // Only configurable for JSON
	SortMapKeys(bool)                   //
	LowercaseEverythingLikeYAMLv3(bool) // Or just print the field name as-is
	SeqIndentStyle(string)              // YAML-specific
}

// TODO: Copy over godoc from yaml.v3
type IsZeroer interface {
	IsZero() bool
}

type ZeroEncodePolicy int

const (
	CheckIsZero ZeroEncodePolicy = 1 << iota
	CheckIsZeroPointer
	CheckIsZeroStructRecursive
)
const checkIsZeroInclusiveUpperbound = CheckIsZeroStructRecursive<<1 - 1

func (p ZeroEncodePolicy) String() string {
	modes := []string{}
	if p&CheckIsZero != 0 {
		modes = append(modes, "Plain")
	}
	if p&CheckIsZeroPointer != 0 {
		modes = append(modes, "Pointer")
	}
	if p&CheckIsZeroStructRecursive != 0 {
		modes = append(modes, "StructRecursive")
	}
	return strings.Join(modes, ",")
}

func IsValidZeroEncodePolicy(p ZeroEncodePolicy) bool {
	return CheckIsZero <= p && p <= checkIsZeroInclusiveUpperbound
}
