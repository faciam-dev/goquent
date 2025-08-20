package orm

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type BoolScanPolicy int

const (
	BoolStrict BoolScanPolicy = iota
	BoolCompat
	BoolLenient
)

func (p BoolScanPolicy) String() string {
	switch p {
	case BoolStrict:
		return "Strict"
	case BoolCompat:
		return "Compat"
	case BoolLenient:
		return "Lenient"
	default:
		return fmt.Sprintf("BoolScanPolicy(%d)", int(p))
	}
}

type ScanOptions struct {
	BoolPolicy BoolScanPolicy
}

type ErrBoolParse struct {
	Column string
	Src    any
	Policy BoolScanPolicy
}

func (e ErrBoolParse) Error() string {
	col := ""
	if e.Column != "" {
		col = fmt.Sprintf("column %q: ", e.Column)
	}
	return fmt.Sprintf("%scannot parse bool from %T(%v) under %s policy", col, e.Src, e.Src, e.Policy)
}

func scanBoolInto(dst *bool, src any, pol BoolScanPolicy) error {
	switch v := src.(type) {
	case bool:
		*dst = v
		return nil
	case int64:
		switch pol {
		case BoolLenient:
			*dst = v != 0
			return nil
		case BoolStrict, BoolCompat:
			if v == 0 {
				*dst = false
				return nil
			}
			if v == 1 {
				*dst = true
				return nil
			}
			return ErrBoolParse{Src: v, Policy: pol}
		default:
			if v == 0 {
				*dst = false
				return nil
			}
			if v == 1 {
				*dst = true
				return nil
			}
			return ErrBoolParse{Src: v, Policy: pol}
		}
	case string:
		b, err := parseBoolString(v, pol)
		if err != nil {
			return err
		}
		*dst = b
		return nil
	case []byte:
		b, err := parseBoolString(string(v), pol)
		if err != nil {
			return err
		}
		*dst = b
		return nil
	case nil:
		// plain bool cannot represent NULL; treat as parse error
		return ErrBoolParse{Src: nil, Policy: pol}
	default:
		return ErrBoolParse{Src: v, Policy: pol}
	}
}

func scanNullBoolInto(dst *sql.NullBool, src any, pol BoolScanPolicy) error {
	if src == nil {
		dst.Bool = false
		dst.Valid = false
		return nil
	}
	var b bool
	if err := scanBoolInto(&b, src, pol); err != nil {
		return err
	}
	dst.Bool = b
	dst.Valid = true
	return nil
}

func scanPtrBoolInto(dst **bool, src any, pol BoolScanPolicy) error {
	if src == nil {
		*dst = nil
		return nil
	}
	var b bool
	if err := scanBoolInto(&b, src, pol); err != nil {
		return err
	}
	if *dst == nil {
		*dst = new(bool)
	}
	**dst = b
	return nil
}

func parseBoolString(s string, pol BoolScanPolicy) (bool, error) {
	x := strings.TrimSpace(strings.ToLower(s))
	switch x {
	case "true", "t", "1":
		return true, nil
	case "false", "f", "0":
		return false, nil
	case "yes", "y", "on":
		if pol == BoolStrict {
			return false, ErrBoolParse{Src: s, Policy: pol}
		}
		return true, nil
	case "no", "n", "off":
		if pol == BoolStrict {
			return false, ErrBoolParse{Src: s, Policy: pol}
		}
		return false, nil
	}
	if pol == BoolLenient {
		if n, err := strconv.ParseInt(x, 10, 64); err == nil {
			return n != 0, nil
		}
	}
	return false, ErrBoolParse{Src: s, Policy: pol}
}

// decoder helpers used in meta cache

func decodeBool(dst reflect.Value, src any, pol BoolScanPolicy) error {
	return scanBoolInto(dst.Addr().Interface().(*bool), src, pol)
}

func decodeNullBool(dst reflect.Value, src any, pol BoolScanPolicy) error {
	return scanNullBoolInto(dst.Addr().Interface().(*sql.NullBool), src, pol)
}

func decodePtrBool(dst reflect.Value, src any, pol BoolScanPolicy) error {
	return scanPtrBoolInto(dst.Addr().Interface().(**bool), src, pol)
}
