package yaml

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	yamlv2 "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	utiljson "k8s.io/apimachinery/pkg/util/json"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	sigsyaml "sigs.k8s.io/yaml"
)

func TestYAMLMapKey(t *testing.T) {
	var n kyaml.Node
	assert.Nil(t, kyaml.Unmarshal([]byte(`{"foo": "bar"}: true`), &n))
	f := n.Content[0]
	f2 := f.Content[0]
	t.Error(f.Tag, f2.Tag, f2.Content[0].Value, f2.Content[1].Value, f.Content[1].Value)
	// TODO: What happens if you pass just a yaml.Node, does it error?
	// Does yaml.v3 need a pointer?
	b, _ := kyaml.Marshal(&n)
	t.Error(string(b))
}

type A struct {
	B `json:",inline"`
	C `json:",inline"`
}

type B struct {
	Foo string
}

func (b *B) UnmarshalJSON(data []byte) error {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	if obj == nil {
		return nil
	}
	name, _ := obj["msg"].(string)
	b.Foo = "foo-" + name
	return nil
}

type C struct {
	Name string
}

func (c *C) UnmarshalJSON(data []byte) error {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	if obj == nil {
		return nil
	}
	name, _ := obj["msg"].(string)
	c.Name = name
	return nil
}

func TestEmbeddedJSON(t *testing.T) {
	var a A
	assert.Nil(t, json.Unmarshal([]byte(`{"msg": "hello"}`), &a))
	t.Error(a)
}

func TestOctals(t *testing.T) {
	var obj map[string]interface{}
	assert.Nil(t, kyaml.Unmarshal([]byte(`foo: 077`), &obj))
	t.Errorf("%T, %v, %T", obj, obj, obj["foo"])
}

func TestMultiFrame(t *testing.T) {
	data := `---
foo: bar
---
bar: "Null"
---
{
	"foo": "bar",
	"foo": "baz"
}
---
`
	d := kyaml.NewDecoder(strings.NewReader(data))
	for {
		var obj interface{}
		err := d.Decode(&obj)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}

		t.Error(obj)
	}
}

func TestRoundtripLiterals(t *testing.T) {
	obj := map[string]interface{}{
		"bar1": "~",
		"bar2": nil,
		"bar3": "null",
		"bar4": "Null",
		"baz":  "hello",
		"foo1": true,
		"foo2": "True",
		"foo3": math.NaN(),
		"foo4": "0777",
	}
	b, err := kyaml.Marshal(obj)
	assert.Nil(t, err)
	t.Error(string(b))
}

func TestEncodeJSON(t *testing.T) {
	scheme := runtime.NewScheme()
	s := k8sjson.NewSerializerWithOptions(k8sjson.DefaultMetaFactory, scheme, scheme, k8sjson.SerializerOptions{
		Yaml: true,
	})

	var buf bytes.Buffer
	assert.Nil(t, s.Encode(&runtime.Unknown{Raw: []byte("foo: bar\n")}, &buf))
	assert.Nil(t, s.Encode(&runtime.Unknown{Raw: []byte("bar: true\n")}, &buf))

	t.Error(buf.String())
}

func TestEscapeHTML(t *testing.T) {
	foo := `foo: <script>xxx</script>`
	var obj interface{}
	assert.Nil(t, kyaml.Unmarshal([]byte(foo), &obj))
	t.Error(obj)

	out, err := kyaml.Marshal(obj)
	assert.Nil(t, err)
	t.Error(string(out))

	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	e.SetEscapeHTML(true)
	assert.Nil(t, e.Encode(obj))
	t.Error(buf.String())

	var out2 interface{}
	assert.Nil(t, json.NewDecoder(&buf).Decode(&out2))
	t.Error(out2)
}

var testData = []byte(`{"foo": "bar", "baz": 1234, "is": true, "arr": [], "obj": {}}`)

type testStruct struct {
	Foo string            `json:"foo" yaml:"foo"`
	Baz int64             `json:"baz" yaml:"baz"`
	Is  bool              `json:"is" yaml:"is"`
	Arr []string          `json:"arr" yaml:"arr"`
	Obj map[string]string `json:"obj" yaml:"obj"`
}

func ExampleYAMLMarshal() {
	ts := testStruct{Foo: "foo", Is: true, Arr: []string{"1", "2"}}

	out, err := Marshal(ts)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(out))
	// Output:
	// fo
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		obj := &testStruct{}
		_ = json.Unmarshal(testData, obj)
	}
}

