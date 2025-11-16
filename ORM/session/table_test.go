package session

import (
	"testing"
)

type User struct {
	Name string `orm:"PRIMARY KEY"`
	Age  int
}

func TestSessionCreateTable(t *testing.T) {
	s := NewTestSession().Model(&User{})
	_ = s.DropTable()
	_ = s.CreateTable()
	if !s.HasTable() {
		t.Fatal("Failed to create table User")
	}
}
