package logr

type FieldType uint8

// Field type, used to pass to `With`.
type Field struct {
	Key       string
	Type      FieldType
	Integer   int64
	String    string
	Interface interface{}
}
