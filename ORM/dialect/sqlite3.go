package dialect

import (
	"fmt"
	"reflect"
	"time"
)

type sqlite struct{}

var _ Dialect = (*sqlite)(nil)

func init() {
	// 在包初始化阶段注册 sqlite3 方言实现，供后续按名称检索
	// RegisterDialect("sqlite3", &sqlite3{})
	// 同时注册 "sqlite" 以兼容 modernc.org/sqlite 驱动的名称
	RegisterDialect("sqlite", &sqlite{})
}

// DataTypeOf 将 Go 的反射类型映射为 SQLite 支持的字段类型
// 说明：
// - 整型细分映射为 integer / bigint
// - 浮点统一为 real
// - 字符串为 text
// - 数组、切片为 blob
// - time.Time 特判为 datetime
// 不支持的类型直接 panic，提醒开发者自行处理。
func (s *sqlite) DataTypeOf(typ reflect.Value) string {
	switch typ.Kind() {
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "integer"
	case reflect.Int64, reflect.Uint64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "real"
	case reflect.String:
		return "text"
	case reflect.Array, reflect.Slice:
		return "blob"
	case reflect.Struct:
		if _, ok := typ.Interface().(time.Time); ok {
			return "datetime"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s)", typ.Type().Name(), typ.Kind()))
}

// TableExistSQL 返回用于判断指定表是否存在的查询语句及参数
// SQLite 通过查询 sqlite_master 系统表来判断
func (s *sqlite) TableExistSQL(tableName string) (string, []interface{}) {
	args := []interface{}{tableName}
	return "SELECT name FROM sqlite_master WHERE type='table' and name = ?", args
}
