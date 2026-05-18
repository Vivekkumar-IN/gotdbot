package gotdbot

func getVariadic[T comparable](opts []T, def T) T {
	if len(opts) == 0 {
		return def
	}
	first := opts[0]
	var zero T
	if first == zero {
		return def
	}
	return first
}
