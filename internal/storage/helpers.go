package storage

// Diff compares the current and incoming values and returns the difference.
func Diff[T any, V comparable](current, incoming []T, val func(T) V) (add, rm []T) {
	curr := make(map[V]struct{}, len(current))
	in := make(map[V]struct{}, len(incoming))

	for _, c := range current {
		curr[val(c)] = struct{}{}
	}

	for _, i := range incoming {
		in[val(i)] = struct{}{}
	}

	for _, c := range current {
		if _, ok := in[val(c)]; !ok {
			rm = append(rm, c)
		}
	}

	for _, i := range incoming {
		if _, ok := curr[val(i)]; !ok {
			add = append(add, i)
		}
	}

	return
}
