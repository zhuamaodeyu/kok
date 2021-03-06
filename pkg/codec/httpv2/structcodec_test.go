package codec

import (
	"reflect"
	"testing"
)

func TestDecodeMapToStruct(t *testing.T) {
	type value struct {
		Int    int    `kok:"int"`
		Int8   int8   `kok:"int8"`
		Int16  int16  `kok:"int16"`
		Int32  int32  `kok:"int32"`
		Int64  int64  `kok:"int64"`
		Uint   uint   `kok:"uint"`
		Uint8  uint8  `kok:"uint8"`
		Uint16 uint16 `kok:"uint16"`
		Uint32 uint32 `kok:"uint32"`
		Uint64 uint64 `kok:"uint64"`
		Bool   bool   `kok:"bool"`
		String string `kok:"string"`
	}

	cases := []struct {
		name    string
		in      map[string]string
		out     interface{}
		wantOut interface{}
		wantErr error
	}{
		{
			name: "struct pointer",
			in: map[string]string{
				"int":    "1",
				"int8":   "2",
				"int16":  "3",
				"int32":  "4",
				"int64":  "5",
				"uint":   "6",
				"uint8":  "7",
				"uint16": "8",
				"uint32": "9",
				"uint64": "10",
				"bool":   "true",
				"string": "hello",
			},
			out: new(value),
			wantOut: &value{
				Int:    1,
				Int8:   2,
				Int16:  3,
				Int32:  4,
				Int64:  5,
				Uint:   6,
				Uint8:  7,
				Uint16: 8,
				Uint32: 9,
				Uint64: 10,
				Bool:   true,
				String: "hello",
			},
		},
		{
			name: "struct",
			in: map[string]string{
				"int":    "1",
				"uint":   "6",
				"bool":   "true",
				"string": "hello",
			},
			out:     value{},
			wantErr: errUnsupportedType,
		},
		{
			name: "string",
			in: map[string]string{
				"int":    "1",
				"uint":   "6",
				"bool":   "true",
				"string": "hello",
			},
			out:     new(string),
			wantErr: errUnsupportedType,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := DecodeMapToStruct(c.in, c.out)
			if err != c.wantErr {
				t.Fatalf("Err: got (%#v), want (%#v)", err, c.wantErr)
			}
			if err == nil && !reflect.DeepEqual(c.out, c.wantOut) {
				t.Fatalf("Out: got (%#v), want (%#v)", c.out, c.wantOut)
			}
		})
	}
}

func TestEncodeStructToMap(t *testing.T) {
	type value struct {
		Int    int    `kok:"int"`
		Int8   int8   `kok:"int8"`
		Int16  int16  `kok:"int16"`
		Int32  int32  `kok:"int32"`
		Int64  int64  `kok:"int64"`
		Uint   uint   `kok:"uint"`
		Uint8  uint8  `kok:"uint8"`
		Uint16 uint16 `kok:"uint16"`
		Uint32 uint32 `kok:"uint32"`
		Uint64 uint64 `kok:"uint64"`
		Bool   bool   `kok:"bool"`
		String string `kok:"string"`
	}

	cases := []struct {
		name    string
		in      interface{}
		wantOut map[string]string
		wantErr error
	}{
		{
			name: "struct pointer",
			in: &value{
				Int:    1,
				Int8:   2,
				Int16:  3,
				Int32:  4,
				Int64:  5,
				Uint:   6,
				Uint8:  7,
				Uint16: 8,
				Uint32: 9,
				Uint64: 10,
				Bool:   true,
				String: "hello",
			},
			wantOut: map[string]string{
				"int":    "1",
				"int8":   "2",
				"int16":  "3",
				"int32":  "4",
				"int64":  "5",
				"uint":   "6",
				"uint8":  "7",
				"uint16": "8",
				"uint32": "9",
				"uint64": "10",
				"bool":   "true",
				"string": "hello",
			},
		},
		{
			name: "struct",
			in: value{
				Int:    1,
				Int8:   2,
				Int16:  3,
				Int32:  4,
				Int64:  5,
				Uint:   6,
				Uint8:  7,
				Uint16: 8,
				Uint32: 9,
				Uint64: 10,
				Bool:   true,
				String: "hello",
			},
			wantOut: map[string]string{
				"int":    "1",
				"int8":   "2",
				"int16":  "3",
				"int32":  "4",
				"int64":  "5",
				"uint":   "6",
				"uint8":  "7",
				"uint16": "8",
				"uint32": "9",
				"uint64": "10",
				"bool":   "true",
				"string": "hello",
			},
		},
		{
			name:    "string",
			in:      "",
			wantErr: errUnsupportedType,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := make(map[string]string)
			err := EncodeStructToMap(c.in, &out)
			if err != c.wantErr {
				t.Fatalf("Err: got (%#v), want (%#v)", err, c.wantErr)
			}
			if err == nil && !reflect.DeepEqual(out, c.wantOut) {
				t.Fatalf("Out: got (%#v), want (%#v)", out, c.wantOut)
			}
		})
	}
}
