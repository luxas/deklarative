// Package filetest helps in verifying that content written to arbitrary io.Writers
// match some expected content in testdata files.
//
// See the test TestExample for an example of how to use this package.
package filetest

import (
	"bytes"
	"io"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
)

// New is a wrapper for goldie.New, but returns a *Tester goldie helper.
func New(t *testing.T, opts ...goldie.Option) *Tester { //nolint:thelper
	return &Tester{
		G:     goldie.New(t, opts...),
		T:     t,
		Files: make(map[string]*Target),
	}
}

// Tester is a high-level primitive for goldie.Goldie. It allows registering
// testdata files to verify io.Writer writes.
type Tester struct {
	G *goldie.Goldie
	T *testing.T
	// Files map a file name (conventionally under testdata/) to a
	// target buffer and set of filters.
	Files map[string]*Target
}

// Target is a write target for arbitrary content sources. Before verifying that
// the written content is right, the filters are applied in orders on the buffered
// content.
type Target struct {
	Buffer  *bytes.Buffer
	Filters []Filter
}

// Filter represents a byte filter; similar to an UNIX pipe.
type Filter func([]byte) []byte

// Add adds a new file target to the Files map. If name already exists in the map,
// it is overwritten.
func (g *Tester) Add(name string) *Target {
	b := &Target{
		Buffer: new(bytes.Buffer),
	}
	g.Files[name] = b
	return b
}

// Filter adds a new filter to the Target.
func (b *Target) Filter(filter Filter) *Target {
	b.Filters = append(b.Filters, filter)
	return b
}

// Writer returns the io.Writer which content sources can write to. The io.Writer
// is/writes to the buffer.
func (b *Target) Writer() io.Writer { return b.Buffer }

func (g *Tester) do(fn func(*testing.T, string, []byte)) {
	for name, a := range g.Files {
		content := a.Buffer.Bytes()
		for _, filter := range a.Filters {
			content = filter(content)
		}

		g.T.Run(name, func(t *testing.T) {
			fn(t, name, content)
		})
	}
}

// Assert verifies the all golden files are up-to-date.
// All file verifications are run in separate sub-tests.
//
// If the "-update" flag is passed to "go test", for example as
// "go test . -update", the files under testdata/ will be
// automatically updated.
func (g *Tester) Assert() { g.do(g.G.Assert) }

// Update updates all file content to match the written bytes to the
// returned io.Writer.
func (g *Tester) Update() {
	g.do(func(t *testing.T, name string, content []byte) { //nolint:thelper
		assert.Nil(t, g.G.Update(t, name, content))
	})
}
