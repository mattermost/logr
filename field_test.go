package logr

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
		name       string
		field      Field
		wantW      string
		wantCommas int
		wantErr    bool
	}{
		{name: "nil", field: Map[map[string]any]("map", nil), wantW: "", wantErr: false},
		{name: "empty", field: Map("map", map[string]any{}), wantW: "", wantErr: false},
		{name: "one elements", field: Map("map", map[string]int{"foo": 0}), wantW: "foo=0", wantErr: false},
		{name: "two elements", field: Map("map", map[string]int{"foo": 0, "bar": 1}), wantW: "foo=0,bar=1", wantCommas: 1, wantErr: false},
		{name: "three elements", field: Map("map", map[string]int{"foo": 0, "bar": 1, "xyz": 2}), wantW: "foo=0,bar=1,xyz=2", wantCommas: 2, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := tt.field.ValueString(w, nil); (err != nil) != tt.wantErr {
				t.Errorf("Field.ValueString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantCommas == 0 {
				if gotW := w.String(); gotW != tt.wantW {
					t.Errorf("Field.ValueString() = %v, want %v", gotW, tt.wantW)
				}
			} else {
				gotW := w.String()
				if commas := strings.Count(gotW, ","); commas != tt.wantCommas {
					t.Errorf("Got %d commas in %s, want %d", commas, gotW, tt.wantCommas)
				}

				gotElems := strings.Split(gotW, ",")
				wantElems := strings.Split(tt.wantW, ",")
				assert.ElementsMatch(t, gotElems, wantElems)
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

// TestFieldSorter tests the sorting interface for fields
func TestFieldSorter_Len(t *testing.T) {
	fields := []Field{
		String("a", "1"),
		String("b", "2"),
		String("c", "3"),
	}

	sorter := FieldSorter(fields)
	assert.Equal(t, 3, sorter.Len())

	emptyFields := []Field{}
	emptySorter := FieldSorter(emptyFields)
	assert.Equal(t, 0, emptySorter.Len())
}

func TestFieldSorter_Less(t *testing.T) {
	fields := []Field{
		String("zebra", "1"),
		String("alpha", "2"),
		String("middle", "3"),
	}

	sorter := FieldSorter(fields)

	// zebra > alpha, so Less(0, 1) should be false
	assert.False(t, sorter.Less(0, 1))

	// alpha < zebra, so Less(1, 0) should be true
	assert.True(t, sorter.Less(1, 0))

	// alpha < middle, so Less(1, 2) should be true
	assert.True(t, sorter.Less(1, 2))
}

func TestFieldSorter_Swap(t *testing.T) {
	fields := []Field{
		String("a", "1"),
		String("b", "2"),
		String("c", "3"),
	}

	sorter := FieldSorter(fields)

	// Swap first and last
	sorter.Swap(0, 2)

	assert.Equal(t, "c", fields[0].Key)
	assert.Equal(t, "b", fields[1].Key)
	assert.Equal(t, "a", fields[2].Key)
}

func TestFieldSorter_SortIntegration(t *testing.T) {
	fields := []Field{
		String("zebra", "1"),
		String("alpha", "2"),
		String("middle", "3"),
		String("beta", "4"),
	}

	sorter := FieldSorter(fields)

	// Use standard sort
	for i := 0; i < sorter.Len()-1; i++ {
		for j := i + 1; j < sorter.Len(); j++ {
			if sorter.Less(j, i) {
				sorter.Swap(i, j)
			}
		}
	}

	// Check sorted order
	assert.Equal(t, "alpha", fields[0].Key)
	assert.Equal(t, "beta", fields[1].Key)
	assert.Equal(t, "middle", fields[2].Key)
	assert.Equal(t, "zebra", fields[3].Key)
}
