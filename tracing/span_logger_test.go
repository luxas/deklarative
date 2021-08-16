package tracing

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

// TODO: Make sure keysAndValues aren't modified when passed to Info/Error.

func Test_spanLogger_WithValues(t *testing.T) {
	log := (&spanLogger{log: logr.Discard()}).
		WithValues("foo", "bar")
	assert.Equal(t, log.(*spanLogger).keysAndValues, []interface{}{"foo", "bar"})

	newlog := log.WithValues("private", true)
	// newlog shouldn't modify the earlier assertion, verify it again
	assert.Equal(t, log.(*spanLogger).keysAndValues, []interface{}{"foo", "bar"})
	// newlog should now have more keys and values
	assert.Equal(t, newlog.(*spanLogger).keysAndValues, []interface{}{
		"foo", "bar", "private", true,
	})
}
