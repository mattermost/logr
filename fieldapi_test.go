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
