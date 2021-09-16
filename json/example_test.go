package json

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/luxas/deklarative/content"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Example_random() {
	var obj map[string]interface{}
	if err := Unmarshal([]byte(`{"foo": 0.2}`), &obj); err != nil {
		fmt.Println(err)
		return
	}
	b, err := Marshal(obj)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(b))

	// Output:
	// {"foo":0.2}
}

func ExampleDecoder_DecodeFrame() {
	data := ` {  "def"  : "bar", "abc": 1  } { "6" : true   }["foo", "bar"]"str"123falsetrue1.2{}[]`

	d := NewDecoder(strings.NewReader(data))
	for {
		f, err := d.DecodeFrame()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(err)
		}

		fmt.Printf("%s: \"%s\"\n", f.ContentType(), f.Content())
		obj := f.DecodedGeneric()
		fmt.Printf("Type: %T, %v\n", obj, obj)
	}

	// Output:
	// application/json: "{  "def"  : "bar", "abc": 1  }"
	// Type: map[string]interface {}, map[abc:1 def:bar]
	// application/json: "{ "6" : true   }"
	// Type: map[string]interface {}, map[6:true]
	// application/json: "["foo", "bar"]"
	// Type: []interface {}, [foo bar]
	// application/json: ""str""
	// Type: string, str
	// application/json: "123"
	// Type: int64, 123
	// application/json: "false"
	// Type: bool, false
	// application/json: "true"
	// Type: bool, true
	// application/json: "1.2"
	// Type: float64, 1.2
	// application/json: "{}"
	// Type: map[string]interface {}, map[]
	// application/json: "[]"
	// Type: []interface {}, []
}

func ExampleDecoder_UseNumber() {
	data := ` {  "def"  : 1, "abc": 1.2  }{"6":1}123"s"3.14{}`
	d := NewDecoder(strings.NewReader(data))
	d.UseNumber()
	for {
		var obj interface{}
		err := d.Decode(&obj)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(err)
		}

		numberInfo := func(n Number, indent string) {
			i64, err := n.Int64()
			fmt.Printf("%sjson.Number.Int64() = %d, %v\n", indent, i64, err)
			f64, err := n.Float64()
			fmt.Printf("%sjson.Number.Float64() = %f, %v\n", indent, f64, err)
		}

		fmt.Printf("Type: %T, %v\n", obj, obj)
		switch o := obj.(type) {
		case map[string]interface{}:
			for k, v := range o {
				if n, ok := v.(Number); ok {
					fmt.Printf("  Field %s:\n", k)
					numberInfo(n, "    ")
				}
			}
		case Number:
			numberInfo(o, "  ")
		}
	}

	// Output:
	// Type: map[string]interface {}, map[abc:1.2 def:1]
	//   Field def:
	//     json.Number.Int64() = 1, <nil>
	//     json.Number.Float64() = 1.000000, <nil>
	//   Field abc:
	//     json.Number.Int64() = 0, strconv.ParseInt: parsing "1.2": invalid syntax
	//     json.Number.Float64() = 1.200000, <nil>
	// Type: map[string]interface {}, map[6:1]
	//   Field 6:
	//     json.Number.Int64() = 1, <nil>
	//     json.Number.Float64() = 1.000000, <nil>
	// Type: json.Number, 123
	//   json.Number.Int64() = 123, <nil>
	//   json.Number.Float64() = 123.000000, <nil>
	// Type: string, s
	// Type: json.Number, 3.14
	//   json.Number.Int64() = 0, strconv.ParseInt: parsing "3.14": invalid syntax
	//   json.Number.Float64() = 3.140000, <nil>
	// Type: map[string]interface {}, map[]
}

func ExampleUnknownNumberStrategyAlwaysFloat64() {
	data := ` {  "def"  : 1, "abc": 1.2  }{"6":1}123"s"3.14{}`
	opt := &DecoderOptions{
		UnknownNumberStrategy: content.UnknownNumberStrategyAlwaysFloat64,
	}
	d1 := NewDecoder(strings.NewReader(data), opt)
	for {
		var obj interface{}
		err := d1.Decode(&obj)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(err)
		}

		fmt.Printf("Type: %T, %v\n", obj, obj)
		if o, ok := obj.(map[string]interface{}); ok {
			for k, v := range o {
				fmt.Printf("  Field %s: %T, %v\n", k, v, v)
			}
		}
	}

	// Output:
	// Type: map[string]interface {}, map[abc:1.2 def:1]
	//   Field def: float64, 1
	//   Field abc: float64, 1.2
	// Type: map[string]interface {}, map[6:1]
	//   Field 6: float64, 1
	// Type: float64, 123
	// Type: string, s
	// Type: float64, 3.14
	// Type: map[string]interface {}, map[]
}

type Target struct {
	FieldA string `json:"fieldA"`
}

func Example_caseSensitiveness() {
	t := &Target{}
	err := Unmarshal([]byte(`{"fieldA": "value"}`), t)
	fmt.Printf("Allowing unknown fields: 'fieldA' = %q, err = %v\n", t.FieldA, err)

	t = &Target{}
	err = Unmarshal([]byte(`{"fielda": "value"}`), t)
	fmt.Printf("Allowing unknown fields: 'fielda' = %q, err = %v\n", t.FieldA, err)

	t = &Target{}
	err = UnmarshalStrict([]byte(`{"fielda": "value"}`), t)
	fmt.Printf("Disallowing unknown fields: 'fielda' = %q, err = %v\n", t.FieldA, err)

	// Output:
	// Allowing unknown fields: 'fieldA' = "value", err = <nil>
	// Allowing unknown fields: 'fielda' = "", err = <nil>
	// Disallowing unknown fields: 'fielda' = "", err = json.Target.ReadObject: found unknown field: fielda, error found in #9 byte of ...|{"fielda": "value"}|..., bigger context ...|{"fielda": "value"}|...
}

