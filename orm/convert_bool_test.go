package orm

import (
	"database/sql"
	"errors"
	"testing"
)

func TestScanBoolIntoPolicies(t *testing.T) {
	var b bool
	if err := scanBoolInto(&b, int64(0), BoolStrict); err != nil || b {
		t.Fatalf("strict 0: %v %v", b, err)
	}
	if err := scanBoolInto(&b, int64(1), BoolStrict); err != nil || !b {
		t.Fatalf("strict 1: %v %v", b, err)
	}
	if err := scanBoolInto(&b, int64(2), BoolStrict); err == nil {
		t.Fatalf("strict 2 expected error")
	}
	if err := scanBoolInto(&b, "t", BoolStrict); err != nil || !b {
		t.Fatalf("strict t: %v %v", b, err)
	}
	if err := scanBoolInto(&b, "yes", BoolStrict); err == nil {
		t.Fatalf("strict yes expected error")
	}

	if err := scanBoolInto(&b, "yes", BoolCompat); err != nil || !b {
		t.Fatalf("compat yes: %v %v", b, err)
	}

	if err := scanBoolInto(&b, "2", BoolLenient); err != nil || !b {
		t.Fatalf("lenient 2: %v %v", b, err)
	}
	if err := scanBoolInto(&b, "-3", BoolLenient); err != nil || !b {
		t.Fatalf("lenient -3: %v %v", b, err)
	}
}

func TestNilHandling(t *testing.T) {
	var b bool
	if err := scanBoolInto(&b, nil, BoolCompat); err == nil {
		t.Fatalf("nil into bool should error")
	}
	var nb sql.NullBool
	if err := scanNullBoolInto(&nb, nil, BoolCompat); err != nil || nb.Valid {
		t.Fatalf("nil into NullBool: %v %v", nb, err)
	}
	var pb *bool
	if err := scanPtrBoolInto(&pb, nil, BoolCompat); err != nil || pb != nil {
		t.Fatalf("nil into *bool: %v %v", pb, err)
	}
}

func TestParseBoolString(t *testing.T) {
	if v, err := parseBoolString("true", BoolStrict); err != nil || !v {
		t.Fatalf("parse true strict: %v %v", v, err)
	}
	if _, err := parseBoolString("yes", BoolStrict); err == nil {
		t.Fatalf("parse yes strict expected error")
	}
	if v, err := parseBoolString("on", BoolCompat); err != nil || !v {
		t.Fatalf("parse on compat: %v %v", v, err)
	}
	if v, err := parseBoolString("2", BoolLenient); err != nil || !v {
		t.Fatalf("parse 2 lenient: %v %v", v, err)
	}
}

func TestScanPtrBoolInto(t *testing.T) {
	var pb *bool
	if err := scanPtrBoolInto(&pb, int64(1), BoolCompat); err != nil {
		t.Fatalf("ptr bool 1: %v", err)
	}
	if pb == nil || !*pb {
		t.Fatalf("ptr bool value wrong: %v", pb)
	}
}

func TestErrBoolParseMessage(t *testing.T) {
	err := scanBoolInto(new(bool), int64(2), BoolStrict)
	var e ErrBoolParse
	if !errors.As(err, &e) {
		t.Fatalf("expected ErrBoolParse, got %v", err)
	}
	e.Column = "nullable"
	if msg := e.Error(); msg == "" {
		t.Fatalf("error message empty")
	}
}
