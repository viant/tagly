package choice

import (
	"encoding/json"
	"reflect"
	"regexp"
	"testing"
)

const feeUnionTag reflect.StructTag = `choice:"4237:AD_SERVING" choice:"4241:AD_SERVING" choice:"92:OTHER" enumAccept:"scalar" enumStyle:"scalar" enumAlias:"4237:Ad Serving" enumCase:"fold" enumTrim:"space" enumInteger:"passthrough" enumUnknownString:"preserve" enumEmpty:"null" enumZero:"null"`

func TestScalarUnionNullPoliciesMatchClosedAndMixedDomains(t *testing.T) {
	for _, extra := range []string{"", ` enumInteger:"passthrough"`, ` enumUnknownString:"preserve"`} {
		tag := reflect.StructTag(`choice:"1:A" enumAccept:"scalar" enumStyle:"scalar" enumZero:"null" enumEmpty:"null" enumTrim:"space"` + extra)
		plan, err := ParseEnum(tag, reflect.TypeOf(ScalarValue{}))
		if err != nil {
			t.Fatal(err)
		}
		domain := plan.InputDomain()
		found := false
		for _, id := range domain.Integers {
			if id == "0" {
				found = true
			}
		}
		if !found {
			t.Fatal("accepted zero absent from input domain")
		}
		for _, input := range []string{"", " \t\n", "\u2003"} {
			found = false
			for _, alternative := range domain.StringAlternatives {
				if regexp.MustCompile(alternative.Pattern).MatchString(input) {
					found = true
				}
			}
			if !found {
				t.Fatalf("accepted empty/space %q absent from domain", input)
			}
			raw, _ := json.Marshal(input)
			if _, err := plan.DecodeJSON(raw); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestScalarUnionPoliciesRejectOtherStorage(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf(0), reflect.TypeOf(""), reflect.TypeOf((*string)(nil)), nil} {
		for _, tag := range []reflect.StructTag{`enumZero:"null"`, `enumEmpty:"null"`, `choice:"0:A" enumAccept:"scalar" enumZero:"null"`, `enumEmpty:"typo"`} {
			if _, err := ParseEnum(tag, typ); err == nil {
				t.Errorf("ignored policy %s for %v", tag, typ)
			}
		}
	}
}

func TestScalarUnionNativePlan(t *testing.T) {
	plan, err := ParseEnum(feeUnionTag, reflect.TypeOf(ScalarValue{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		raw, output string
		numeric     bool
	}{
		{`4237`, `"AD_SERVING"`, true}, {`4241`, `"AD_SERVING"`, true},
		{`99999`, `99999`, true}, {`92`, `"OTHER"`, true},
		{`" Ad Serving "`, `"AD_SERVING"`, false},
		{`"ad_serving"`, `"AD_SERVING"`, false},
		{`"NOT_A_FEE"`, `"NOT_A_FEE"`, false},
		{`" "`, `null`, false}, {`0`, `null`, false}, {`null`, `null`, false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			raw, err := plan.DecodeJSON([]byte(tc.raw))
			if err != nil {
				t.Fatal(err)
			}
			var stored ScalarValue
			if err := json.Unmarshal(raw, &stored); err != nil {
				t.Fatal(err)
			}
			if (stored.Integer != nil) != tc.numeric {
				t.Fatalf("input branch lost: %#v", stored)
			}
			if tc.numeric && tc.raw != "0" && json.Number(tc.raw).String() != mustScalarNumber(t, stored) {
				t.Fatal("numeric ID rewritten")
			}
			wire, err := plan.Encode(stored)
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(wire)
			if err != nil || string(got) != tc.output {
				t.Fatalf("wire = %s, %v; want %s", got, err, tc.output)
			}
		})
	}
	if plan.WireType(true) != "" || plan.WireType(false) != "" {
		t.Fatal("union advertised as single scalar")
	}
	for _, domain := range []*InputDomain{plan.InputDomain(), plan.OutputDomain()} {
		if domain == nil || !domain.Nullable || !domain.OpenIntegers || !domain.OpenStrings || domain.Minimum != "-9223372036854775808" || domain.Maximum != "9223372036854775807" {
			t.Fatalf("incorrect union domain: %#v", domain)
		}
	}
}

func TestScalarUnionClosedPolicy(t *testing.T) {
	plan, err := ParseEnum(`choice:"1:A" choice:"2:A" enumAccept:"scalar" enumStyle:"scalar"`, reflect.TypeOf(ScalarValue{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`3`, `"unknown"`, `"a"`, `" A "`} {
		if _, err := plan.DecodeJSON([]byte(raw)); err == nil {
			t.Errorf("closed policy accepted %s", raw)
		}
	}
	if d := plan.OutputDomain(); d.OpenIntegers || d.OpenStrings || len(d.Integers) != 0 || len(d.Strings) != 1 || d.Strings[0] != "A" {
		t.Fatalf("wrong closed output domain: %#v", d)
	}
	if _, err := plan.Encode("wrong storage"); err == nil {
		t.Fatal("accepted wrong storage")
	}
	s, i := "A", int64(1)
	if _, err := plan.Encode(ScalarValue{String: &s, Integer: &i}); err == nil {
		t.Fatal("accepted ambiguous storage")
	}
}

func TestScalarUnionReusedStorageAndPointerNull(t *testing.T) {
	plan, err := ParseEnum(feeUnionTag, reflect.TypeOf(ScalarValue{}))
	if err != nil {
		t.Fatal(err)
	}
	var stored ScalarValue
	for _, raw := range []string{`4237`, `"OTHER"`, `4241`, `null`, `"AD_SERVING"`, `0`} {
		converted, err := plan.DecodeJSON([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(converted, &stored); err != nil {
			t.Fatal(err)
		}
		if _, err := stored.Value(); err != nil {
			t.Fatalf("stale branch after %s: %v", raw, err)
		}
		if (raw == `null` || raw == `0`) && (stored.String != nil || stored.Integer != nil) {
			t.Fatalf("stale value after %s", raw)
		}
	}
	pointer, err := ParseEnum(feeUnionTag, reflect.TypeOf((*ScalarValue)(nil)))
	if err != nil {
		t.Fatal(err)
	}
	value := &stored
	converted, err := pointer.DecodeJSON([]byte(`null`))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(converted, &value); err != nil {
		t.Fatal(err)
	}
	if value != nil {
		t.Fatal("explicit null did not clear pointer")
	}
}

func mustScalarNumber(t *testing.T, v ScalarValue) string {
	t.Helper()
	raw, err := json.Marshal(*v.Integer)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestScalarUnionInvalidPlansAndInputs(t *testing.T) {
	for _, tag := range []reflect.StructTag{
		`choice:"1:A" enumAccept:"scalar" enumStyle:"literal"`,
		`choice:"1:A" choice:"1:B" enumAccept:"scalar" enumStyle:"scalar"`,
		`choice:"1:A" choice:"2:B" enumAlias:"1:b" enumCase:"fold" enumAccept:"scalar" enumStyle:"scalar"`,
		`choice:"9223372036854775808:A" enumAccept:"scalar" enumStyle:"scalar"`,
		`choice:"1:A" enumAccept:"scalar" enumStyle:"scalar" enumEmpty:"typo"`,
	} {
		if _, err := ParseEnum(tag, reflect.TypeOf(ScalarValue{})); err == nil {
			t.Errorf("accepted invalid plan %s", tag)
		}
	}
	plan, err := ParseEnum(feeUnionTag, reflect.TypeOf(ScalarValue{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`true`, `{}`, `[]`, `1.0`, `1e2`, `9223372036854775808`, `"A" 1`} {
		if _, err := plan.DecodeJSON([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}