func BenchmarkJSONDecoder(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		obj := &testStruct{}
		_ = json.NewDecoder(bytes.NewReader(testData)).Decode(obj)
	}
}

func BenchmarkJSONIterUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		obj := &testStruct{}
		_ = jsoniter.Unmarshal(testData, obj)
	}
}

func BenchmarkJSONIterDecoder(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		obj := &testStruct{}
		_ = jsoniter.NewDecoder(bytes.NewReader(testData)).Decode(obj)
	}
}

func BenchmarkJSONUtilUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		obj := &testStruct{}
		_ = utiljson.Unmarshal(testData, obj)
	}
}

func BenchmarkYAMLv2Unmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		obj := &testStruct{}
		_ = yamlv2.Unmarshal(testData, obj)
	}
}

func BenchmarkYAMLv2Decoder(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		obj := &testStruct{}
		_ = yamlv2.NewDecoder(bytes.NewReader(testData)).Decode(obj)
	}
}

func BenchmarkK8sYAMLUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		obj := &testStruct{}
		_ = sigsyaml.Unmarshal(testData, obj)
	}
}

func BenchmarkK8sYAMLUnmarshalStrict(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		obj := &testStruct{}
		_ = sigsyaml.UnmarshalStrict(testData, obj)
	}
}

func BenchmarkKyamlUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		obj := &testStruct{}
		_ = kyaml.Unmarshal(testData, obj)
	}
}

func BenchmarkKyamlDecoder(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		obj := &testStruct{}
		_ = kyaml.NewDecoder(bytes.NewReader(testData)).Decode(obj)
	}
}

func BenchmarkUnstructuredJSONUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var obj interface{}
		_ = json.Unmarshal(testData, &obj)
	}
}

func BenchmarkUnstructuredJSONIterUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var obj interface{}
		_ = json.Unmarshal(testData, &obj)
	}
}

func BenchmarkUnstructuredJSONUtilUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var obj interface{}
		_ = utiljson.Unmarshal(testData, &obj)
	}
}

func BenchmarkUnstructuredYAMLv2Unmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var obj interface{}
		_ = yamlv2.Unmarshal(testData, &obj)
	}
}

func BenchmarkUnstructuredK8sYAMLUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var obj interface{}
		_ = sigsyaml.Unmarshal(testData, &obj)
	}
}

func BenchmarkUnstructuredKyamlUnmarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var obj interface{}
		_ = kyaml.Unmarshal(testData, &obj)
	}
}

type nonStringKeyTest struct {
	// TIL that
	// "The map's key type must either be any string type, an integer, implement json.Unmarshaler, or implement encoding.TextUnmarshaler."
	// i.e. that {"6": "foo"} can be unmarshalTestled/marshalled to map[int(64)]string, but not floats or bools
	//
	// json-iter supports strings, ints, floats and bools as key, and should hence be pretty
	// good to deal with YAML. It's always string-encoded as per the JSON spec, but still.
	M map[float64]string
}

func Example_lossyFloat64() {
	var obj map[string]interface{}
	tests := map[string]int{
		"-2**53":     -1 << 53,
		"-2**53 - 1": -1<<53 - 1,
		"2**53":      1 << 53,
		"2**53 + 1":  1<<53 + 1,
	}

	for name, num := range tests {
		obj = map[string]interface{}{}
		data := `{"a":` + strconv.Itoa(num) + `}`
		if err := json.Unmarshal([]byte(data), &obj); err != nil {
			fmt.Println(err)
			return
		}

		out, err := json.Marshal(obj)
		if err != nil {
			fmt.Println(err)
			return
		}
		roundtripped := string(out)

		fmt.Printf("%s can be roundtripped: %t\n", name, roundtripped == data)
		fmt.Printf("Want: %s\n", data)
		fmt.Printf("Got: %s\n\n", roundtripped)
	}

	/*obj = map[string]interface{}{}
	data = `{"a":` + strconv.Itoa(int(math.Pow(2, 53))+1) + `}`
	if err := json.Unmarshal([]byte(data), &obj); err != nil {
		fmt.Println(err)
		return
	}

	out, err = json.Marshal(obj)
	if err != nil {
		fmt.Println(err)
		return
	}
	roundtripped = string(out)

	fmt.Printf("2**53 + 1 can be roundtripped: %t\n", roundtripped == data)
	fmt.Printf("Want: %s\n", data)
	fmt.Printf("Got: %s\n", roundtripped)*/

	// Output:
	// fo
}

