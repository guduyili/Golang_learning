package orm

import (
	"database/sql"
	"orm/dialect"
	"orm/log"
	"orm/session"
)

type Engine struct {
	db      *sql.DB
	dialect dialect.Dialect
}

func NewEngine(driver, source string) (e *Engine, err error) {
	db, err := sql.Open(driver, source)
	if err != nil {
		log.Error(err)
		return
	}

	if err = db.Ping(); err != nil {
		log.Error(err)
		return
	}
	// 获取对应的方言
	dial, ok := dialect.GetDialect(driver)
	if !ok {
		log.Error("dialect % Not Found", driver)
		return
	}

	e = &Engine{db: db, dialect: dial}
	log.Info("Connect database success")
	return
}

func (e *Engine) Close() {
	if e == nil || e.db == nil {
		log.Info("Close skipped: engine or db is nil")
		return
	}
	if err := e.db.Close(); err != nil {
		log.Error("Failed to close database")
	}
	log.Info("Close database success")
}

func (engine *Engine) NewSession() *session.Session {
	if engine == nil || engine.db == nil {
		// 让错误更早、更明确
		log.Error("NewSession called before NewEngine succeeded: db is nil")
		panic("orm: engine.db is nil; ensure NewEngine succeeded and error was checked")
	}
	return session.New(engine.db, engine.dialect)
}
