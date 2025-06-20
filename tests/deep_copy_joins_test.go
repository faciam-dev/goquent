package tests

import "testing"

func TestUpdateWithJoinDeepCopy(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	_, err := db.Table("users").Join("profiles", "users.id", "=", "profiles.user_id").Where("profiles.bio", "=", "go developer").Update(map[string]any{"age": 55})
	if err != nil {
		t.Fatalf("update with join: %v", err)
	}
	var row map[string]any
	if err := db.Table("users").Where("id", 1).FirstMap(&row); err != nil {
		t.Fatalf("select after update: %v", err)
	}
	if row["age"] != int64(55) {
		t.Errorf("expected age 55, got %v", row["age"])
	}
}

func TestUpdateWithLeftJoinDeepCopy(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	_, err := db.Table("users").LeftJoin("profiles", "users.id", "=", "profiles.user_id").Where("profiles.bio", "like", "%python%").Update(map[string]any{"age": 26})
	if err != nil {
		t.Fatalf("update with left join: %v", err)
	}
	var row map[string]any
	if err := db.Table("users").Where("id", 2).FirstMap(&row); err != nil {
		t.Fatalf("select after update: %v", err)
	}
	if row["age"] != int64(26) {
		t.Errorf("expected age 26, got %v", row["age"])
	}
}

func TestUpdateWithCrossJoinDeepCopy(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	_, err := db.Table("users").CrossJoin("profiles").Where("profiles.user_id", "users.id").Where("profiles.bio", "=", "go developer").Update(map[string]any{"age": 60})
	if err != nil {
		t.Fatalf("update with cross join: %v", err)
	}
	var row map[string]any
	if err := db.Table("users").Where("id", 1).FirstMap(&row); err != nil {
		t.Fatalf("select after cross join: %v", err)
	}
	if row["age"] != int64(60) {
		t.Errorf("expected age 60, got %v", row["age"])
	}
}

func TestDeleteWithJoinDeepCopy(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	_, err := db.Table("users").Join("profiles", "users.id", "=", "profiles.user_id").Where("profiles.bio", "=", "python developer").Delete()
	if err != nil {
		t.Fatalf("delete with join: %v", err)
	}
	var row map[string]any
	err = db.Table("users").Where("id", 2).FirstMap(&row)
	if err == nil {
		t.Fatalf("expected no rows, got %+v", row)
	}
}
