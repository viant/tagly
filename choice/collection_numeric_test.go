package choice

import (
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
)

func TestNumericCollectionScalarOwner(t *testing.T) {
	tag := reflect.StructTag(`mcpEnum:"true" enumAccept:"literal" enumStyle:"literal" choice:"1:DISPLAY" choice:"2:VIDEO"`)
	plan, err := ParseEnum(tag, reflect.TypeFor[[]int]())
	if err != nil || plan == nil {
		t.Fatal(plan, err)
	}
	if !reflect.DeepEqual(plan.CollectionItems().WireValues(), []any{"DISPLAY", "VIDEO"}) {
		t.Fatal(plan.CollectionItems())
	}
	for _, raw := range []string{`["DISPLAY","VIDEO"]`, `[]`, `null`, `[null]`} {
		got, err := plan.DecodeJSON([]byte(raw))
		if err != nil {
			t.Fatal(raw, err)
		}
		var stored []int
		if err := json.Unmarshal(got, &stored); err != nil {
			t.Fatal(err)
		}
		if raw == `["DISPLAY","VIDEO"]` && !reflect.DeepEqual(stored, []int{1, 2}) {
			t.Fatal(stored)
		}
	}
	for _, raw := range []string{`["OTHER"]`, `[1]`, `[{}]`, `"DISPLAY"`, `["DISPLAY"] []`} {
		if _, err := plan.DecodeJSON([]byte(raw)); err == nil {
			t.Fatal(raw)
		}
	}
	value, err := plan.Encode([]int{1, 2})
	if err != nil || !reflect.DeepEqual(value, []any{"DISPLAY", "VIDEO"}) {
		t.Fatal(value, err)
	}
	if _, err := plan.Encode([]int{3}); err == nil {
		t.Fatal("unknown output")
	}
	strict, err := ParseEnum(reflect.StructTag(string(tag)+` enumNull:"reject"`), reflect.TypeFor[[]int]())
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`null`, `[null]`} {
		if _, err := strict.DecodeJSON([]byte(raw)); !errors.Is(err, ErrNullInput) {
			t.Fatal(raw, err)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if _, err := plan.DecodeJSON([]byte(`["DISPLAY"]`)); err != nil {
					t.Error(err)
				}
				if _, err := plan.Encode([]int{1}); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
}

func TestNumericCollectionOrdinalAndOverflow(t *testing.T) {
	tag := reflect.StructTag(`mcpEnum:"true" enumAccept:"ordinal" enumStyle:"literal" choice:"1:ONE"`)
	plan, err := ParseCollectionEnum(tag, reflect.TypeFor[[]int8]())
	if err != nil {
		t.Fatal(err)
	}
	if got, err := plan.DecodeJSON([]byte(`[1]`)); err != nil || string(got) != `[1]` {
		t.Fatal(string(got), err)
	}
	for _, raw := range []string{`[128]`, `[1.1]`, `["ONE"]`} {
		if _, err := plan.DecodeJSON([]byte(raw)); err == nil {
			t.Fatal(raw)
		}
	}
	if _, err := ParseCollectionEnum(`mcpEnum:"true" enumAccept:"literal" choice:"128:OVERFLOW"`, reflect.TypeFor[[]int8]()); err == nil {
		t.Fatal("overflow declaration")
	}
}

func BenchmarkNumericCollectionConversion(b *testing.B) {
	plan, err := ParseEnum(`mcpEnum:"true" enumAccept:"literal" enumStyle:"literal" choice:"1:ONE"`, reflect.TypeFor[[]int]())
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := plan.DecodeJSON([]byte(`["ONE","ONE"]`)); err != nil {
			b.Fatal(err)
		}
		if _, err := plan.Encode([]int{1, 1}); err != nil {
			b.Fatal(err)
		}
	}
}

func TestNumericCollectionPublicDispatch(t *testing.T) {
	cases := []struct {
		tag               reflect.StructTag
		raw, want, output string
	}{
		{`mcpEnum:"true" choice:"1:ONE" enumStyle:"literal"`, `[1]`, `[1]`, `["ONE"]`},
		{`mcpEnum:"true" choice:"1:ONE" enumAccept:"ordinal" enumStyle:"literal"`, `[1]`, `[1]`, `["ONE"]`},
		{`mcpEnum:"true" choice:"1:ONE" enumAccept:"literal"`, `["ONE"]`, `[1]`, `[1]`},
		{`mcpEnum:"true" choice:"1:ONE" enumAccept:"scalar" enumStyle:"literal"`, `[1,"ONE"]`, `[1,1]`, `["ONE","ONE"]`},
		{`mcpEnum:"true" choice:"0:false" choice:"1:true" enumAccept:"boolean" enumStyle:"boolean"`, `[false,true]`, `[0,1]`, `[false,true]`},
	}
	for _, tc := range cases {
		t.Run(string(tc.tag), func(t *testing.T) {
			plan, err := ParseEnum(tc.tag, reflect.TypeFor[[]int]())
			if err != nil || plan == nil {
				t.Fatal(plan, err)
			}
			raw, err := plan.DecodeJSON([]byte(tc.raw))
			if err != nil || string(raw) != tc.want {
				t.Fatal(string(raw), err)
			}
			var stored []int
			if err := json.Unmarshal(raw, &stored); err != nil {
				t.Fatal(err)
			}
			output, err := plan.Encode(stored)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(output)
			if err != nil || string(encoded) != tc.output {
				t.Fatal(string(encoded), err)
			}
			if constraint, err := Parse(tc.tag, reflect.TypeFor[[]int]()); err != nil || constraint != nil {
				t.Fatal(constraint, err)
			}
		})
	}
	if plan, err := ParseEnum(``, reflect.TypeFor[[]int]()); plan != nil || err != nil {
		t.Fatal(plan, err)
	}
}

func TestManagedCollectionUnsupportedStorageRetainsScalarError(t *testing.T) {
	for _, storage := range []reflect.Type{reflect.TypeFor[[]bool](), reflect.TypeFor[[]float64](), reflect.TypeFor[[]struct{}]()} {
		if plan, err := ParseEnum(`mcpEnum:"true" choice:"1:ONE" enumAccept:"literal"`, storage); err == nil || plan != nil {
			t.Fatalf("unsupported storage %s: plan=%v error=%v", storage, plan, err)
		}
	}
}
