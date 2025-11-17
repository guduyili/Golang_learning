package session

import "testing"

var (
	user1 = &User{"Tom", 18}
	user2 = &User{"Sam", 25}
	user3 = &User{"jw", 114}
)

func testRecordInit(t *testing.T) *Session {
	t.Helper()
	s := NewTestSession().Model(&User{})

	err1 := s.DropTable()
	err2 := s.CreateTable()

	_, err3 := s.Insert(user1, user2)

	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatal("failed init")
	}
	return s
}

func TestSession_Insert(t *testing.T) {
	s := testRecordInit(t)
	affected, err := s.Insert(user3)
	if err != nil || affected != 1 {
		t.Fatal("failed to insert record:", err)
	}
}

func TestSessionFind(t *testing.T) {
	s := testRecordInit(t)
	var users []User
	if err := s.Find(&users); err != nil || len(users) != 2 {
		t.Fatal("failed to find records:", err)
	}
}
