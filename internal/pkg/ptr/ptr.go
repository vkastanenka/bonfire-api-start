package ptr

// To returns a pointer to a copy of value v.
func To[T any](v T) *T {
	return &v
}

// Clone returns a pointer to a copy of the value pointed to by v.
// Returns nil if v is nil.
func Clone[T any](v *T) *T {
	if v == nil {
		return nil
	}
	vCopy := *v
	return &vCopy
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
