package structerr

// StructError is an interface for errors that are structs, and can be compared for
// errors.Is equality. Equality is determined by type equality, that is, if the pointer
// receiver is *MyError and target can be successfully casted using target.(*MyError),
// then target and the pointer receiver error are equal, otherwise not.
//
// This is needed because errors.Is does not support equality like this for structs
// by default.
type StructError interface {
	error
	Is(target error) bool
}

// UnwrapError extends the StructError with a method to retrieve the underlying
// error cause, as per the interface presented in errors.Unwrap.
type UnwrapError interface {
	StructError
	Unwrap() error
}