func TestLossyFloatJSON(t *testing.T) {
	var obj interface{}
	foo := `{"a":` + strconv.Itoa(-int(math.Pow(2, 53))-1) + `}`
	assert.Nil(t, json.Unmarshal([]byte(foo), &obj))
	out, err := json.Marshal(obj)
	assert.Nil(t, err)
	assert.Equal(t, foo, string(out))

	assert.Nil(t, utiljson.Unmarshal([]byte(foo), &obj))
	out, err = utiljson.Marshal(obj)
	assert.Nil(t, err)
	assert.Equal(t, foo, string(out))

	assert.Nil(t, jsoniter.Unmarshal([]byte(foo), &obj))
	out, err = jsoniter.Marshal(obj)
	assert.Nil(t, err)
	assert.Equal(t, foo, string(out))
}

func TestLossyFloatYAML(t *testing.T) {
	var obj map[string]interface{}
	//             10000000000000001
	foo := `a: ` + strconv.Itoa(9007199255000000) + "\n"

	assert.Nil(t, yamlv2.Unmarshal([]byte(foo), &obj))
	out, err := yamlv2.Marshal(obj)
	assert.Nil(t, err)
	assert.Equal(t, foo, string(out))
	t.Logf("%T", obj["a"])

	assert.Nil(t, kyaml.Unmarshal([]byte(foo), &obj))
	out, err = kyaml.Marshal(obj)
	assert.Nil(t, err)
	assert.Equal(t, foo, string(out))
	t.Logf("%T", obj["a"])

	assert.Nil(t, sigsyaml.Unmarshal([]byte(foo), &obj))
	out, err = sigsyaml.Marshal(obj)
	assert.Nil(t, err)
	assert.Equal(t, foo, string(out))
	t.Logf("%T", obj["a"])
	t.Error()
}

type empty struct {
	S string `json:"s,omitempty" yaml:"s,omitempty"`
}

func (e *empty) IsZero() bool { return true }

//func (empty) MarshalJSON() ([]byte, error) { return []byte("null"), nil }
func Example_encode() {
	type foo struct {
		T     metav1.Time `json:"t,omitempty" yaml:"t,omitempty"`
		Str   string      `json:"str,omitempty" yaml:"str,omitempty"`
		Empty *empty      `json:"empty,omitempty" yaml:"empty,omitempty"`
	}
	out, err := kyaml.Marshal(foo{T: metav1.Now(), Str: "bl", Empty: &empty{" "}})
	fmt.Printf("%s, %v\n", out, err)

	/*o := metav1.ObjectMeta{CreationTimestamp: metav1.Now()}
	out, err = kyaml.Marshal(o)
	fmt.Printf("%s, %v\n", out, err)

	o = metav1.ObjectMeta{}
	out, err = kyaml.Marshal(o)
	fmt.Printf("%s, %v\n", out, err)*/

	// Output:
	// foo
}

/*
func TestNonStringKeysToJSON(t *testing.T) {
	data := `foo:
  8080:
    bla: true
  true: bla
  4.1: 4.3
`
	var n Node
	assert.Nil(t, kyaml.Unmarshal([]byte(data), &n))
	jsonObj, err := ToJSONGeneric(&n)
	assert.Nil(t, err)

	showChildren(t, &n)
	t.Errorf("%T %s", jsonObj, jsonObj)
	fooNode := jsonObj.(map[string]interface{})["foo"]
	t.Errorf("%T %s", fooNode, fooNode)

	out, err := jsoniter.Marshal(jsonObj)
	assert.Nil(t, err)
	t.Error(string(out))

	jsonObj, err = ConvertToJSONableObject(jsonObj, nil)
	assert.Nil(t, err)

	t.Errorf("%T %s", jsonObj, jsonObj)
	fooNode = jsonObj.(map[string]interface{})["foo"]
	t.Errorf("%T %s", fooNode, fooNode)

}*/

func showChildren(t *testing.T, n *Node) {
	if n.Value != "" {
		t.Error(n)
	}
	for _, ch := range n.Content {
		showChildren(t, ch)
	}
}

