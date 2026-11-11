package jsonx

import (
	"encoding/json"
	"reflect"
	"testing"
)

type objectRecord struct {
	ID *int `json:"id"`
}
type objectRecords []objectRecord
type jsonOwnedRecord struct{}

func (*jsonOwnedRecord) UnmarshalJSON([]byte) error { return nil }

type jsonOutputRecord struct{}

func (jsonOutputRecord) MarshalJSON() ([]byte, error) { return []byte(`1`), nil }

type textOwnedRecord struct{}

func (*textOwnedRecord) UnmarshalText([]byte) error { return nil }

type textOutputRecord struct{}

func (textOutputRecord) MarshalText() ([]byte, error) { return []byte(`1`), nil }

type ownedRecords []objectRecord

func (*ownedRecords) UnmarshalJSON([]byte) error { return nil }

func TestObjectToSliceAdmission(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf([]objectRecord{}), reflect.TypeOf(objectRecords{})} {
		p, err := Compile(`jsonx:"objectToSlice"`, typ)
		if err != nil {
			t.Fatal(err)
		}
		if !p.Active() {
			t.Fatal("objectToSlice silently ignored")
		}
		d := p.InputDomain()
		object := reflect.ValueOf(d).FieldByName("Object")
		if !object.IsValid() || !object.Bool() || !d.Array || !d.Null || d.Boolean || d.String || d.Number || d.Integer {
			t.Fatalf("wrong domain: %+v", d)
		}
	}
	for _, typ := range []reflect.Type{nil, reflect.TypeOf(objectRecord{}), reflect.TypeOf(&objectRecords{}), reflect.TypeOf([]*objectRecord{}), reflect.TypeOf([1]objectRecord{}), reflect.TypeOf([]int{}), reflect.TypeOf([]map[string]int{}), reflect.TypeOf([]jsonOwnedRecord{}), reflect.TypeOf([]jsonOutputRecord{}), reflect.TypeOf([]textOwnedRecord{}), reflect.TypeOf([]textOutputRecord{}), reflect.TypeOf(ownedRecords{})} {
		if _, err := Compile(`jsonx:"objectToSlice"`, typ); err == nil {
			t.Fatalf("accepted unsupported destination %v", typ)
		}
	}
	if _, err := Compile(`jsonx:"objectToSlice,commaStringToSlice"`, reflect.TypeOf([]objectRecord{})); err == nil {
		t.Fatal("accepted conflicting conversion")
	}
}

func TestObjectToSliceLexicalAndTypedBoundary(t *testing.T) {
	p, err := Compile(`jsonx:"objectToSlice"`, reflect.TypeOf([]objectRecord{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		raw, want            string
		count                int
		nilSlice, typedError bool
	}{
		{`null`, `null`, 0, true, false},
		{` [] `, ` [] `, 0, false, false},
		{`{}`, `[{}]`, 1, false, false},
		{` {"id":1} `, `[ {"id":1} ]`, 1, false, false},
		{`[{"id":1}]`, `[{"id":1}]`, 1, false, false},
		{`[{"id":1},{"id":2}]`, `[{"id":1},{"id":2}]`, 2, false, false},
		{`[null]`, `[null]`, 1, false, false},
		{`{"id":null}`, `[{"id":null}]`, 1, false, false},
		{`{"id":0,"id":2,"unknown":900719925474099312345}`, `[{"id":0,"id":2,"unknown":900719925474099312345}]`, 1, false, false},
		{`[1]`, `[1]`, 0, false, true},
		{`{"id":1.5}`, `[{"id":1.5}]`, 0, false, true},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			raw := []byte(tc.raw)
			got, omit, err := p.Apply(raw)
			if err != nil || omit || string(got) != tc.want {
				t.Fatalf("got %s omit %v err %v", got, omit, err)
			}
			if string(raw) != tc.raw {
				t.Fatal("mutated source bytes")
			}
			var records []objectRecord
			err = json.Unmarshal(got, &records)
			if tc.typedError {
				if err == nil {
					t.Fatal("typed decoder accepted invalid member")
				}
				return
			}
			if err != nil || len(records) != tc.count || (records == nil) != tc.nilSlice {
				t.Fatalf("typed result %+v err %v", records, err)
			}
		})
	}
	for _, raw := range []string{``, ` `, `true`, `1`, `"record"`, `{`, `[`, `{} {}`, `[] false`, `{"id":}`, `[{},]`} {
		if _, _, err := p.Apply([]byte(raw)); err == nil {
			t.Fatalf("accepted invalid root %q", raw)
		}
	}
	omitPolicy, err := Compile(`jsonx:"objectToSlice,null=omit"`, reflect.TypeOf([]objectRecord{}))
	if err != nil {
		t.Fatal(err)
	}
	if _, omit, err := omitPolicy.Apply([]byte(`null`)); err != nil || !omit {
		t.Fatalf("null omission %v %v", omit, err)
	}
	if got, omit, err := omitPolicy.Apply([]byte(`[]`)); err != nil || omit || string(got) != `[]` {
		t.Fatalf("empty array %s %v %v", got, omit, err)
	}
	plain, err := Compile(``, reflect.TypeOf([]objectRecord{}))
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := plain.Apply([]byte(`{}`))
	if err != nil || string(got) != `{}` {
		t.Fatal("changed untagged input")
	}
	var records []objectRecord
	if json.Unmarshal(got, &records) == nil {
		t.Fatal("untagged object admitted as slice")
	}
}
