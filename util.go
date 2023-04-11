package main

func m2kv[K comparable, V any](m map[K]V) (keys []K, vals []V) {
	keys = make([]K, 0, len(m))
	vals = make([]V, 0, len(m))
	for key, val := range m {
		keys = append(keys, key)
		vals = append(vals, val)
	}
	return
}

//	func in[V comparable](v V, vs []V) bool {
//		if vs == nil {
//			return false
//		}
//		for _, val := range vs {
//			if val == v {
//				return true
//			}
//		}
//		return false
//	}
func set[V comparable](vs []V) (set []V, m map[V]struct{}) {
	if vs == nil {
		return
	}
	m = make(map[V]struct{}, 0)
	for _, val := range vs {
		if _, ok := m[val]; !ok {
			m[val] = struct{}{}
			set = append(set, val)
		}
	}
	return
}

func tf[V any](ok bool, t, f V) V {
	if ok {
		return t
	}
	return f
}