func TestFoo(t *testing.T) {
	var obj interface{}
	data := "foo: 04:30"
	assert.Nil(t, kyaml.Unmarshal([]byte(data), &obj))
	out, err := kyaml.Marshal(obj)
	assert.Nil(t, err)
	assert.Equal(t, data, string(out))
}

func TestNonStringKeys3(t *testing.T) {
	obj := &nonStringKeyTest{M: map[float64]string{5: "foo"}}
	out, err := utiljson.Marshal(obj)
	assert.Nil(t, err)
	e, ok := err.(*json.UnmarshalTypeError)
	if ok {
		t.Log(e.Type)
	}
	t.Logf("%s", out)
	t.Error()
}

func TestNonStringKeys2(t *testing.T) {
	obj := &nonStringKeyTest{}
	err := sigsyaml.Unmarshal([]byte(`{"M":{"5.1": "foo"}}`), obj)
	assert.Nil(t, err)
	e, ok := err.(*json.UnmarshalTypeError)
	if ok {
		t.Log(e.Type)
	}
	t.Logf("%v", obj)
	t.Error()
}

func TestNonStringKeys(t *testing.T) {
	foo := `
m:
  6: foo
`
	obj := &nonStringKeyTest{}
	assert.Nil(t, kyaml.Unmarshal([]byte(foo), obj))
	t.Logf("yaml.v3: %v", obj)

	rn, err := kyaml.Parse(foo)
	assert.Nil(t, err)
	jsonMap, err := rn.Map()
	assert.Nil(t, err)
	t.Logf("yaml.v3: %v", jsonMap)
	t.Logf("%T %T %s", jsonMap, jsonMap["m"], (jsonMap["m"].(map[interface{}]interface{}))[6])
	newJson, err := utiljson.Marshal(jsonMap)
	assert.Nil(t, err)
	t.Logf("%s", newJson)
	t.Error()
}

type hasJSONTag struct {
	MyStructField string `json:"myStructField" yaml:"myStructField"`
}

type noJSONTag struct {
	MyStructField string
}

func TestDuplicateKeys(t *testing.T) {
	foo := `
mystructfield: baz
myStructField: bar
MyStructField: foo
`
	f := &hasJSONTag{}
	assert.Nil(t, kyaml.Unmarshal([]byte(foo), f))
	t.Logf("yaml.v3: %v", f)
	f = &hasJSONTag{}
	assert.Nil(t, yamlv2.Unmarshal([]byte(foo), f))
	t.Logf("yaml.v2: %v", f)
	f = &hasJSONTag{}
	assert.Nil(t, sigsyaml.Unmarshal([]byte(foo), f))
	t.Logf("sigs.yaml: %v", f)

	f2 := &noJSONTag{}
	assert.Nil(t, kyaml.Unmarshal([]byte(foo), f2))
	t.Logf("yaml.v3: %v", f2)
	f2 = &noJSONTag{}
	assert.Nil(t, yamlv2.Unmarshal([]byte(foo), f2))
	t.Logf("yaml.v2: %v", f2)
	f2 = &noJSONTag{}
	assert.Nil(t, sigsyaml.Unmarshal([]byte(foo), f2))
	t.Logf("sigs.yaml: %v", f2)

	t.Error("foo")
}

func TestEncoder(t *testing.T) {
	/*g := filetest.New(t)
	defer g.Assert()
	defer g.Update()

	objs := []interface{}{
		MarshalTest{A: "foo", B: 123, C: 1.2},
		MarshalTest{A: "foo", B: 123, C: 1.2},
	}
	e := kyaml.NewEncoder(g.Add("TestEncoder").Writer())
	for _, obj := range objs {
		assert.Nil(t, e.Encode(obj))
	}*/

	//str := g.Files["TestEncoder"].Buffer.String()
	//fmt.Println("data", str)
	str := `
a: foo
b: 123
c: 1.2
---
a: foo
f:
  - bla

  - blabla

b:    1234

c: 1.2

`
	/*
		p := kyaml.Parser{Value: str}
			rn, err := p.Filter(nil)
			assert.Nil(t, err)
			fmt.Println(rn.String())

			rn, err = p.Filter(nil)
			assert.Nil(t, err)
			fmt.Println(rn.String())
	*/

	d := kyaml.NewDecoder(strings.NewReader(str))
	n := kyaml.Node{}
	assert.Nil(t, d.Decode(&n))
	var buf bytes.Buffer
	e := kyaml.NewEncoder(&buf)
	assert.Nil(t, e.Encode(&n))
	assert.Nil(t, e.Close())
	fmt.Println(buf.String())

	n = kyaml.Node{}
	assert.Nil(t, d.Decode(&n))
	buf.Reset()
	e = kyaml.NewEncoder(&buf)
	assert.Nil(t, e.Encode(&n))
	assert.Nil(t, e.Close())
	out, err := json.MarshalIndent(n, "", "  ")
	fmt.Println(string(out), err)
	fmt.Println(buf.String())

	//
	//fmt.Println("foo", n)

	//for range objs {

	/*newObj := MarshalTest{}
	assert.Nil(t, d.Decode(&newObj))
	assert.Equal(t, obj, newObj)*/
	//var msg yaml.Unmarshaler
	//}
	t.Error("out")
}

