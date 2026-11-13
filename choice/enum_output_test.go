package choice

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
)

type outputToken string

const outputAliasTag reflect.StructTag = `choice:"0:Pending" choice:"10:Active" choice:"20:Unsupported" enumAccept:"scalar" enumStyle:"literal" enumAlias:"10:input-active" enumCase:"fold" enumTrim:"space" enumOutputAlias:"0:pending-token" enumOutputAlias:"10:active:token" enumOutputAlias:"20:unknown"`

func TestOutputAliasesEncodeStoredTokens(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(outputToken("")), reflect.TypeOf((*outputToken)(nil))} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/strict=%v", typ, strict), func(t *testing.T) {
				tag := outputAliasTag
				if strict {
					tag += ` enumOutputUnknown:"reject"`
				}
				plan, err := ParseEnum(tag, typ)
				if err != nil {
					t.Fatal(err)
				}
				for _, tc := range []struct{ stored, want string }{
					{"Pending", "Pending"}, {"Active", "Active"}, {"Unsupported", "Unsupported"},
					{"pending-token", "Pending"}, {"active:token", "Active"}, {"unknown", "Unsupported"},
				} {
					stored := reflect.New(plan.Type).Elem()
					stored.SetString(tc.stored)
					value := stored.Interface()
					if typ.Kind() == reflect.Pointer {
						value = stored.Addr().Interface()
					}
					got, err := plan.Encode(value)
					if err != nil || got != tc.want {
						t.Fatalf("Encode(%q)=%v, %v; want %q", tc.stored, got, err, tc.want)
					}
				}
				// Output normalization is exact: input folding/trimming/aliases
				// never implicitly broaden the stored-token mapping.
				for _, unknown := range []string{"arbitrary", "UNKNOWN", " unknown ", "active", "input-active", ""} {
					got, err := plan.Encode(outputToken(unknown))
					if strict {
						if got != nil || !errors.Is(err, ErrUnknownOutput) {
							t.Fatalf("strict Encode(%q)=%v, %v", unknown, got, err)
						}
					} else if err != nil || got != unknown {
						t.Fatalf("passthrough Encode(%q)=%v, %v", unknown, got, err)
					}
				}
				for _, absent := range []any{nil, (*outputToken)(nil), (**outputToken)(nil)} {
					if got, err := plan.Encode(absent); err != nil || got != nil {
						t.Fatalf("nil output=%v, %v", got, err)
					}
				}
				if plan.WireType(false) != "string" || plan.OutputDomain() != nil {
					t.Fatal("output schema must use native string representation")
				}
				want := []any(nil)
				if strict {
					want = []any{"Pending", "Active", "Unsupported"}
				}
				if got := plan.WireValues(false); !reflect.DeepEqual(got, want) {
					t.Fatalf("output choices=%v; want %v", got, want)
				}
			})
		}
	}
}

