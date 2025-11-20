package session

import "orm/log"

// Begin 开启一个事务（对应 database/sql 的 *sql.Tx.Begin）。
// 1) 调用后 s.tx 将被设置为有效的 *sql.Tx；后续 Exec/Query 将通过 s.DB() 自动走 tx。
func (s *Session) Begin() (err error) {
	log.Info("transaction begin")
	if s.tx, err = s.db.Begin(); err != nil {
		log.Error(err)
		return
	}
	return
}

// Commit 提交当前事务（对应 database/sql 的 *sql.Tx.Commit）。
func (s *Session) Commit() (err error) {
	log.Info("tansaction commit")
	if err = s.tx.Commit(); err != nil {
		log.Error(err)
	}
	return
}

func (s *Session) Rollback() (err error) {
	log.Info("transaction rollback")
	if err = s.tx.Rollback(); err != nil {
		log.Error(err)
	}
	return
}
