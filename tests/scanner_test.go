package tests

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock" // TODO: consider removing external mock library

	"github.com/faciam-dev/goquent/orm/scanner"
)

func TestMapConvertsBytes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "alice")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r, err := db.Query("SELECT id, name FROM users")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer r.Close()

	m, err := scanner.Map(r)
	if err != nil {
		t.Fatalf("scan map: %v", err)
	}
	if m["name"] != "alice" {
		t.Errorf("expected alice, got %v", m["name"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMapsConvertsBytes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "alice").
		AddRow(2, "bob")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r, err := db.Query("SELECT id, name FROM users")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer r.Close()

	ms, err := scanner.Maps(r)
	if err != nil {
		t.Fatalf("scan maps: %v", err)
	}
	if len(ms) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(ms))
	}
	if ms[0]["name"] != "alice" || ms[1]["name"] != "bob" {
		t.Errorf("unexpected rows: %v", ms)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
