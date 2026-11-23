package choice

import (
	"reflect"
	"testing"
)

func TestScalarUnionConstraintHasNoFiniteStorageDomain(t *testing.T) {
	tag := reflect.StructTag(`enumAccept:"scalar" enumStyle:"scalar" enumInteger:"passthrough" enumUnknownString:"preserve"`)
	field := reflect.TypeOf((*ScalarValue)(nil))
	enum, err := ParseEnum(tag, field)
	if err != nil || enum == nil {
		t.Fatalf("expected a native scalar representation, got %v, %v", enum, err)
	}
	constraint, err := Parse(tag, field)
	if err != nil || constraint != nil {
		t.Fatalf("open scalar union has no finite storage constraint: %v, %v", constraint, err)
	}
}
