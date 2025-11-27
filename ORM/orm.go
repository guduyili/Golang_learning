package orm

import (
	"database/sql"
	"fmt"
	"orm/dialect"
	"orm/log"
	"orm/session"
	"strings"
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

type TxFunc func(*session.Session) (interface{}, error)

// Transaction 在事务中执行回调 f：
//   - 开始事务
//     遇到 return f(s)：先求值 f(s)，得到 (res, e)。
//     将结果赋给命名返回值 result、err。
//     进入所有已登记的 defer（逆序执行）。
//     defer 结束后，使用命名返回值离开函数。
func (engine *Engine) Transaction(f TxFunc) (result interface{}, err error) {
	s := engine.NewSession()

	if err := s.Begin(); err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = s.Rollback()
			panic(p) // - 若发生 panic：先回滚，再将 panic 重新抛出；
		} else if err != nil {
			_ = s.Rollback() // - 若发生错误：回滚事务；
		} else {
			err = s.Commit() // - 否则：提交事务。
		}
	}()
	return f(s)
}

// difference 返回 a-b
// 比较两个字符串切片，返回只在 a 中出现的元素
func difference(a []string, b []string) (diff []string) {
	m := make(map[string]bool)
	for _, v := range b {
		m[v] = true
	}
	for _, v := range a {
		if _, ok := m[v]; !ok {
			diff = append(diff, v)
		}
	}
	return
}

// Migrate 自动迁移表结构
// - 若表不存在 直接创建表
// — 若表存在：
//  1. `rows.Columns` 根据返回的第一行元数据，查询现有列名
//  2. 对比模型字段与现有列，计算需要的新增/删除的列
//  3. 对新增的列执行 ALTER TABLE ADD COLUMN
//  4. 若需要删除列：通过创建临时表 + 复制需要的列 + 删除原表 + 重命名临时表实现“删列”。
func (engine *Engine) Migrate(value interface{}) error {
	_, err := engine.Transaction(func(s *session.Session) (result interface{}, err error) {
		if !s.Model(value).HasTable() {
			log.Info("table %s doesn't exist", s.RefTable().Name)
			return nil, s.CreateTable()
		}

		table := s.RefTable()
		rows, _ := s.Raw(fmt.Sprintf("SELECT * FROM %s LIMIT 1;", table.Name)).QueryRows()
		columns, _ := rows.Columns()
		_ = rows.Close()

		addCols := difference(table.FieldNames, columns)
		delCols := difference(columns, table.FieldNames)
		log.Info("added cols %v, deleted cols %v", addCols, delCols)

		for _, col := range addCols {
			f := table.GetField(col)
			sqlStr := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table.Name, f.Name, f.Type)
			if _, err = s.Raw(sqlStr).Exec(); err != nil {
				return
			}
		}

		if len(delCols) == 0 {
			return
		}

		tmp := "tmp_" + table.Name
		fieldStr := strings.Join(table.FieldNames, ", ")
		//  创建临时表
		s.Raw(fmt.Sprintf("CREATE TABLE %s AS SELECT %s FROM %s;", tmp, fieldStr, table.Name))
		s.Raw(fmt.Sprintf("DROP TABLE %s;", table.Name))
		s.Raw(fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", tmp, table.Name))
		_, err = s.Exec()
		return
	})
	return err
}
