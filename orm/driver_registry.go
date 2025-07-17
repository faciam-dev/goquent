package orm

import (
	sqldriver "database/sql/driver"
	"sync"
)

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]sqldriver.Driver)
)

// RegisterDriver registers a database driver.
func RegisterDriver(name string, d sqldriver.Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	drivers[name] = d
}

// GetDriver retrieves a registered driver.
func GetDriver(name string) (sqldriver.Driver, bool) {
	driversMu.RLock()
	defer driversMu.RUnlock()
	d, ok := drivers[name]
	return d, ok
}
