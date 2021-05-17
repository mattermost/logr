package logr

import (
	"bytes"
	"testing"
)

/*
	UnknownType FieldType = iota
	StringType
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
		// TODO: Add test cases.
		{name: "StringType", field: Field{Key: "str", Type: StringType, String: "test"}, wantW: "test", wantErr: false},
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
