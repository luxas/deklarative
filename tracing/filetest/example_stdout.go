package filetest

import (
	"bytes"
	"io"
	"os"
	"unicode"
)

// ExampleStdout is a dumb wrapper around os.Stdout that removes trailing
// spaces from each line in each io.Writer.Write call. This is useful for
// examples where os.Stdout output is matched against some output text in
// comments. However, gofmt trims trailing spaces in the go source code,
// which means it's impossible to match trailing spaces without trimming
// in a wrapper like this.
const ExampleStdout = exampleWriter(0)

var _ io.Writer = ExampleStdout

//nolint:gochecknoglobals
var newlineSep = []byte{'\n'}

type exampleWriter int

func (exampleWriter) Write(p []byte) (int, error) {
	lines := bytes.Split(p, newlineSep)
	for i := range lines {
		lines[i] = bytes.TrimRightFunc(lines[i], unicode.IsSpace)
	}
	return os.Stdout.Write(bytes.Join(lines, newlineSep))
}
