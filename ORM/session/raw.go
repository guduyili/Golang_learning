package session

import (
	"database/sql"
	"orm/log"
	"strings"
)

type Session struct {
	db      *sql.DB
	sql     strings.Builder
	sqlVars []interface{}
}

// New 创建会话实例（轻量、无连接池，复用底层 *sql.DB）
func New(db *sql.DB) *Session {
	return &Session{
		db: db,
	}
}

// Clear 重置内部 SQL 构建状态，避免跨次调用串台
func (s *Session) Clear() {
	s.sql.Reset()
	s.sqlVars = nil
}

// DB 返回底层数据库连接
func (s *Session) DB() *sql.DB {
	return s.db
}

// Exec 执行写类语句，如 INSERT/UPDATE/DELETE/DDL
// 说明：使用 defer Clear() 确保每次调用后自动清空构建器
func (s *Session) Exec() (result sql.Result, err error) {
	defer s.Clear()
	log.Info(s.sql.String(), s.sqlVars)
	if result, err = s.DB().Exec(s.sql.String(), s.sqlVars...); err != nil {
		log.Error(err)
	}
	return
}

// QueryRow 查询单行记录（如 SELECT ... WHERE id=?）
func (s *Session) QueryRow() (row *sql.Row) {
	defer s.Clear()
	log.Info(s.sql.String(), s.sqlVars)
	return s.DB().QueryRow(s.sql.String(), s.sqlVars...)
}

// QueryRows 查询多行记录（如 SELECT ...）
func (s *Session) QueryRows() (rows *sql.Rows, err error) {
	defer s.Clear()
	log.Info(s.sql.String(), s.sqlVars)
	if rows, err = s.DB().Query(s.sql.String(), s.sqlVars...); err != nil {
		log.Error(err)
	}
	return
}

// Raw 设置原生 SQL 及其参数，供后续执行使用
func (s *Session) Raw(sql string, values ...interface{}) *Session {
	s.sql.WriteString(sql)
	s.sql.WriteString(" ")
	s.sqlVars = append(s.sqlVars, values...)
	return s
}
