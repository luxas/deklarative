package content

type Frame interface {
	// TODO: Is there a need for a deep-copy function?

	ContentType() ContentType
	Content() []byte

	DecodedGeneric() interface{}
	IsEmpty() bool
}

func NewFrame(ct ContentType, content []byte, obj interface{}, isEmpty bool) Frame {
	return &frame{ct, content, obj, isEmpty}
}

type frame struct {
	ct      ContentType
	content []byte
	obj     interface{}
	isEmpty bool
}

func (f *frame) ContentType() ContentType    { return f.ct }
func (f *frame) Content() []byte             { return f.content }
func (f *frame) DecodedGeneric() interface{} { return f.obj }
func (f *frame) IsEmpty() bool               { return f.isEmpty }
