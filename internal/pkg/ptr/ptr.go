package ptr

func To[T any](v T) *T {
	return &v
}

func Map[T any](v *T) *T {
	if v == nil {
		return nil
	}
	vCopy := *v
	return &vCopy
}
