package filetest

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExample(t *testing.T) {
	// The ideomatic way to use this is to define g somewhere in the beginning,
	// and run g.Assert() as a deferred function, so that all writes registered
	// in the test have time to be applied.
	g := New(t)
	defer g.Assert()

	// Update runs before Assert; hence this will always succeed as a sample test
	// In real life, g.Update() wouldn't probably be called unconditionally, only
	// when the user passes the "-update" flag.
	defer g.Update()

	// Get a writer that will be comparing what was written to it with the
	// contents of foo.txt, after the filter has been applied.
	// Multiple filters can be applied.
	w := g.Add("foo.txt").
		Filter(replaceSpacing).
		Filter(replaceSpacing).
		Writer()

	// Use the writer; at the end of the test it'll be verified that writeSomethingTo
	// wrote contents that was expected.
	err := writeSomethingTo(w)
	assert.Nil(t, err)
}

// replaceSpacing is a sample filter function.
func replaceSpacing(in []byte) []byte {
	return bytes.TrimSpace(bytes.ReplaceAll(in, []byte("  "), []byte(" ")))
}

// writeSomethingTo is a sample function that produces byte output, writing to an
// io.Writer.
func writeSomethingTo(w io.Writer) error {
	_, err := w.Write([]byte(`


abc   ss   a    

`))
	return err
}
