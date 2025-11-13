package session

import (
	"database/sql"
	"os"
	"testing"
)

var TestDB *sql.DB

func TestMain(m *testing.M) {
	// 初始化数据库连接
	TestDB, _ = sql.Open("sqlite3", "../orm.db")
	code := m.Run()
	_ = TestDB.Close()
	os.Exit(code)
}

func NewTestSession() *Session {
	return New(TestDB)
}

func TestSessionExec(t *testing.T) {
	s := NewTestSession()

	_, _ = s.Raw("DROP TABLE IF EXISTS User;").Exec()
	_, _ = s.Raw("CREATE TABLE User(Name text);").Exec()

	ret, _ := s.Raw("INSERT INTO User(`Name`) VALUES (?),(?);", "Jw", "Boyue").Exec()
	if count, err := ret.RowsAffected(); err != nil || count != 2 {
		t.Fatal("expect 2, but got", count)
	}
}

func TestSessionQueryRows(t *testing.T) {
	s := NewTestSession()
	// 清理并建表
	_, _ = s.Raw("DROP TABLE IF EXISTS User;").Exec()
	_, _ = s.Raw("CREATE TABLE User(Name text);").Exec()
	// 查询计数应为 0
	row := s.Raw("SELECT count(*) FROM User").QueryRow()
	var count int
	if err := row.Scan(&count); err != nil || count != 0 {
		t.Fatal("failed to query db", err)
	}
}