func Example_duplicateFields() {
	t := &Target{}
	err := Unmarshal([]byte(`{"fieldA": "value", "fielda": "val2"}`), t)
	fmt.Printf("Case-sensitiveness means no duplicates: 'fieldA' = %q, err = %v\n", t.FieldA, err)

	t = &Target{}
	err = UnmarshalStrict([]byte(`{"fieldA": "value", "fieldA": "val2"}`), t)
	fmt.Printf("Got duplicates in struct field: 'fieldA' = %q, err = %v\n", t.FieldA, err)

	t = &Target{}
	err = Unmarshal([]byte(`{"fielda": "value", "fielda": "val2"}`), t)
	fmt.Printf("Got duplicates not in struct field: 'fieldA' = %q, err = %v\n", t.FieldA, err)

	// Output:
	// Allowing unknown fields: 'fieldA' = "value", err = <nil>
	// Allowing unknown fields: 'fielda' = "", err = <nil>
	// Disallowing unknown fields: 'fielda' = "", err = json.Target.ReadObject: found unknown field: fielda, error found in #9 byte of ...|{"fielda": "value"}|..., bigger context ...|{"fielda": "value"}|...
}

type empty struct {
	S string `json:"s,omitempty"`
}

func (e *empty) IsZero() bool { return e.S == "" }
func Example_encode() {
	type foo struct {
		T     time.Time `json:"t,omitempty"`
		Str   string    `json:"str,omitempty"`
		Empty empty     `json:"empty,omitempty"`
	}
	out, err := Marshal(foo{Empty: empty{""}})
	fmt.Printf("%s, %v\n", out, err)

	o := metav1.ObjectMeta{CreationTimestamp: metav1.Now()}
	out, err = Marshal(o)
	fmt.Printf("%s, %v\n", out, err)

	o = metav1.ObjectMeta{}
	out, err = Marshal(o)
	fmt.Printf("%s, %v\n", out, err)

	// Output:
	// foo
}

func ExampleMarshalIndent() {
	type T struct {
		String string     `json:"string"`
		Raw    RawMessage `json:"raw"`
	}

	t := &T{"Foo", []byte(`{"hello": true}`)}
	out, err := MarshalIndent(t, "  ", "  ")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("MarshalIndent() =")
	fmt.Println(string(out))

	// Output:
	// MarshalIndent() =
	//   {
	//     "string": "Foo",
	//     "raw": {
	//       "hello": true
	//     }
	//   }
}

/*
func ExampleDecoder_DecodeFrame_float64Only() {
	data := ` {  "def"  : "bar", "abc": 1  } { "6" : true   }["foo", "bar"]"str"123falsetrue1.2{}[]`

	d := NewDecoder(strings.NewReader(data), UnknownNumberStrategyAlwaysFloat64())
	for {
		f, err := d.DecodeFrame()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(err)
		}

		fmt.Printf("%s: \"%s\"\n", f.ContentType(), f.Content())
		obj := f.DecodedGeneric()
		fmt.Printf("Type: %T, %v\n", obj, obj)
	}

	// Output:
	// application/json: "{  "def"  : "bar", "abc": 1  }"
	// Type: map[string]interface {}, map[abc:1 def:bar]
	// application/json: "{ "6" : true   }"
	// Type: map[string]interface {}, map[6:true]
	// application/json: "["foo", "bar"]"
	// Type: []interface {}, [foo bar]
	// application/json: ""str""
	// Type: string, str
	// application/json: "123"
	// Type: float64, 123
	// application/json: "false"
	// Type: bool, false
	// application/json: "true"
	// Type: bool, true
	// application/json: "1.2"
	// Type: float64, 1.2
	// application/json: "{}"
	// Type: map[string]interface {}, map[]
	// application/json: "[]"
	// Type: []interface {}, []
}

func ExampleDecoder_DecodeFrame_jsonNumber() {
	data := ` {  "def"  : "bar", "abc": 1  } { "6" : true   }["foo", "bar"]"str"123falsetrue1.2{}[]`

	d := NewDecoder(strings.NewReader(data))
	d.UseNumber()
	for {
		f, err := d.DecodeFrame()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(err)
		}

		fmt.Printf("%s: \"%s\"\n", f.ContentType(), f.Content())
		obj := f.DecodedGeneric()
		fmt.Printf("Type: %T, %v\n", obj, obj)
	}

	// Output:
	// application/json: "{  "def"  : "bar", "abc": 1  }"
	// Type: map[string]interface {}, map[abc:1 def:bar]
	// application/json: "{ "6" : true   }"
	// Type: map[string]interface {}, map[6:true]
	// application/json: "["foo", "bar"]"
	// Type: []interface {}, [foo bar]
	// application/json: ""str""
	// Type: string, str
	// application/json: "123"
	// Type: json.Number, 123
	// application/json: "false"
	// Type: bool, false
	// application/json: "true"
	// Type: bool, true
	// application/json: "1.2"
	// Type: json.Number, 1.2
	// application/json: "{}"
	// Type: map[string]interface {}, map[]
	// application/json: "[]"
	// Type: []interface {}, []
}*/
