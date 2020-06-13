package encoder

import (
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/wiggin77/logr"
)

type plainObjectEncoder struct {
	w         io.Writer
	prefix    string
	separator string
	sep       bool
}

func NewPlainObjectEncoder(w io.Writer, separator string) *plainObjectEncoder {
	return &plainObjectEncoder{w: w, separator: separator}
}

func (enc *plainObjectEncoder) writeKey(key string) {
	if enc.prefix != "" {
		enc.w.Write([]byte(enc.prefix))
		enc.w.Write([]byte("."))
	}
	if enc.sep {
		enc.w.Write([]byte(enc.separator))
	} else {
		enc.sep = true
	}
	enc.w.Write([]byte(key))
	enc.w.Write([]byte(":"))
}

func (enc *plainObjectEncoder) AddArray(key string, marshaler logr.ArrayMarshaler) error {
	enc.writeKey(key)
	enc.writeKey("[")
	arrEncoder := NewPlainArrayEncoder(enc.w, enc.separator)
	err := marshaler.MarshalLogArray(arrEncoder)
	enc.writeKey("]")
	return err
}

func (enc *plainObjectEncoder) AddObject(key string, marshaler logr.ObjectMarshaler) error {
	enc.writeKey(key)
	enc.writeKey("{")
	err := marshaler.MarshalLogObject(enc)
	enc.writeKey("}")
	return err
}

func (enc *plainObjectEncoder) AddBinary(key string, value []byte) {
	enc.writeKey(key)
	enc.w.Write([]byte(hex.Dump(value)))
}

func (enc *plainObjectEncoder) AddByteString(key string, value []byte) {
	enc.writeKey(key)
	fmt.Fprintf(enc.w, "%s", value)
}

func (enc *plainObjectEncoder) AddBool(key string, value bool) {
	enc.writeKey(key)
	if value {
		enc.w.Write([]byte("true"))
	} else {
		enc.w.Write([]byte("false"))
	}
}

func (enc *plainObjectEncoder) AddComplex128(key string, value complex128) {
	enc.writeKey(key)
	fmt.Fprintf(enc.w, "%v", value)
}

func (enc *plainObjectEncoder) AddComplex64(key string, value complex64) {
	enc.writeKey(key)
	fmt.Fprintf(enc.w, "%v", value)
}

func (enc *plainObjectEncoder) AddDuration(key string, value time.Duration) {
	enc.writeKey(key)
	enc.w.Write([]byte(value.String()))
}

func (enc *plainObjectEncoder) AddFloat64(key string, value float64) {
	enc.writeKey(key)
	fmt.Fprintf(enc.w, "%f", value)
}

func (enc *plainObjectEncoder) AddFloat32(key string, value float32) {
	enc.AddFloat64(key, float64(value))
}

func (enc *plainObjectEncoder) AddInt(key string, value int) {
	enc.AddInt64(key, int64(value))
}

func (enc *plainObjectEncoder) AddInt64(key string, value int64) {
	enc.writeKey(key)
	fmt.Fprintf(enc.w, "%d", value)
}

func (enc *plainObjectEncoder) AddInt32(key string, value int32) {
	enc.AddInt64(key, int64(value))
}

func (enc *plainObjectEncoder) AddInt16(key string, value int16) {
	enc.AddInt64(key, int64(value))
}

func (enc *plainObjectEncoder) AddInt8(key string, value int8) {
	enc.AddInt64(key, int64(value))
}

func (enc *plainObjectEncoder) AddString(key, value string) {
	enc.writeKey(key)
	enc.w.Write([]byte(value))
}

func (enc *plainObjectEncoder) AddTime(key string, value time.Time) {
	enc.writeKey(key)
	enc.w.Write([]byte(value.Format(logr.DefTimestampFormat)))
}

func (enc *plainObjectEncoder) AddUint(key string, value uint) {
	enc.AddUint64(key, uint64(value))
}

func (enc *plainObjectEncoder) AddUint64(key string, value uint64) {
	enc.writeKey(key)
	fmt.Fprintf(enc.w, "%d", value)
}

func (enc *plainObjectEncoder) AddUint32(key string, value uint32) {
	enc.AddUint64(key, uint64(value))
}

func (enc *plainObjectEncoder) AddUint16(key string, value uint16) {
	enc.AddUint64(key, uint64(value))
}

func (enc *plainObjectEncoder) AddUint8(key string, value uint8) {
	enc.AddUint64(key, uint64(value))
}

func (enc *plainObjectEncoder) AddUintptr(key string, value uintptr) {
	enc.writeKey(key)
	fmt.Fprintf(enc.w, "%v", value)
}

func (enc *plainObjectEncoder) AddReflected(key string, value interface{}) error {
	enc.writeKey(key)
	fmt.Fprintf(enc.w, "%v", value)
	return nil
}

func (enc *plainObjectEncoder) OpenNamespace(key string) {
	if enc.prefix == "" {
		enc.prefix = key
		return
	}
	enc.prefix = key + "." + enc.prefix
}
