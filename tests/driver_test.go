package tests

import (
	"testing"

	"github.com/faciam-dev/goquent/orm/driver"
)

func TestQuoteIdentEscapesBackticks(t *testing.T) {
	d := driver.MySQLDialect{}
	got := d.QuoteIdent("te`st")
	if got != "`te``st`" {
		t.Errorf("unexpected quote result: %s", got)
	}
}