type MarshalTest struct {
	A string
	B int64
	// CHANGE: Fixed this to be float64, instead of float32
	C float64
}

func TestMarshal(t *testing.T) {
	f64String := strconv.FormatFloat(math.MaxFloat64, 'g', -1, 64)
	s := MarshalTest{"a", math.MaxInt32, math.MaxFloat64}
	e := fmt.Sprintf("A: a\nB: %d\nC: %s\n", math.MaxInt32, f64String)

	// TODO: Test on a struct that has json tags as well.
	y, err := Marshal(s)
	assert.Nil(t, err)
	assert.Equal(t, e, string(y))

	s2 := UnmarshalSlice{[]NestedSlice{NestedSlice{"abc", strPtr("def")}, NestedSlice{"123", strPtr("456")}}}
	e2 := `a:
- b: abc
  c: def
- b: "123"
  c: "456"
`
	y, err = Marshal(s2)
	assert.Nil(t, err)
	assert.Equal(t, e2, string(y))
}

type UnmarshalString struct {
	A    string `json:"a"`
	True int64  `json:"true"`
}

type UnmarshalStringMap struct {
	A map[string]string `json:"a"`
}

type UnmarshalNestedString struct {
	A NestedString `json:"a"`
}

type NestedString struct {
	A string `json:"a"`
}

type UnmarshalSlice struct {
	A []NestedSlice `json:"a"`
}

type NestedSlice struct {
	B string  `json:"b"`
	C *string `json:"c"`
}

func TestUnmarshal(t *testing.T) {
	y := []byte(`a: "1"`) // TODO: Verify that numbers need to be quoted in k8s
	s1 := UnmarshalString{}
	e1 := UnmarshalString{A: "1"}
	unmarshalTest(t, y, &s1, &e1)

	y = []byte(`a: "true"`)
	s1 = UnmarshalString{}
	e1 = UnmarshalString{A: "true"}
	unmarshalTest(t, y, &s1, &e1)

	y = []byte(`true: 1`)
	s1 = UnmarshalString{}
	e1 = UnmarshalString{True: 1}
	unmarshalTest(t, y, &s1, &e1)

	y = []byte(`
a:
  a: "1"`)
	s2 := UnmarshalNestedString{}
	e2 := UnmarshalNestedString{NestedString{"1"}}
	unmarshalTest(t, y, &s2, &e2)

	y = []byte(`
a:
- b: abc
  c: def
- b: "123"
  c: "456"
`)
	s3 := UnmarshalSlice{}
	e3 := UnmarshalSlice{[]NestedSlice{NestedSlice{"abc", strPtr("def")}, NestedSlice{"123", strPtr("456")}}}
	unmarshalTest(t, y, &s3, &e3)

	y = []byte(`
a:
  b: "1"`)
	s4 := UnmarshalStringMap{}
	e4 := UnmarshalStringMap{map[string]string{"b": "1"}}
	unmarshalTest(t, y, &s4, &e4)

	y = []byte(`
a:
  name: TestA
b:
  name: TestB
`)
	type NamedThing struct {
		Name string `json:"name"`
	}
	s5 := map[string]*NamedThing{}
	e5 := map[string]*NamedThing{
		"a": &NamedThing{Name: "TestA"},
		"b": &NamedThing{Name: "TestB"},
	}
	unmarshalTest(t, y, &s5, &e5)
}

func unmarshalTest(t *testing.T, y []byte, s, e interface{}) {
	t.Helper()
	err := Unmarshal(y, s)
	if err != nil {
		t.Errorf("error unmarshalTesting YAML: %v", err)
	}

	if !reflect.DeepEqual(s, e) {
		t.Errorf("unmarshalTest YAML was unsuccessful, expected: %+#v, got: %+#v",
			e, s)
	}
}

