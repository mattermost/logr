package logr

import (
	"errors"
	"testing"
	"time"
)

func TestFieldInt(t *testing.T) {
	_ = Int("int", int(0))
	type myInt int
	_ = Int("int", myInt(0))

	_ = Int("int8", int8(0))
	type myInt8 int8
	_ = Int("int8", myInt8(0))

	_ = Int("int16", int16(0))
	type myInt16 int16
	_ = Int("int16", myInt16(0))

	_ = Int("int32", int32(0))
	type myInt32 int32
	_ = Int("int32", myInt32(0))

	_ = Int("int64", int64(0))
	type myInt64 int64
	_ = Int("int64", myInt64(0))
}

func TestFieldUint(t *testing.T) {
	_ = Uint("uint", uint(0))
	type myUint uint
	_ = Uint("uint", myUint(0))

	_ = Uint("uint8", uint8(0))
	type myUint8 uint8
	_ = Uint("uint8", myUint8(0))

	_ = Uint("uint16", uint16(0))
	type myUint16 uint16
	_ = Uint("uint16", myUint16(0))

	_ = Uint("uint32", uint32(0))
	type myUint32 uint32
	_ = Uint("uint32", myUint32(0))

	_ = Uint("uint64", uint64(0))
	type myUint64 uint64
	_ = Uint("uint64", myUint64(0))

	_ = Uint("uintptr", uintptr(0))
	type myUintptr uintptr
	_ = Uint("uintptr", myUintptr(0))
}

func TestFieldFloat(t *testing.T) {
	_ = Float("float32", float32(0))
	type myFloat32 float32
	_ = Float("float32", myFloat32(0))

	_ = Float("float64", float64(0))
	type myFloat64 float32
	_ = Float("float64", myFloat64(0))
}

func TestFieldString(t *testing.T) {
	_ = String("string", "foo")
	type myString string
	_ = String("string", myString("foo"))

	type myByteSlice string
	_ = String("string", []byte{})
	_ = String("string", myByteSlice([]byte{}))
}

func TestFieldStringer(t *testing.T) {
	_ = Stringer("stringer", time.Now())
	type myStringer = time.Time
	_ = Stringer("stringer", myStringer(time.Now()))
}

func TestFieldErr(t *testing.T) {
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
