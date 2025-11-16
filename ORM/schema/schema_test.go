package schema

import (
	"orm/dialect"
	"testing"
)

type User struct {
	Name string `orm:"PRIMARY KEY"`
	Age  int
}

var TestDial, _ = dialect.GetDialect("sqlite")

func TestParse(t *testing.T) {
	schema := Parse(&User{}, TestDial)
	if schema.Name != "User" || len(schema.Fields) != 2 {
		t.Fatal("falied to parse User struct")
	}

	if schema.GetField("Name").Tag != "PRIMARY KEY" {
		t.Fatal("failed to parse tag for Name field")
	}
}
