package choice

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
)

const booleanEnumTag reflect.StructTag = `choice:"0:false" choice:"1:true" enumAccept:"boolean" enumStyle:"boolean"`

type booleanFlag int8

func TestBooleanEnumStorageAndWire(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeFor[int](), reflect.TypeFor[int8](), reflect.TypeFor[int16](), reflect.TypeFor[int32](), reflect.TypeFor[int64](), reflect.TypeFor[uint](), reflect.TypeFor[uint8](), reflect.TypeFor[uint16](), reflect.TypeFor[uint32](), reflect.TypeFor[uint64](), reflect.TypeFor[*booleanFlag]()} {
		plan, err := ParseEnum(booleanEnumTag, typ)
		if err != nil {
			t.Fatal(typ, err)
		}
		if plan.WireType(true) != "boolean" || plan.WireType(false) != "boolean" || plan.InputDomain() != nil || plan.OutputDomain() != nil {
			t.Fatal("incorrect single boolean domain")
		}
		for _, input := range []bool{false, true} {
			code := "0"
			if input {
				code = "1"
			}
			got, err := plan.DecodeJSON([]byte(fmt.Sprintf(" \n%v\t", input)))
			if err != nil || string(got) != code {
				t.Fatalf("decode: %s %v", got, err)
			}
			stored, err := plan.storage(json.Number(code))
			if err != nil {
				t.Fatal(err)
			}
			if typ.Kind() == reflect.Pointer {
				ptr := reflect.New(plan.Type)
				ptr.Elem().Set(reflect.ValueOf(stored))
				stored = ptr.Interface()
			}
			wire, err := plan.Encode(stored)
			if err != nil || wire != input {
				t.Fatalf("encode: %v %v", wire, err)
			}
		}
		for _, input := range []bool{false, true} {
			if !reflect.DeepEqual(plan.WireValues(input), []any{false, true}) {
				t.Fatal(plan.WireValues(input))
			}
		}
		values, err := plan.StorageValues()
		if err != nil || len(values) != 2 {
			t.Fatal(values, err)
		}
		for _, raw := range []string{`0`, `1`, `2`, `1.0`, `1e0`, `"false"`, `"true"`, `[]`, `{}`, `false true`, ``, `False`} {
			if _, err := plan.DecodeJSON([]byte(raw)); err == nil {
				t.Fatalf("accepted %s", raw)
			}
		}
		if raw, err := plan.DecodeJSON([]byte(`null`)); err != nil || string(raw) != "null" {
			t.Fatal(string(raw), err)
		}
		for _, stored := range []any{-1, 2, uint64(2), int64(1 << 60)} {
			if _, err := plan.Encode(stored); !errors.Is(fmt.Errorf("field: %w", err), ErrUnknownOutput) {
				t.Fatal(stored, err)
			}
		}
		if got, err := plan.Encode((*booleanFlag)(nil)); err != nil || got != nil {
			t.Fatal(got, err)
		}
	}
	strict, err := ParseEnum(booleanEnumTag+` enumNull:"reject"`, reflect.TypeFor[*int]())
	if err != nil {
		t.Fatal(err)
	}
	if strict.AllowsNullInput() {
		t.Fatal("null allowed")
	}
	if _, err := strict.DecodeJSON([]byte(`null`)); !errors.Is(err, ErrNullInput) {
		t.Fatal(err)
	}
}

func TestBooleanEnumDeclarations(t *testing.T) {
	for _, choice := range []reflect.StructTag{`choice:"false" choice:"true"`, `choice:"0:true" choice:"1:false"`, `choice:"0:false"`, `choice:"0:false" choice:"2:true"`, `choice:"0:false" choice:"1:true" choice:"2:other"`, `choice:"00:false" choice:"1:true"`} {
		if _, err := ParseEnum(choice+` enumStyle:"boolean"`, reflect.TypeFor[int]()); err == nil {
			t.Fatal(choice)
		}
	}
	for _, typ := range []reflect.Type{reflect.TypeFor[string](), reflect.TypeFor[bool](), reflect.TypeFor[float64](), reflect.TypeFor[ScalarValue]()} {
		if _, err := ParseEnum(booleanEnumTag, typ); err == nil {
			t.Fatal(typ)
		}
	}
	for _, policy := range []reflect.StructTag{` enumAlias:"0:no"`, ` enumCase:"fold"`, ` enumTrim:"space"`, ` enumInteger:"passthrough"`, ` enumUnknownString:"0"`, ` enumUnknownInteger:"0"`, ` enumOutputAlias:"0:no"`, ` enumOutputUnknown:"reject"`} {
		if _, err := ParseEnum(booleanEnumTag+policy, reflect.TypeFor[int]()); err == nil {
			t.Fatal(policy)
		}
	}
	// Input and output representations retain their independent owners.
	for _, tc := range []struct {
		tag       reflect.StructTag
		raw, want string
		wire      any
	}{
		{`enumAccept:"boolean" enumStyle:"ordinal"`, `true`, `1`, json.Number("1")},
		{`enumAccept:"literal" enumStyle:"boolean"`, `"true"`, `1`, true},
		{`enumAccept:"ordinal" enumStyle:"boolean"`, `1`, `1`, true},
		{`enumAccept:"scalar" enumStyle:"boolean" enumAlias:"1:yes"`, `"yes"`, `1`, true},
	} {
		plan, err := ParseEnum(`choice:"0:false" choice:"1:true" `+tc.tag, reflect.TypeFor[int]())
		if err != nil {
			t.Fatal(err)
		}
		raw, err := plan.DecodeJSON([]byte(tc.raw))
		if err != nil || string(raw) != tc.want {
			t.Fatal(string(raw), err)
		}
		wire, err := plan.Encode(1)
		if err != nil || wire != tc.wire {
			t.Fatal(wire, err)
		}
	}
}

func TestBooleanEnumConcurrentPlan(t *testing.T) {
	plan, err := ParseEnum(booleanEnumTag, reflect.TypeFor[int]())
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 100; j++ {
				if got, err := plan.Encode(1); got != true || err != nil {
					t.Error(got, err)
				}
				if got, err := plan.DecodeJSON([]byte(`false`)); string(got) != "0" || err != nil {
					t.Error(string(got), err)
				}
				plan.WireValues(true)[0] = "mutated"
			}
		}()
	}
	workers.Wait()
	if !reflect.DeepEqual(plan.WireValues(true), []any{false, true}) {
		t.Fatal("metadata mutation")
	}
}

func BenchmarkBooleanEnumWarm(b *testing.B) {
	plan, err := ParseEnum(booleanEnumTag, reflect.TypeFor[int]())
	if err != nil {
		b.Fatal(err)
	}
	b.Run("Encode", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := plan.Encode(1); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("DecodeJSON", func(b *testing.B) {
		raw := []byte(`true`)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := plan.DecodeJSON(raw); err != nil {
				b.Fatal(err)
			}
		}
	})
}
