package choice

import (
	"errors"
	"reflect"
	"sync"
	"testing"
)

const literalMembershipTag reflect.StructTag = `choice:"0:ASAP" choice:"1:SPEND_EVENLY" enumAccept:"literal" enumUnknownString:"reject"`

func TestLiteralStringMembership(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf(""), reflect.TypeOf((*string)(nil)), reflect.TypeOf(pacingAlias("")), reflect.TypeOf((*pacingAlias)(nil))} {
		t.Run(typ.String(), func(t *testing.T) {
			plan, err := ParseEnum(literalMembershipTag, typ)
			if err != nil {
				t.Fatal(err)
			}
			for _, raw := range []string{`"ASAP"`, `"SPEND_EVENLY"`, `null`} {
				got, err := plan.DecodeJSON([]byte(raw))
				if err != nil || string(got) != raw {
					t.Fatalf("%s: %s %v", raw, got, err)
				}
			}
			for _, raw := range []string{`"UNKNOWN"`, `""`, `"asap"`, `" ASAP "`, `"0"`, `0`, `1`, `true`, `{}`, `[]`} {
				if got, err := plan.DecodeJSON([]byte(raw)); err == nil {
					t.Fatalf("accepted %s as %s", raw, got)
				}
			}
			// The input policy does not impose a new output policy.
			if got, err := plan.Encode("UNKNOWN"); err != nil || got != "UNKNOWN" {
				t.Fatalf("output: %v %v", got, err)
			}
			legacy, err := ParseEnum(`choice:"0:ASAP" enumAccept:"literal"`, typ)
			if err != nil {
				t.Fatal(err)
			}
			for _, raw := range []string{`"UNKNOWN"`, `""`} {
				if got, err := legacy.DecodeJSON([]byte(raw)); err != nil || string(got) != raw {
					t.Fatalf("legacy: %s %v", got, err)
				}
			}
		})
	}
}

func TestLiteralStringMembershipNullEmptyAndOutputAliases(t *testing.T) {
	plan, err := ParseEnum(literalMembershipTag+` choice:"2:" enumNull:"reject" enumOutputAlias:"0:now" enumOutputUnknown:"reject"`, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := plan.DecodeJSON([]byte(`""`)); err != nil || string(got) != `""` {
		t.Fatalf("declared empty: %s %v", got, err)
	}
	if _, err := plan.DecodeJSON([]byte(`null`)); !errors.Is(err, ErrNullInput) {
		t.Fatalf("null cause: %v", err)
	}
	if _, err := plan.DecodeJSON([]byte(`"now"`)); err == nil {
		t.Fatal("output alias admitted as input")
	}
	if got, err := plan.Encode("now"); err != nil || got != "ASAP" {
		t.Fatalf("output alias: %v %v", got, err)
	}
	if _, err := plan.Encode("UNKNOWN"); !errors.Is(err, ErrUnknownOutput) {
		t.Fatalf("output cause: %v", err)
	}
}

func TestLiteralStringMembershipConfiguration(t *testing.T) {
	for _, tag := range []reflect.StructTag{
		`choice:"ASAP" enumUnknownString:"reject"`,
		`choice:"ASAP" enumAccept:"literal" enumUnknownString:"preserve"`,
		`choice:"ASAP" enumAccept:"literal" enumUnknownString:"0"`,
		`choice:"0:reject" enumAccept:"scalar" enumUnknownString:"reject"`,
		literalMembershipTag + ` enumAlias:"0:now"`, literalMembershipTag + ` enumCase:"fold"`,
		literalMembershipTag + ` enumTrim:"space"`, literalMembershipTag + ` enumInteger:"passthrough"`,
		literalMembershipTag + ` enumUnknownInteger:"0"`,
	} {
		if _, err := ParseEnum(tag, reflect.TypeOf("")); err == nil {
			t.Errorf("accepted %s", tag)
		}
	}
	for _, typ := range []reflect.Type{reflect.TypeOf(0), reflect.TypeOf([]string{}), reflect.TypeOf(wireJSONString(""))} {
		if _, err := ParseEnum(literalMembershipTag, typ); err == nil {
			t.Errorf("accepted storage %s", typ)
		}
	}
	if _, err := ParseEnum(literalMembershipTag+` mcpEnum:"true"`, reflect.TypeOf([]string{})); err == nil {
		t.Fatal("changed collection policy support")
	}
}

func TestLiteralStringMembershipConcurrentReuse(t *testing.T) {
	plan, err := ParseEnum(literalMembershipTag, reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 100; j++ {
				if _, err := plan.DecodeJSON([]byte(`"ASAP"`)); err != nil {
					t.Error(err)
				}
				if _, err := plan.DecodeJSON([]byte(`"UNKNOWN"`)); err == nil {
					t.Error("unknown admitted")
				}
			}
		}()
	}
	workers.Wait()
}

func BenchmarkLiteralStringMembership(b *testing.B) {
	plan, err := ParseEnum(literalMembershipTag, reflect.TypeOf(""))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := plan.DecodeJSON([]byte(`"ASAP"`)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLiteralStringMembershipPrepare(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := ParseEnum(literalMembershipTag, reflect.TypeOf("")); err != nil {
			b.Fatal(err)
		}
	}
}
