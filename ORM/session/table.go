package session

import (
	"fmt"
	"orm/log"
	"orm/schema"
	"reflect"
	"strings"
)

// Model 绑定当前 Session 对应的模型结构，通过反射解析结构字段生成 Schema。
// 若之前未设置 refTable，或传入的结构类型与缓存的不同，则重新 Parse。
// 这样可以避免在同一个 Session 中持有过期的字段描述。
func (s *Session) Model(value interface{}) *Session {
	if s.refTable == nil || reflect.TypeOf(value) != reflect.TypeOf(s.refTable.Model) {
		s.refTable = schema.Parse(value, s.dialect)
	}
	return s
}

func (s *Session) RefTable() *schema.Schema {
	if s.refTable == nil {
		log.Error("Model is not set")
	}

	return s.refTable
}

func (s *Session) CreateTable() error {
	table := s.RefTable()
	var columns []string

	for _, field := range table.Fields {
		columns = append(columns, fmt.Sprintf("%s %s %s", field.Name, field.Type, field.Tag))
	}

	desc := strings.Join(columns, ",")
	// CREATE TABLE User (Name text PRIMARY KEY,Age integer );
	_, err := s.Raw(fmt.Sprintf("CREATE TABLE %s (%s);", table.Name, desc)).Exec()
	return err
}

func (s *Session) DropTable() error {
	_, err := s.Raw(fmt.Sprintf("DROP TABLE IF EXISTS %s;", s.RefTable().Name)).Exec()
	return err
}

// HasTable 判断表是否存在。调用 Dialect.TableExistSQL 获取不同数据库的兼容查询。
// 通过扫描返回的 name 字段与当前 Schema 名称是否一致来决定
func (s *Session) HasTable() bool {
	sql, values := s.dialect.TableExistSQL(s.RefTable().Name)
	row := s.Raw(sql, values...).QueryRow()
	var tmp string
	_ = row.Scan(&tmp)
	return tmp == s.RefTable().Name
}
