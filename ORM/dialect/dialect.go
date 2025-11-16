package dialect

import "reflect"

var dialectsMap = map[string]Dialect{}

type Dialect interface {
	// 用于将 Go 语言的类型转换为该数据库的数据类型。
	DataTypeOf(typ reflect.Value) string

	// 返回某个表是否存在的 SQL 语句，参数是表名(table)。
	TableExistSQL(tableName string) (string, []interface{})
}

func RegisterDialect(name string, dialect Dialect) {
	dialectsMap[name] = dialect
}

func GetDialect(name string) (dialect Dialect, ok bool) {
	dialect, ok = dialectsMap[name]
	return
}
