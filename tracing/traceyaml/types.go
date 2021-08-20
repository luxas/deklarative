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
	Names         []string    `json:"names"`
	StartConfig   *SpanConfig `json:"startConfig,omitempty"`
	Events        []Event     `json:"events,omitempty"`
	Errors        []Error     `json:"errors,omitempty"`
	StatusChanges []Status    `json:"statusChanges,omitempty"`
	Attributes    []Attribute `json:"attributes,omitempty"`
	EndConfig     *SpanConfig `json:"endConfig,omitempty"`

	Children []*SpanInfo `json:"children,omitempty"`
	mu       *sync.Mutex
	isChild  bool
}

// Event represents an event registered using span.AddEvent().
type Event struct {
	Name        string `json:"name"`
	EventConfig `json:",inline,omitempty"`
}

// Error represents an error registered using span.RecordError().
type Error struct {
	Error       string `json:"error"`
	EventConfig `json:",inline,omitempty"`
}

// EventConfig is created from []trace.EventOption.
type EventConfig struct {
	Attributes []Attribute `json:"attributes,omitempty"`
}

// Status represents a status update registered using s.Span.SetStatus().
type Status struct {
	Code        codes.Code `json:"code"`
	Description string     `json:"description,omitempty"`
}

// SpanConfig is created from []trace.SpanStartOption or []trace.SpanEndOption.
type SpanConfig struct {
	Attributes []Attribute    `json:"attributes,omitempty"`
	Links      []trace.Link   `json:"links,omitempty"`
	NewRoot    bool           `json:"newRoot,omitempty"`
	SpanKind   trace.SpanKind `json:"spanKind,omitempty"`
}

// Attribute represents an attribute registered using s.Span.SetAttributes().
type Attribute struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
	Type  string      `json:"type"`
}
