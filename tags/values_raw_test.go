package tags

import (
	"errors"
	"reflect"
	"testing"
)

var _ func(Values, func(string, string) error) error = Values.MatchPairs

func TestValuesRawPairsPreserveQuotes(t *testing.T) {
	input := Values("table=`project.dataset.table`,refTable=[project:dataset.table],db=\"my.schema\",message='with,comma',label=\"text\",one='x',values={1,2},unique")
	raw, decoded := map[string]string{}, map[string]string{}
	if err := input.MatchRawPairs(func(k, v string) error { raw[k] = v; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := input.MatchPairs(func(k, v string) error { decoded[k] = v; return nil }); err != nil {
		t.Fatal(err)
	}
	wantRaw := map[string]string{"table": "`project.dataset.table`", "refTable": "[project:dataset.table]", "db": `"my.schema"`, "message": "'with,comma'", "label": `"text"`, "one": "'x'", "values": "{1,2}", "unique": ""}
	wantDecoded := map[string]string{"table": "project.dataset.table", "refTable": "[project:dataset.table]", "db": "my.schema", "message": "'with,comma'", "label": "text", "one": "x", "values": "{1,2}", "unique": ""}
	if !reflect.DeepEqual(raw, wantRaw) || !reflect.DeepEqual(decoded, wantDecoded) {
		t.Fatalf("raw=%#v decoded=%#v", raw, decoded)
	}
	failure := errors.New("stop")
	for _, match := range []func(func(string, string) error) error{input.MatchPairs, input.MatchRawPairs} {
		calls := 0
		if err := match(func(string, string) error { calls++; return failure }); !errors.Is(err, failure) || calls != 1 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	}
}
