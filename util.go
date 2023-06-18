package main

import (
	"fmt"
	"os"
	"path"
	"runtime/debug"
	"strings"
)

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

func Getenv(key, val string) string {
	s := os.Getenv(key)
	if s == "" {
		return val
	}
	return s
}

// Get source of code
func src(deep int) (s string) {
	s = string(debug.Stack())
	str := strings.Split(s, "\n")
	if l := len(str); l <= deep {
		deep = l - 1
		for k, v := range str {
			lt.Println(k, v)
		}
	}
	s = str[deep]
	s = strings.Split(s, " +0x")[0]
	_, s = path.Split(s)
	s += ":"
	return
}

// Wrap source of code and message to error
func Errorf(format string, args ...any) error {
	return fmt.Errorf("%s %w", src(8), fmt.Errorf(format, args...))
}

// Wrap source of code and error to error
func srcError(err error) error {
	if err != nil {
		return fmt.Errorf("%s %w", src(8), err)
	}
	return err
}

func PrintOk(s string, err error) {
	if err != nil {
		let.Println(src(8), s, err)
	} else {
		lt.Println(src(8), s, "ok")
	}
}
