package clause

import (
	"fmt"
	"strings"
)

//实现各个子句的生成规则。

type generator func(values ...interface{}) (string, []interface{})

var generators map[Type]generator

func init() {
	generators = make(map[Type]generator)
	generators[INSERT] = _insert
	generators[VALUES] = _values
	generators[SELECT] = _select
	generators[WHERE] = _where
	generators[ORDERBY] = _orderBy
	generators[LIMIT] = _limit
	generators[UPDATE] = _update
	generators[DELETE] = _delete
	generators[COUNT] = _count
}

// genBindVars 生成形如 "?, ?, ?" 的占位符字符串
// genBindVars(3) -> "?, ?, ?"
func genBindVars(num int) string {
	var vars []string
	for i := 0; i < num; i++ {
		vars = append(vars, "?")
	}
	return strings.Join(vars, ", ")
}

// insert 生成 INSERT 子句
// 不包含VALUES部分 便于与批量行插入逻辑分离（多行 VALUES）
// Set(INSERT, "User", []string{"Name", "Age"})
// -> INSERT INTO User (Name, Age)
func _insert(values ...interface{}) (string, []interface{}) {
	// INSER INTO $tableName ($fields)
	tableName := values[0]
	fields := strings.Join(values[1].([]string), ", ")
	return fmt.Sprintf("INSERT INTO %s (%v)", tableName, fields), []interface{}{}
}

// 1. 第一次读取一行数据的长度决定 bindStr (? 占位符的个数与格式)。
// 2. 每行复用相同 bindStr，避免重复生成。
// 3. 将所有行的实际值顺序追加到 vars。
func _values(values ...interface{}) (string, []interface{}) {
	// VALUES ($v1), ($v2), ...
	var bindStr string
	var sql strings.Builder
	var vars []interface{}

	sql.WriteString("VALUES ")
	for i, value := range values {
		v := value.([]interface{})
		if bindStr == "" {
			bindStr = genBindVars(len(v))
		}
		sql.WriteString(fmt.Sprintf("(%v)", bindStr))
		if i+1 != len(values) {
			sql.WriteString(", ")
		}

		vars = append(vars, v...)
	}
	return sql.String(), vars
}

func _select(values ...interface{}) (string, []interface{}) {
	// SELECT $fields FROM $tableNames
	tableName := values[0]
	fields := strings.Join(values[1].([]string), ", ")
	return fmt.Sprintf("SELECT %s FROM %s", fields, tableName), []interface{}{}
}

func _limit(values ...interface{}) (string, []interface{}) {
	// LIMIT $num
	return "LIMIT ?", values
}

// _where 生成 WHERE 条件。第一个参数为条件表达式（含占位符），后续所有参数为绑定变量
// 示例：Set(WHERE, "Name = ?", "Tom") -> WHERE Name = ? + [Tom]
func _where(values ...interface{}) (string, []interface{}) {
	// WHERE $desc
	desc, vars := values[0].(string), values[1:]
	return fmt.Sprintf("WHERE %s", desc), vars
}

// 示例：Set(ORDERBY, "Age DESC, Name ASC")
//
//	-> SQL: "ORDER BY Age DESC, Name ASC", VARS: []
func _orderBy(values ...interface{}) (string, []interface{}) {
	// ORDER BY
	return fmt.Sprintf("ORDER BY %s", values[0]), []interface{}{}
}

// UPDATE 生成 UPDATE 子句
func _update(values ...interface{}) (string, []interface{}) {
	// UPDATE $tableName SET $field1 = ?, $field2 = ?
	tableName := values[0]
	m := values[1].(map[string]interface{})
	var keys []string
	var vars []interface{}
	for k, v := range m {
		keys = append(keys, k+" = ?")
		vars = append(vars, v)
	}
	return fmt.Sprintf("UPDATE %s SET %s", tableName, strings.Join(keys, ", ")), vars
}

// DELETE 生成 DELETE 子句
func _delete(values ...interface{}) (string, []interface{}) {
	// DELETE FROM $tableName
	return fmt.Sprintf("DELETE FROM %s", values[0]), []interface{}{}
}

// COUNT 生成 COUNT 子句
func _count(values ...interface{}) (string, []interface{}) {
	// SELECT COUNT(*) FROM $tableName
	return _select(values[0], []string{"COUNT(*)"})
}
