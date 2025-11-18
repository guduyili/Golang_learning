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

func TestSession_Find(t *testing.T) {
	s := testRecordInit(t)
	var users []User
	if err := s.Find(&users); err != nil || len(users) != 2 {
		t.Fatal("failed to find records:", err)
	}
}

func TestSession_Update(t *testing.T) {
	s := testRecordInit(t)
	affected, _ := s.Where("Name = ?", "Tom").Update("Age", 30)

	u := &User{}
	_ = s.OrderBy("Age DESC").First(u)
	if affected != 1 || u.Age != 30 {
		t.Fatal("failed to update record")
	}
}

func TestSession_Limit(t *testing.T) {
	s := testRecordInit(t)
	var users []User
	err := s.Limit(1).Find(&users)
	if err != nil || len(users) != 1 {
		t.Fatal("failed to limit records:", err)
	}
}

func TestSession_DeleteAndCount(t *testing.T) {
	s := testRecordInit(t)
	affected, _ := s.Where("Name = ?", "Tom").Delete()

	count, _ := s.Count()
	if affected != 1 || count != 1 {
		t.Fatal("failed to delete record or count records")
	}
}