func TestUnmarshalStrict(t *testing.T) {
	y := []byte("a: 1")
	s1 := UnmarshalString{}
	e1 := UnmarshalString{A: "1"}
	unmarshalTestStrict(t, y, &s1, &e1)

	y = []byte("a: true")
	s1 = UnmarshalString{}
	e1 = UnmarshalString{A: "true"}
	unmarshalTestStrict(t, y, &s1, &e1)

	y = []byte("true: 1")
	s1 = UnmarshalString{}
	e1 = UnmarshalString{True: 1}
	unmarshalTestStrict(t, y, &s1, &e1)

	y = []byte("a:\n  a: 1")
	s2 := UnmarshalNestedString{}
	e2 := UnmarshalNestedString{NestedString{"1"}}
	unmarshalTestStrict(t, y, &s2, &e2)

	y = []byte("a:\n  - b: abc\n    c: def\n  - b: 123\n    c: 456\n")
	s3 := UnmarshalSlice{}
	e3 := UnmarshalSlice{[]NestedSlice{NestedSlice{"abc", strPtr("def")}, NestedSlice{"123", strPtr("456")}}}
	unmarshalTestStrict(t, y, &s3, &e3)

	y = []byte("a:\n  b: 1")
	s4 := UnmarshalStringMap{}
	e4 := UnmarshalStringMap{map[string]string{"b": "1"}}
	unmarshalTestStrict(t, y, &s4, &e4)

	y = []byte(`
a:
  name: TestA
b:
  name: TestB
`)
	type NamedThing struct {
		Name string `json:"name"`
	}
	s5 := map[string]*NamedThing{}
	e5 := map[string]*NamedThing{
		"a": &NamedThing{Name: "TestA"},
		"b": &NamedThing{Name: "TestB"},
	}
	unmarshalTest(t, y, &s5, &e5)

	// When using not-so-strict unmarshalTest, we should
	// be picking up the ID-1 as the value in the "id" field
	y = []byte(`
a:
  name: TestA
  id: ID-A
  id: ID-1
`)
	type NamedThing2 struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	}
	s6 := map[string]*NamedThing2{}
	e6 := map[string]*NamedThing2{
		"a": {Name: "TestA", ID: "ID-1"},
	}
	unmarshalTest(t, y, &s6, &e6)
}

func TestUnmarshalStrictFails(t *testing.T) {
	y := []byte("a: true\na: false")
	s1 := UnmarshalString{}
	unmarshalTestStrictFail(t, y, &s1)

	y = []byte("a:\n  - b: abc\n    c: 32\n      b: 123")
	s2 := UnmarshalSlice{}
	unmarshalTestStrictFail(t, y, &s2)

	y = []byte("a:\n  b: 1\n    c: 3")
	s3 := UnmarshalStringMap{}
	unmarshalTestStrictFail(t, y, &s3)

	type NamedThing struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	}
	// When using strict unmarshalTest, we should see
	// the unmarshalTest fail if there are multiple keys
	y = []byte(`
a:
  name: TestA
  id: ID-A
  id: ID-1
`)
	s4 := NamedThing{}
	unmarshalTestStrictFail(t, y, &s4)

	// Strict unmarshalTest should fail for unknown fields
	y = []byte(`
name: TestB
id: ID-B
unknown: Some-Value
`)
	s5 := NamedThing{}
	unmarshalTestStrictFail(t, y, &s5)
}

func unmarshalTestStrict(t *testing.T, y []byte, s, e interface{}) {
	err := UnmarshalStrict(y, s)
	if err != nil {
		t.Errorf("error unmarshalTesting YAML: %v", err)
	}

	if !reflect.DeepEqual(s, e) {
		t.Errorf("unmarshalTest YAML was unsuccessful, expected: %+#v, got: %+#v",
			e, s)
	}
}

func unmarshalTestStrictFail(t *testing.T, y []byte, s interface{}) {
	err := UnmarshalStrict(y, s)
	if err == nil {
		t.Errorf("error unmarshalTesting YAML: %v", err)
	}
}

type Case struct {
	input  string
	output string
	// By default we test that reversing the output == input. But if there is a
	// difference in the reversed output, you can optionally specify it here.
	reverse *string
}

type RunType int

