package logr

import "testing"

func TestFieldArray(t *testing.T) {
	Array("array", []string{})
	Array("array", []any{})

	type myArray []any
	Array("array", myArray{})

	type myGenericArray[T any] []T
	Array("array", myGenericArray[any]{})
}

func TestFieldMap(t *testing.T) {
	Map("array", map[string]any{})
	Map("array", map[int]any{})

	type myMap map[string]string
	Map("array", myMap{})

	type myGenericMap[K comparable, V any] map[K]V
	Map("array", myGenericMap[int, any]{})
}
