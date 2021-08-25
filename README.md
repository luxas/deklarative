# deklarative

[![godev](https://img.shields.io/static/v1?label=godev&message=reference&color=00add8)](https://pkg.go.dev/github.com/luxas/deklarative)
[![build](https://github.com/luxas/deklarative/workflows/build/badge.svg)](https://github.com/luxas/deklarative/actions)

A comprehensive set of tools for parsing, manipulating, combining and streaming
declarative API specifications. In the world of k8s and cloud native with a lot of config languates, versions, there is a need for better tooling. `deklarative` aims to fix these in a high-level but extensible manner.

Libraries for framing, reading and writing byte streams and encoding, decoding, converting objects.

## Go modules

### `tracing`

A library for tracing your program using [OpenTelemetry] and logging using `go-logr` in a powerful, user-friendly, yet extensible way. Logging and tracing are interconnected in this approach, making it possible to look at instrumentation data
from multiple angles.

[`go-logr`]: https://github.com/go-logr
[OpenTelemetry]: https://opentelemetry.io/

### `content`

A set of core interfaces defining primitives for declarative content metadata,
recognizers, and content types.

### `json`

A JSON library delegating JSON encoding/decoding to [`json-iterator/go`], in the same way as [Kubernetes implements it].

[`json-iterator/go`]: https://github.com/json-iterator/go
[Kubernetes implements it]: https://github.com/kubernetes/apimachinery/blob/v0.22.0/pkg/runtime/serializer/json/json.go#L113-L184

### `yaml`

A YAML library delegating YAML 1.2 encoding/decoding to other libraries like [`yaml.v3`] and [`kyaml`]. YAML is first converted to JSON, and then the `json` library is always used for decoding/encoding.

[`yaml.v3`]: https://github.com/go-yaml/yaml/tree/v3
[`kyaml`]: https://github.com/kubernetes-sigs/kustomize/tree/master/kyaml/yaml

### `stream`

A library for working with byte streams. It is interconnecting recognizer and metadata interfaces from `content` with `io.Reader`s and `io.Writer`s. It supports
propagating contexts for telemetry propagation using the `tracing` library.

### `frame`

A library building on top of `stream` for extracting frames from a byte stream, e.g. JSON objects or YAML documents, respectively.

### `serialize`

A library using the `frame` library for reading frames, and then encoding/decoding these using e.g. the `json` and `yaml` lower-level libraries.

## Maintainers

- Lucas Käldström (@luxas)