const (
	RunTypeJSONToYAML RunType = iota
	RunTypeYAMLToJSON
)

func TestJSONToYAML(t *testing.T) {
	cases := []Case{
		{
			`{"t":"a"}`,
			"t: a\n",
			nil,
		}, {
			`{"t":null}`,
			"t: null\n",
			nil,
		},
	}

	runCases(t, RunTypeJSONToYAML, cases)
}

func TestYAMLToJSON(t *testing.T) {
	cases := []Case{
		{
			"t: a\n",
			`{"t":"a"}`,
			nil,
		}, {
			"t: \n",
			`{"t":null}`,
			strPtr("t: null\n"),
		}, {
			"t: null\n",
			`{"t":null}`,
			nil,
		}, {
			"1: a\n",
			`{"1":"a"}`,
			strPtr("\"1\": a\n"),
		}, {
			"1000000000000000000000000000000000000: a\n",
			`{"1e+36":"a"}`,
			strPtr("\"1e+36\": a\n"),
		}, {
			"1e+36: a\n",
			`{"1e+36":"a"}`,
			strPtr("\"1e+36\": a\n"),
		}, {
			"\"1e+36\": a\n",
			`{"1e+36":"a"}`,
			nil,
		}, {
			"\"1.2\": a\n",
			`{"1.2":"a"}`,
			nil,
		}, {
			"- t: a\n",
			`[{"t":"a"}]`,
			nil,
		}, {
			"- t: a\n" +
				"- t:\n" +
				"    b: 1\n" +
				"    c: 2\n",
			`[{"t":"a"},{"t":{"b":1,"c":2}}]`,
			nil,
		}, {
			`[{t: a}, {t: {b: 1, c: 2}}]`,
			`[{"t":"a"},{"t":{"b":1,"c":2}}]`,
			strPtr("- t: a\n" +
				"- t:\n" +
				"    b: 1\n" +
				"    c: 2\n"),
		}, {
			"- t: \n",
			`[{"t":null}]`,
			strPtr("- t: null\n"),
		}, {
			"- t: null\n",
			`[{"t":null}]`,
			nil,
		},
	}

	// Cases that should produce errors.
	_ = []Case{
		{
			"~: a",
			`{"null":"a"}`,
			nil,
		}, {
			"a: !!binary gIGC\n",
			"{\"a\":\"\x80\x81\x82\"}",
			nil,
		},
	}

	runCases(t, RunTypeYAMLToJSON, cases)
}

func runCases(t *testing.T, runType RunType, cases []Case) {
	var f func([]byte) ([]byte, error)
	var invF func([]byte) ([]byte, error)
	var msg string
	var invMsg string
	if runType == RunTypeJSONToYAML {
		f = JSONToYAML
		invF = YAMLToJSON
		msg = "JSON to YAML"
		invMsg = "YAML back to JSON"
	} else {
		f = YAMLToJSON
		invF = JSONToYAML
		msg = "YAML to JSON"
		invMsg = "JSON back to YAML"
	}

	for _, c := range cases {
		// Convert the string.
		t.Logf("converting %s\n", c.input)
		output, err := f([]byte(c.input))
		if err != nil {
			t.Errorf("Failed to convert %s, input: `%s`, err: %v", msg, c.input, err)
		}

		// Check it against the expected output.
		if string(output) != c.output {
			t.Errorf("Failed to convert %s, input: `%s`, expected `%s`, got `%s`",
				msg, c.input, c.output, string(output))
		}

		// Set the string that we will compare the reversed output to.
		reverse := c.input
		// If a special reverse string was specified, use that instead.
		if c.reverse != nil {
			reverse = *c.reverse
		}

		// Reverse the output.
		input, err := invF(output)
		if err != nil {
			t.Errorf("Failed to convert %s, input: `%s`, err: %v", invMsg, string(output), err)
		}

		// Check the reverse is equal to the input (or to *c.reverse).
		if string(input) != reverse {
			t.Errorf("Failed to convert %s, input: `%s`, expected `%s`, got `%s`",
				invMsg, string(output), reverse, string(input))
		}
	}

}

// To be able to easily fill in the *Case.reverse string above.
func strPtr(s string) *string {
	return &s
}

