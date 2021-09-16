package json

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

type myStruct struct {
	DEF myCustom
	ABC myCustom
}

type myCustom struct{}

func (myCustom) MarshalJSON() ([]byte, error) {
	return []byte(`{"foo": "bar", "bar": 1}`), nil
}

func TestMarshal(t *testing.T) {
	obj := myStruct{}
	out, _ := jsoniter.ConfigCompatibleWithStandardLibrary.MarshalIndent(obj, "", "    ")
	t.Error(string(out))

	var buf bytes.Buffer
	_ = json.Indent(&buf, out, "", "  ")
	t.Error(buf.String())
}

func TestLossyFloatJSON(t *testing.T) {
	obj := map[string]interface{}{}
	foo := `{"a":1000000000000000001}`
	/*assert.Nil(t, json.Unmarshal([]byte(foo), &obj))
	out, err := json.Marshal(obj)
	assert.Nil(t, err)
	assert.Equal(t, foo, string(out))*/

	d := NewDecoder(strings.NewReader(foo))
	assert.Nil(t, d.Decode(&obj))
	t.Logf("%T", obj["a"])
	out, err := Marshal(obj)
	assert.Nil(t, err)
	assert.Equal(t, foo, string(out))

	obj = map[string]interface{}{}
	assert.Nil(t, Unmarshal([]byte(foo), &obj))
	out, err = Marshal(obj)
	assert.Nil(t, err)
	assert.Equal(t, foo, string(out))
	t.Error()

	/*assert.Nil(t, jsoniter.Unmarshal([]byte(foo), &obj))
	out, err = jsoniter.Marshal(obj)
	assert.Nil(t, err)
	assert.Equal(t, foo, string(out))*/
}
