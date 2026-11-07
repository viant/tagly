package format

import (
	"reflect"
	"testing"
)

func TestInputNamePolicies(t *testing.T) {
	policy, err := Parse(`format:"inputNames=frequencyCap,inputNamesPriority=canonical,inputNamesKinds=object|null,inputNamesCase=fold"`)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewInputNameMatcher([]InputName{{Name: "frequencyCapping", Aliases: policy.InputNames}}, policy)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		members []InputMember
		want    []bool
	}{
		{[]InputMember{{"frequencyCap", "object"}}, []bool{true}},
		{[]InputMember{{"FREQUENCYCAP", "null"}}, []bool{true}},
		{[]InputMember{{"frequencyCap", "number"}}, []bool{false}},
		{[]InputMember{{"frequencyCapping", "object"}, {"frequencyCap", "object"}}, []bool{true, false}},
		{[]InputMember{{"frequencyCap", "object"}, {"frequencyCapping", "null"}}, []bool{false, true}},
		{[]InputMember{{"frequencyCapping", "number"}, {"frequencyCap", "object"}}, []bool{true, false}},
		{[]InputMember{{"frequencyCap", "object"}, {"frequencyCap", "null"}}, []bool{true, true}},
	} {
		if got := m.Select(tc.members); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%v => %v want%v", tc.members, got, tc.want)
		}
	}
	flight, err := Parse(`format:"inputNames=flightConstraints,inputNamesCase=fold,inputNamesExclude=FlightConstraints"`)
	if err != nil {
		t.Fatal(err)
	}
	fm, err := NewInputNameMatcher([]InputName{{Name: "campaignFlightCaps", Aliases: flight.InputNames}}, flight)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"flightConstraints", "FLIGHTCONSTRAINTS"} {
		if fm.Match(name) != 0 {
			t.Fatal(name)
		}
	}
	if fm.Match("FlightConstraints") != -1 || fm.Select([]InputMember{{"FlightConstraints", "array"}})[0] {
		t.Fatal("excluded alias admitted")
	}
}
func TestInputNamePolicyDeclarationErrors(t *testing.T) {
	for _, tag := range []reflect.StructTag{`format:"inputNamesPriority=canonical"`, `format:"inputNames=x,inputNamesCase=loose"`, `format:"inputNames=x,inputNamesKinds=wrong"`, `format:"inputNames=x,inputNamesPriority=first"`, `format:"inputNames=x,inputNamesExclude="`} {
		if _, err := Parse(tag); err == nil {
			t.Errorf("accepted %s", tag)
		}
	}
	p, _ := Parse(`format:"inputNames=alias,inputNamesCase=fold,inputNamesExclude=value"`)
	if _, err := NewInputNameMatcher([]InputName{{Name: "value", Aliases: "alias"}}, p); err == nil {
		t.Error("canonical exclusion accepted")
	}
	a, _ := Parse(`format:"inputNames=alias,inputNamesCase=fold"`)
	if _, err := NewInputNameMatcher([]InputName{{Name: "first", Aliases: "alias"}, {Name: "second", Aliases: "ALIAS"}}, a, nil); err == nil {
		t.Error("ambiguous fold aliases accepted")
	}
}

func TestInputNameAliasPriority(t *testing.T) {
	p, err := Parse(`format:"inputNames=CampaignFlightId|Ignored,inputNamesPriority=alias,inputNamesCase=fold,inputNamesKinds=number|null,inputNamesExclude=Ignored"`)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewInputNameMatcher([]InputName{{Name: "flightId", Aliases: p.InputNames}}, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		members []InputMember
		want    []bool
	}{
		{[]InputMember{{"CampaignFlightId", "number"}, {"flightId", "string"}}, []bool{true, false}},
		{[]InputMember{{"FLIGHTID", "number"}, {"campaignflightid", "null"}}, []bool{false, true}},
		{[]InputMember{{"flightId", "number"}, {"CampaignFlightId", "string"}}, []bool{true, false}},
		{[]InputMember{{"Ignored", "number"}, {"flightId", "number"}}, []bool{false, true}},
		{[]InputMember{{"CampaignFlightId", "number"}, {"flightId", "number"}, {"CAMPAIGNFLIGHTID", "number"}}, []bool{true, false, true}},
		{[]InputMember{{"other", "number"}}, []bool{true}},
	} {
		if got := m.Select(tc.members); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%v got%v want%v", tc.members, got, tc.want)
		}
	}
	same := &Tag{InputNames: "FLIGHTID", InputNamesPriority: "alias"}
	m, err = NewInputNameMatcher([]InputName{{Name: "flightId", Aliases: same.InputNames}}, same)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Select([]InputMember{{"FLIGHTID", "number"}, {"flightId", "number"}}); !reflect.DeepEqual(got, []bool{true, true}) {
		t.Fatal(got)
	}
}
