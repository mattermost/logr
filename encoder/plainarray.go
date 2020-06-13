package encoder

import (
	"fmt"
	"io"
	"time"

	"github.com/wiggin77/logr"
)

type plainArrayEncoder struct {
	w         io.Writer
	separator string
	sep       bool
}

func NewPlainArrayEncoder(w io.Writer, separator string) *plainArrayEncoder {
	return &plainArrayEncoder{w: w, separator: separator}
}

func (enc *plainArrayEncoder) writeSep() {
	if enc.sep {
		enc.w.Write([]byte(enc.separator))
	} else {
		enc.sep = true
	}
}

func (enc *plainArrayEncoder) AppendBool(value bool) {
	enc.writeSep()
	if value {
		enc.w.Write([]byte("true"))
	} else {
		enc.w.Write([]byte("false"))
	}
}

func (enc *plainArrayEncoder) AppendByteString(value []byte) {
	enc.writeSep()
	fmt.Fprintf(enc.w, "%s", value)
}

func (enc *plainArrayEncoder) AppendComplex128(value complex128) {
	enc.writeSep()
	fmt.Fprintf(enc.w, "%v", value)
}

func (enc *plainArrayEncoder) AppendComplex64(value complex64) {
	enc.writeSep()
	fmt.Fprintf(enc.w, "%v", value)
}

func (enc *plainArrayEncoder) AppendFloat64(value float64) {
	enc.writeSep()
	fmt.Fprintf(enc.w, "%f", value)
}

func (enc *plainArrayEncoder) AppendFloat32(value float32) {
	enc.AppendFloat64(float64(value))
}

func (enc *plainArrayEncoder) AppendInt(value int) {
	enc.AppendInt64(int64(value))
}

func (enc *plainArrayEncoder) AppendInt64(value int64) {
	enc.writeSep()
	fmt.Fprintf(enc.w, "%d", value)
}

func (enc *plainArrayEncoder) AppendInt32(value int32) {
	enc.AppendInt64(int64(value))
}

func (enc *plainArrayEncoder) AppendInt16(value int16) {
	enc.AppendInt64(int64(value))
}

func (enc *plainArrayEncoder) AppendInt8(value int8) {
	enc.AppendInt64(int64(value))
}

func (enc *plainArrayEncoder) AppendString(value string) {
	enc.writeSep()
	enc.w.Write([]byte(value))
}

func (enc *plainArrayEncoder) AppendUint(value uint) {
	enc.AppendUint64(uint64(value))
}

func (enc *plainArrayEncoder) AppendUint64(value uint64) {
	enc.writeSep()
	fmt.Fprintf(enc.w, "%d", value)
}

func (enc *plainArrayEncoder) AppendUint32(value uint32) {
	enc.AppendUint64(uint64(value))
}

func (enc *plainArrayEncoder) AppendUint16(value uint16) {
	enc.AppendUint64(uint64(value))
}

func (enc *plainArrayEncoder) AppendUint8(value uint8) {
	enc.AppendUint64(uint64(value))
}

func (enc *plainArrayEncoder) AppendUintptr(value uintptr) {
	enc.writeSep()
	fmt.Fprintf(enc.w, "%v", value)
}

func (enc *plainArrayEncoder) AppendDuration(value time.Duration) {
	enc.writeSep()
	enc.w.Write([]byte(value.String()))
}

func (enc *plainArrayEncoder) AppendTime(value time.Time) {
	enc.writeSep()
	enc.w.Write([]byte(value.Format(logr.DefTimestampFormat)))
}

func (enc *plainArrayEncoder) AppendArray(marshaler logr.ArrayMarshaler) error {
	enc.w.Write([]byte("["))
	err := marshaler.MarshalLogArray(enc)
	enc.w.Write([]byte("]"))
	return err
}

func (enc *plainArrayEncoder) AppendObject(marshaler logr.ObjectMarshaler) error {
	enc.w.Write([]byte("{"))
	objEncoder := NewPlainObjectEncoder(enc.w, enc.separator)
	err := marshaler.MarshalLogObject(objEncoder)
	enc.w.Write([]byte("}"))
	return err
}

func (enc *plainArrayEncoder) AppendReflected(value interface{}) error {
	enc.writeSep()
	fmt.Fprintf(enc.w, "%v", value)
	return nil
}
