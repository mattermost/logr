package logr

import (
	"bytes"
	"testing"
)

/*
	UnknownType FieldType = iota
	StringType
	StringerType
	StructType
	ErrorType
	BoolType
	TimestampMillisType
	TimeType
	DurationType
	Int64Type
	Int32Type
	IntType
	Uint64Type
	Uint32Type
	UintType
	Float64Type
	Float32Type
	BinaryType
	ArrayType
	MapType
*/

func TestField_ValueString(t *testing.T) {
	tests := []struct {
		name    string
		field   Field
		wantW   string
		wantErr bool
	}{
		{name: "StringType", field: String("str", "test"), wantW: "test", wantErr: false},
		{name: "StringerType", field: Stringer("strgr", newTestStringer("Hello")), wantW: "Hello", wantErr: false},
		{name: "StringerType with nil", field: Stringer("nilstrgr", nil), wantW: "", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := tt.field.ValueString(w, nil); (err != nil) != tt.wantErr {
				t.Errorf("Field.ValueString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("Field.ValueString() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func TestFieldForAny(t *testing.T) {
	testString := "hello"
	var nilPointer *string

	tests := []struct {
		name    string
		field   Field
		wantW   string
		wantErr bool
	}{
		{name: "StringType", field: Any("str", "test"), wantW: "test", wantErr: false},
		{name: "StringerType", field: Any("strgr", newTestStringer("Hello")), wantW: "Hello", wantErr: false},
		{name: "String pointer", field: Any("strptr", &testString), wantW: testString, wantErr: false},
		{name: "String pointer with nil", field: Any("nilptr", nilPointer), wantW: "", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := tt.field.ValueString(w, nil); (err != nil) != tt.wantErr {
				t.Errorf("Field.ValueString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("Field.ValueString() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func TestField_Array(t *testing.T) {
	tests := []struct {
		name    string
		field   Field
		wantW   string
		wantErr bool
	}{
		{name: "nil", field: Array[[]any]("array", nil), wantW: "", wantErr: false},
		{name: "empty", field: Array("array", []string{}), wantW: "", wantErr: false},
		{name: "one elements", field: Array("array", []string{"foo"}), wantW: "foo", wantErr: false},
		{name: "two elements", field: Array("array", []string{"foo", "bar"}), wantW: "foo,bar", wantErr: false},
		{name: "three elements", field: Array("array", []string{"foo", "bar", "xyz"}), wantW: "foo,bar,xyz", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := tt.field.ValueString(w, nil); (err != nil) != tt.wantErr {
				t.Errorf("Field.ValueString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("Field.ValueString() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func TestField_Map(t *testing.T) {
	tests := []struct {
		name    string
		field   Field
		wantW   string
		wantErr bool
	}{
		{name: "nil", field: Map[map[string]any]("map", nil), wantW: "", wantErr: false},

		{name: "empty", field: Map("map", map[string]any{}), wantW: "", wantErr: false},
		{name: "one elements", field: Map("map", map[string]int{"foo": 0}), wantW: "foo=0", wantErr: false},
		{name: "two elements", field: Map("map", map[string]int{"foo": 0, "bar": 1}), wantW: "foo=0,bar=1", wantErr: false},
		{name: "three elements", field: Map("map", map[string]int{"foo": 0, "bar": 1, "xyz": 2}), wantW: "foo=0,bar=1,xyz=2", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := tt.field.ValueString(w, nil); (err != nil) != tt.wantErr {
				t.Errorf("Field.ValueString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("Field.ValueString() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

type testStringer struct {
	s string
}

func newTestStringer(s string) *testStringer {
	return &testStringer{
		s: s,
	}
}

func (ts *testStringer) String() string {
	return ts.s
}