func TestYAMLToJSONStrict(t *testing.T) {
	const data = `
foo: bar
foo: baz
`
	if _, err := YAMLToJSON([]byte(data)); err != nil {
		t.Error("expected YAMLtoJSON to pass on duplicate field names")
	}
	// TODO
	if _, err := YAMLToJSON([]byte(data)); err == nil {
		t.Error("expected YAMLtoJSONStrict to fail on duplicate field names")
	}
}

/*
func TestJSONObjectToYAMLObject(t *testing.T) {
	const bigUint64 = ((uint64(1) << 63) + 500) / 1000 * 1000
	intOrInt64 := func(i64 int64) interface{} {
		if i := int(i64); i64 == int64(i) {
			return i
		}
		return i64
	}

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected yaml.MapSlice
	}{
		{name: "nil", expected: yaml.MapSlice(nil)},
		{name: "empty", input: map[string]interface{}{}, expected: yaml.MapSlice(nil)},
		{
			name: "values",
			input: map[string]interface{}{
				"nil slice":          []interface{}(nil),
				"nil map":            map[string]interface{}(nil),
				"empty slice":        []interface{}{},
				"empty map":          map[string]interface{}{},
				"bool":               true,
				"float64":            float64(42.1),
				"fractionless":       float64(42),
				"int":                int(42),
				"int64":              int64(42),
				"int64 big":          float64(math.Pow(2, 62)),
				"negative int64 big": -float64(math.Pow(2, 62)),
				"map":                map[string]interface{}{"foo": "bar"},
				"slice":              []interface{}{"foo", "bar"},
				"string":             string("foo"),
				"uint64 big":         bigUint64,
			},
			expected: yaml.MapSlice{
				{Key: "nil slice"},
				{Key: "nil map"},
				{Key: "empty slice", Value: []interface{}{}},
				{Key: "empty map", Value: yaml.MapSlice(nil)},
				{Key: "bool", Value: true},
				{Key: "float64", Value: float64(42.1)},
				{Key: "fractionless", Value: int(42)},
				{Key: "int", Value: int(42)},
				{Key: "int64", Value: int(42)},
				{Key: "int64 big", Value: intOrInt64(int64(1) << 62)},
				{Key: "negative int64 big", Value: intOrInt64(-(1 << 62))},
				{Key: "map", Value: yaml.MapSlice{{Key: "foo", Value: "bar"}}},
				{Key: "slice", Value: []interface{}{"foo", "bar"}},
				{Key: "string", Value: string("foo")},
				{Key: "uint64 big", Value: bigUint64},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JSONObjectToYAMLObject(tt.input)
			sortMapSlicesInPlace(tt.expected)
			sortMapSlicesInPlace(got)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("jsonToYAML() = %v, want %v", spew.Sdump(got), spew.Sdump(tt.expected))
			}

			jsonBytes, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("unexpected json.Marshal error: %v", err)
			}
			var gotByRoundtrip yaml.MapSlice
			if err := yaml.Unmarshal(jsonBytes, &gotByRoundtrip); err != nil {
				t.Fatalf("unexpected yaml.Unmarshal error: %v", err)
			}

			// yaml.Unmarshal loses precision, it's rounding to the 4th last digit.
			// Replicate this here in the test, but don't change the type.
			for i := range got {
				switch got[i].Key {
				case "int64 big", "uint64 big", "negative int64 big":
					switch v := got[i].Value.(type) {
					case int64:
						d := int64(500)
						if v < 0 {
							d = -500
						}
						got[i].Value = int64((v+d)/1000) * 1000
					case uint64:
						got[i].Value = uint64((v+500)/1000) * 1000
					case int:
						d := int(500)
						if v < 0 {
							d = -500
						}
						got[i].Value = int((v+d)/1000) * 1000
					default:
						t.Fatalf("unexpected type for key %s: %v:%T", got[i].Key, v, v)
					}
				}
			}

			if !reflect.DeepEqual(got, gotByRoundtrip) {
				t.Errorf("yaml.Unmarshal(json.Marshal(tt.input)) = %v, want %v\njson: %s", spew.Sdump(gotByRoundtrip), spew.Sdump(got), string(jsonBytes))
			}
		})
	}
}

func sortMapSlicesInPlace(x interface{}) {
	switch x := x.(type) {
	case []interface{}:
		for i := range x {
			sortMapSlicesInPlace(x[i])
		}
	case yaml.MapSlice:
		sort.Slice(x, func(a, b int) bool {
			return x[a].Key.(string) < x[b].Key.(string)
		})
	}
}
*/
