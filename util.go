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
