package orm

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/faciam-dev/goquent/orm/internal/stringutil"
)

type fieldMeta struct {
	Col       string
	IndexPath []int
	PK        bool
	Readonly  bool
	OmitEmpty bool
}

type typeMeta struct {
	FieldsByName map[string]fieldMeta
	FieldsByNorm map[string]fieldMeta
	PKCols       []string
}

var metaCache sync.Map // map[reflect.Type]*typeMeta

func normalize(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "_", "")
}

func getTypeMeta(t reflect.Type) (*typeMeta, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("type %s is not struct", t)
	}
	if m, ok := metaCache.Load(t); ok {
		return m.(*typeMeta), nil
	}
	m := &typeMeta{
		FieldsByName: make(map[string]fieldMeta),
		FieldsByNorm: make(map[string]fieldMeta),
	}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		tag := sf.Tag.Get("db")
		if tag == "-" {
			continue
		}
		col := ""
		var opts []string
		if tag != "" {
			parts := strings.Split(tag, ",")
			col = parts[0]
			if len(parts) > 1 {
				opts = parts[1:]
			}
		}
		if col == "" {
			col = stringutil.ToSnake(sf.Name)
		}
		fm := fieldMeta{Col: col, IndexPath: sf.Index}
		for _, o := range opts {
			switch o {
			case "pk":
				fm.PK = true
				m.PKCols = append(m.PKCols, col)
			case "readonly":
				fm.Readonly = true
			case "omitempty":
				fm.OmitEmpty = true
			}
		}
		m.FieldsByName[col] = fm
		m.FieldsByNorm[normalize(col)] = fm
	}
	metaCache.Store(t, m)
	return m, nil
}

// ResetMetaCache clears cached reflection metadata. Intended for tests.
func ResetMetaCache() {
	metaCache = sync.Map{}
}
