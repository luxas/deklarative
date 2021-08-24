package traceyaml

import (
	"sync"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SpanInfo captures all events, errors, names, attributes, configuration
// and children that can be registered to a span in the order they were
// registered. JSON tags exist on all type such that it can be marshalled
// to JSON and/or YAML easily.
type SpanInfo struct {
	SpanName string `json:"spanName" yaml:"spanName"`

	Attributes Attributes `json:"attributes,omitempty" yaml:"attributes,omitempty"`
	Errors     []Error    `json:"errors,omitempty" yaml:"errors,omitempty"`
	Events     []Event    `json:"events,omitempty" yaml:"events,omitempty"`

	StartConfig *SpanConfig `json:"startConfig,omitempty" yaml:"startConfig,omitempty"`
	EndConfig   *SpanConfig `json:"endConfig,omitempty" yaml:"endConfig,omitempty"`

	StatusChanges []Status `json:"statusChanges,omitempty" yaml:"statusChanges,omitempty"`
	NameChanges   []string `json:"nameChanges,omitempty" yaml:"nameChanges,omitempty"`

	Children []*SpanInfo `json:"children,omitempty" yaml:"children,omitempty"`
	mu       *sync.Mutex
	isChild  bool
}

// Event represents an event registered using span.AddEvent().
type Event struct {
	Name        string `json:"name" yaml:"name"`
	EventConfig `json:",inline,omitempty" yaml:",inline,omitempty"`
}

// Error represents an error registered using span.RecordError().
type Error struct {
	Error       string `json:"error" yaml:"error"`
	EventConfig `json:",inline,omitempty" yaml:",inline,omitempty"`
}

// EventConfig is created from []trace.EventOption.
type EventConfig struct {
	Attributes Attributes `json:"attributes,omitempty" yaml:"attributes,omitempty"`
}

// Status represents a status update registered using s.Span.SetStatus().
type Status struct {
	Code        codes.Code `json:"code" yaml:"code"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
}

// SpanConfig is created from []trace.SpanStartOption or []trace.SpanEndOption.
type SpanConfig struct {
	Attributes Attributes     `json:"attributes,omitempty" yaml:"attributes,omitempty"`
	Links      []trace.Link   `json:"links,omitempty" yaml:"links,omitempty"`
	NewRoot    bool           `json:"newRoot,omitempty" yaml:"newRoot,omitempty"`
	SpanKind   trace.SpanKind `json:"spanKind,omitempty" yaml:"spanKind,omitempty"`
}

// Attributes is a map between an attribute key and value, as defined by
// OpenTelemetry. If the same key is added twice, the latter value is persisted.
type Attributes map[string]interface{}
