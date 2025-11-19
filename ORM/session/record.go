package session

import (
	"errors"
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
		s.CallMethod(BeforeInsert, value)
		s.clause.Set(clause.INSERT, table.Name, table.FieldNames)
		recordValues = append(recordValues, table.RecordValues(value))
	}

	s.clause.Set(clause.VALUES, recordValues...)
	sql, vars := s.clause.Build(clause.INSERT, clause.VALUES)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}

	s.CallMethod(AfterInsert, nil)
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
	// b. 为每个字段构造其地址切片 scanTargets，用 rows.Scan 将数据填充到实例字段。
	// c. 将实例 append 到原始切片。
	for rows.Next() {
		//利用反射创建 destType 的实例 dest，
		// 将 dest 的所有字段平铺开，构造切片 scanTargets
		dest := reflect.New(destType).Elem()
		var scanTargets []interface{}
		for _, name := range table.FieldNames {
			scanTargets = append(scanTargets, dest.FieldByName(name).Addr().Interface())
		}
		//调用 rows.Scan() 将该行记录每一列的值依次赋值给 values 中的每一个字段。
		if err := rows.Scan(scanTargets...); err != nil {
			return err
		}
		s.CallMethod(BeforeQuery, dest.Addr().Interface())
		s.CallMethod(AfterQuery, dest.Addr().Interface())
		//将 dest 添加到切片 destSlice 中。循环直到所有的记录都添加到切片 destSlice 中
		destSlice.Set(reflect.Append(destSlice, dest))
	}
	return rows.Close()
}

// Update 根据当前 WHERE 条件更新字段：
// 支持两种调用方式：
// 1) Map：Update(map[string]interface{"Age":30,"Name":"Tom"})
// 2) KV 可变参数：Update("Age",30,"Name","Tom")
// 生成语句示例：UPDATE User SET Age = ?, Name = ? WHERE Name = ?
func (s *Session) Update(kv ...interface{}) (int64, error) {
	s.CallMethod(BeforeUpdate, nil)
	m, ok := kv[0].(map[string]interface{})
	if !ok {
		m = make(map[string]interface{})
		for i := 0; i < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
	}
	s.clause.Set(clause.UPDATE, s.RefTable().Name, m)
	sql, vars := s.clause.Build(clause.UPDATE, clause.WHERE)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	s.CallMethod(AfterUpdate, nil)
	return result.RowsAffected()
}

func (s *Session) Where(desc string, args ...interface{}) *Session {
	var vars []interface{}
	s.clause.Set(clause.WHERE, append(append(vars, desc), args...)...)
	return s
}

func (s *Session) OrderBy(desc string) *Session {
	s.clause.Set(clause.ORDERBY, desc)
	return s
}

// First 获取第一条记录：内部通过 Limit(1).Find(...) 实现：
// 示例：u := &User{}; s.Where("Age = ?", 18).First(u)
// 若无记录返回错误 errors.New("NOT FOUND")。
func (s *Session) First(value interface{}) error {
	dest := reflect.Indirect(reflect.ValueOf(value))
	destSlice := reflect.New(reflect.SliceOf(dest.Type())).Elem()
	if err := s.Limit(1).Find(destSlice.Addr().Interface()); err != nil {
		return err
	}

	if destSlice.Len() == 0 {
		return errors.New("NOT FOUND")
	}

	dest.Set(destSlice.Index(0))
	return nil
}

// Limit 设置 LIMIT 子句，限制返回行数；可链式调用：s.Limit(1).First(&u)
func (s *Session) Limit(num int) *Session {
	s.clause.Set(clause.LIMIT, num)
	return s
}

func (s *Session) Count() (int64, error) {
	s.clause.Set(clause.COUNT, s.RefTable().Name)
	sql, vars := s.clause.Build(clause.COUNT, clause.WHERE)
	row := s.Raw(sql, vars...).QueryRow()
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Session) Delete() (int64, error) {
	s.CallMethod(BeforeDelete, nil)
	s.clause.Set(clause.DELETE, s.RefTable().Name)
	sql, vars := s.clause.Build(clause.DELETE, clause.WHERE)

	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	s.CallMethod(AfterDelete, nil)
	return result.RowsAffected()
}
