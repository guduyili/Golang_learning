package session

import (
	"orm/log"
	"testing"
)

type Account struct {
	ID       int `orm:"PRIMARY KEY"`
	Password string
}

func (account *Account) BeforeInsert(s *Session) error {
	log.Info("before insert hook", account)
	account.ID += 1145
	return nil
}

func (account *Account) BeforeQuery(s *Session) error {
	log.Info("before query hook", account)
	account.Password = "******"
	return nil
}

func TestSessionCallMethod(t *testing.T) {
	s := NewTestSession().Model(&Account{})
	_ = s.DropTable()
	_ = s.CreateTable()
	_, _ = s.Insert(&Account{1, "123456"}, &Account{2, "qwerty"})

	a := &Account{}
	err := s.First(a)
	if a.ID != 1146 || err != nil || a.Password != "******" {
		t.Fatal("failed to call before insert hook", a)
	}

}
