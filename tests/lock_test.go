package tests

import (
	"strings"
	"testing"
)

func TestSharedLockBuild(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	sqlStr, err := db.Table("users").Where("id", 1).SharedLock().RawSQL()
	if err != nil {
		t.Fatalf("raw sql: %v", err)
	}
	if !strings.Contains(sqlStr, "LOCK IN SHARE MODE") {
		t.Errorf("expected LOCK IN SHARE MODE, got %s", sqlStr)
	}
}

func TestLockForUpdateBuild(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	sqlStr, err := db.Table("users").Where("id", 1).LockForUpdate().RawSQL()
	if err != nil {
		t.Fatalf("raw sql: %v", err)
	}
	if !strings.Contains(sqlStr, "FOR UPDATE") {
		t.Errorf("expected FOR UPDATE, got %s", sqlStr)
	}
}
