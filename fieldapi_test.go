package logr

import (
	"errors"
	"testing"
	"time"
)

func TestFielInt(t *testing.T) {
	_ = Int("int", int(0))
	type myUnit int
	_ = Int("int", myUnit(0))

	_ = Int("int8", int8(0))
	type myUnit8 int8
	_ = Int("int8", myUnit8(0))

	_ = Int("int16", int16(0))
	type myUnit16 int16
	_ = Int("int16", myUnit16(0))

	_ = Int("int32", int32(0))
	type myUnit32 int32
	_ = Int("int32", myUnit32(0))

	_ = Int("int64", int64(0))
	type myUnit64 int64
	_ = Int("int64", myUnit64(0))
}

func TestFielUnit(t *testing.T) {
	_ = Uint("uint", uint(0))
	type myUnit uint
	_ = Uint("uint", myUnit(0))

	_ = Uint("uint8", uint8(0))
	type myUnit8 uint8
	_ = Uint("uint8", myUnit8(0))

	_ = Uint("uint16", uint16(0))
	type myUnit16 uint16
	_ = Uint("uint16", myUnit16(0))

	_ = Uint("uint32", uint32(0))
	type myUnit32 uint32
	_ = Uint("uint32", myUnit32(0))

	_ = Uint("uint64", uint64(0))
	type myUnit64 uint64
	_ = Uint("uint64", myUnit64(0))

	_ = Uint("uintptr", uintptr(0))
	type myUintptr uintptr
	_ = Uint("uintptr", myUintptr(0))
}

func TestFielFloat(t *testing.T) {
	_ = Float("float32", float32(0))
	type myFloat32 float32
	_ = Float("float32", myFloat32(0))

	_ = Float("float64", float64(0))
	type myFloat64 float32
	_ = Float("float64", myFloat64(0))
}

func TestFielString(t *testing.T) {
	_ = String("string", "foo")
	type myString string
	_ = String("string", myString("foo"))
}

func TestFielStringer(t *testing.T) {
	_ = Stringer("stringer", time.Now())
	type myStringer = time.Time
	_ = Stringer("stringer", myStringer(time.Now()))
}

func TestFielErr(t *testing.T) {
	_ = Err(errors.New("some error"))
	type myError error
	_ = Err(myError(errors.New("some error")))
}

func TestFieldNamedErr(t *testing.T) {
	_ = NamedErr("named err", errors.New("some error"))
	type myError error
	_ = NamedErr("named err", myError(errors.New("some error")))
}

func TestFieldBool(t *testing.T) {
	_ = Bool("bool", false)
	type myBool bool
	_ = Bool("bool", myBool(false))
}

func TestFieldDuration(t *testing.T) {
	_ = Duration("duration", time.Duration(0))
}

func TestFieldMillis(t *testing.T) {
	_ = Millis("millis", int64(0))
}

func TestFieldArray(t *testing.T) {
	_ = Array("array", []string{})
	_ = Array("array", []any{})

	type myArray []any
	_ = Array("array", myArray{})

	type myGenericArray[T any] []T
	_ = Array("array", myGenericArray[any]{})
}

func TestFieldMap(t *testing.T) {
	_ = Map("array", map[string]any{})
	_ = Map("array", map[int]any{})

	type myMap map[string]string
	_ = Map("array", myMap{})

	type myGenericMap[K comparable, V any] map[K]V
	_ = Map("array", myGenericMap[int, any]{})
}