func TestOutputAliasesLeaveInputPoliciesUnchanged(t *testing.T) {
	baseline, err := ParseEnum(`choice:"0:Pending" choice:"10:Active" choice:"20:Unsupported" enumAccept:"scalar" enumStyle:"literal" enumAlias:"10:input-active" enumCase:"fold" enumTrim:"space"`, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	for _, policy := range []reflect.StructTag{"", ` enumOutputUnknown:"reject"`} {
		plan, err := ParseEnum(outputAliasTag+policy, reflect.TypeOf(""))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(plan.InputDomain(), baseline.InputDomain()) {
			t.Fatal("output aliases changed input schema")
		}
		for _, raw := range []string{`0`, `10`, `"Active"`, `" active "`, `"input-active"`, `"pending-token"`, `"active:token"`, `"unknown"`, `"arbitrary"`, `null`} {
			got, gotErr := plan.DecodeJSON([]byte(raw))
			want, wantErr := baseline.DecodeJSON([]byte(raw))
			if string(got) != string(want) || (gotErr == nil) != (wantErr == nil) {
				t.Fatalf("input %s changed: %s, %v; baseline %s, %v", raw, got, gotErr, want, wantErr)
			}
		}
	}
	// Legacy literal input remains permissive even with strict output.
	plan, err := ParseEnum(`choice:"0:Pending" enumAccept:"literal" enumOutputUnknown:"reject" enumOutputAlias:"0:token"`, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := plan.DecodeJSON([]byte(`"token"`)); err != nil || string(got) != `"token"` {
		t.Fatalf("literal input changed: %s, %v", got, err)
	}
}

func TestOutputAliasesRejectInvalidDeclarations(t *testing.T) {
	for _, tc := range []struct {
		tag reflect.StructTag
		typ reflect.Type
	}{
		{`enumOutputAlias:"0:token"`, reflect.TypeOf("")},
		{`enumOutputUnknown:"reject"`, reflect.TypeOf("")},
		{`enumOutputAlias:unquoted`, reflect.TypeOf("")},
		{`enumOutputUnknown:unquoted`, reflect.TypeOf("")},
		{`choice:"0:Pending" enumOutputAlias:"token"`, reflect.TypeOf("")},
		{`choice:"0:Pending" enumOutputAlias:"no:token"`, reflect.TypeOf("")},
		{`choice:"0:Pending" enumOutputAlias:"1.0:token"`, reflect.TypeOf("")},
		{`choice:"0:Pending" enumOutputAlias:"1:token"`, reflect.TypeOf("")},
		{`choice:"0:Pending" enumOutputAlias:"0:token" enumOutputAlias:"0:token"`, reflect.TypeOf("")},
		{`choice:"0:Pending" choice:"1:Active" enumOutputAlias:"0:token" enumOutputAlias:"1:token"`, reflect.TypeOf("")},
		{`choice:"0:Pending" enumOutputAlias:"0:Pending"`, reflect.TypeOf("")},
		{`choice:"0:Pending" choice:"1:Active" enumOutputAlias:"0:Active"`, reflect.TypeOf("")},
		{`choice:"0:Pending" enumOutputUnknown:""`, reflect.TypeOf("")},
		{`choice:"0:Pending" enumOutputUnknown:"passthrough"`, reflect.TypeOf("")},
		{`choice:"0:Pending" enumOutputUnknown:"reject" enumOutputUnknown:"reject"`, reflect.TypeOf("")},
		{`choice:"0:Pending" enumStyle:"ordinal" enumOutputAlias:"0:token"`, reflect.TypeOf("")},
		{`choice:"0:Pending" enumStyle:"ordinal" enumOutputUnknown:"reject"`, reflect.TypeOf("")},
		{`choice:"0:Pending" enumStyle:"literal" enumOutputAlias:"0:token"`, reflect.TypeOf(0)},
		{`choice:"0:Pending" enumStyle:"literal" enumOutputUnknown:"reject"`, reflect.TypeOf(0)},
		{`choice:"0:Pending" enumAccept:"scalar" enumStyle:"scalar" enumOutputAlias:"0:token"`, reflect.TypeOf(ScalarValue{})},
		{`choice:"0:Pending" enumAccept:"scalar" enumStyle:"scalar" enumOutputUnknown:"reject"`, reflect.TypeOf(ScalarValue{})},
		{`choice:"0:Pending" enumOutputAlias:"0:token"`, reflect.TypeOf(wireJSONString(""))},
	} {
		t.Run(string(tc.tag)+tc.typ.String(), func(t *testing.T) {
			if _, err := ParseEnum(tc.tag, tc.typ); err == nil {
				t.Fatal("accepted invalid output policy")
			}
		})
	}
}

func TestOutputAliasesUseImmutablePlan(t *testing.T) {
	plan, err := ParseEnum(outputAliasTag+` enumOutputUnknown:"reject"`, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 100; j++ {
				if got, err := plan.Encode("unknown"); got != "Unsupported" || err != nil {
					t.Errorf("Encode=%v, %v", got, err)
				}
				values := plan.WireValues(false)
				values[0] = "changed"
			}
		}()
	}
	workers.Wait()
	if got, err := plan.Encode("Pending"); got != "Pending" || err != nil {
		t.Fatalf("metadata mutated output plan: %v, %v", got, err)
	}
}

func TestOutputPolicyWithoutAliasesAndExactEmptyTokens(t *testing.T) {
	strict, err := ParseEnum(`choice:"7:Canonical" enumOutputUnknown:"reject"`, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := strict.Encode("Canonical"); got != "Canonical" || err != nil {
		t.Fatalf("canonical output=%v, %v", got, err)
	}
	if _, err := strict.Encode("other"); !errors.Is(err, ErrUnknownOutput) {
		t.Fatalf("unknown output error=%v", err)
	}
	if !reflect.DeepEqual(strict.WireValues(false), []any{"Canonical"}) {
		t.Fatal("strict policy without aliases lost finite output schema")
	}
	exact, err := ParseEnum(`choice:"0:Empty" choice:"1:Lower" choice:"2:Upper" enumOutputAlias:"0:" enumOutputAlias:"1:token" enumOutputAlias:"2:TOKEN"`, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	for token, want := range map[string]string{"": "Empty", "token": "Lower", "TOKEN": "Upper", "Token": "Token"} {
		if got, err := exact.Encode(token); got != want || err != nil {
			t.Fatalf("exact Encode(%q)=%v, %v; want %q", token, got, err, want)
		}
	}
	if !HasEnum(reflect.TypeOf(struct {
		Value string `enumOutputUnknown:"reject"`
	}{})) {
		t.Fatal("output-only policies must be discovered for compilation errors")
	}
}
