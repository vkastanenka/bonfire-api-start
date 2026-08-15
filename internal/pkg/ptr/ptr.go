package ptr

// To returns a pointer to a copy of value v.
func To[T any](v T) *T {
	return &v
}

// From returns the value pointed to by v, or the type's zero value if v is nil.
func From[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}
	return *v
}

// Map applies transform function f to the value pointed to by v and returns
// a pointer to the result. Returns nil if v is nil.
func Map[T, U any](v *T, f func(T) U) *U {
	if v == nil {
		return nil
	}
	res := f(*v)
	return &res
}
