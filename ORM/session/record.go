package session

import (
	"orm/clause"
	"reflect"
)

// 1. 对每个待插入的对象调用 Model(value) 以解析/缓存其表结构（RefTable）。
// 2. 使用 clause 机制设置 INSERT 子句（表名+字段名），然后收集所有记录的字段值。
// 3. 设置 VALUES 子句，Build 得到最终 SQL 与绑定变量列表。
// 4. 通过 Raw(sql, vars...).Exec() 执行，返回影响行数。
// 注意：多个 value 会被组成多行 VALUES 插入，提升批量写入效率。
func (s *Session) Insert(values ...interface{}) (int64, error) {
	recordValues := make([]interface{}, 0)

	for _, value := range values {
		table := s.Model(value).RefTable()
		s.clause.Set(clause.INSERT, table.Name, table.FieldNames)
		recordValues = append(recordValues, table.RecordValues(value))
	}

	s.clause.Set(clause.VALUES, recordValues...)
	sql, vars := s.clause.Build(clause.INSERT, clause.VALUES)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// 反射要点：destSlice.Set(reflect.Append(destSlice, dest)) 将新元素追加回用户传入的切片。
func (s *Session) Find(values interface{}) error {
	destSlice := reflect.Indirect(reflect.ValueOf(values))

	//destSlice.Type().Elem()
	//获取切片的单个元素的类型 destType，
	destType := destSlice.Type().Elem()

	// 使用 reflect.New() 方法创建一个 destType 的实例，
	// 作为 Model() 的入参，映射出表结构 RefTable()。
	table := s.Model(reflect.New(destType).Elem().Interface()).RefTable()

	s.clause.Set(clause.SELECT, table.Name, table.FieldNames)
	sql, vars := s.clause.Build(clause.SELECT, clause.WHERE, clause.ORDERBY, clause.LIMIT)
	rows, err := s.Raw(sql, vars...).QueryRows()
	if err != nil {
		return err
	}

	// 执行查询后遍历 rows，每一行：
	// a. 新建一个元素实例 dest
	// b. 为每个字段构造其地址切片 values，用 rows.Scan 将数据填充到实例字段。
	// c. 将实例 append 到原始切片。
	for rows.Next() {
		//利用反射创建 destType 的实例 dest，
		// 将 dest 的所有字段平铺开，构造切片 values
		dest := reflect.New(destType).Elem()
		var values []interface{}
		for _, name := range table.FieldNames {
			values = append(values, dest.FieldByName(name).Addr().Interface())
		}
		//调用 rows.Scan() 将该行记录每一列的值依次赋值给 values 中的每一个字段。
		if err := rows.Scan(values...); err != nil {
			return err
		}
		//将 dest 添加到切片 destSlice 中。循环直到所有的记录都添加到切片 destSlice 中
		destSlice.Set(reflect.Append(destSlice, dest))
	}
	return rows.Close()
}
