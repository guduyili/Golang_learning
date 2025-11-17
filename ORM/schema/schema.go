package schema

import (
	"go/ast"
	"orm/dialect"
	"reflect"
)

// Field 表示数据库表的一列，对应结构体的一个字段
type Field struct {
	Name string
	Type string
	Tag  string
}

// Schema 表示数据库中的一张表，由结构体解析而来
type Schema struct {
	Model      interface{}      // 原始结构体对象
	Name       string           // 表名
	Fields     []Field          // 字段列表
	FieldNames []string         // 字段名列表
	fieldMap   map[string]Field // 字段名到 Field 的映射
}

// GetField 根据字段名获取对应的 Field 信息
func (s *Schema) GetField(name string) Field {
	field := s.fieldMap[name]
	return field
}

type ITableName interface {
	TableName() string
}

// Parse 将结构体解析为Schema对象
func Parse(dest interface{}, d dialect.Dialect) *Schema {
	modelType := reflect.Indirect(reflect.ValueOf(dest)).Type()
	var tableName string
	t, ok := dest.(ITableName)
	if !ok {
		tableName = modelType.Name()
	} else {
		tableName = t.TableName()
	}

	schema := &Schema{
		Model:    dest,
		Name:     tableName,
		fieldMap: make(map[string]Field),
	}

	// 遍历所有字段：跳过匿名字段与未导出字段
	for i := 0; i < modelType.NumField(); i++ {
		p := modelType.Field(i)
		zeroVal := reflect.New(p.Type).Elem()
		if !p.Anonymous && ast.IsExported(p.Name) {
			field := &Field{
				Name: p.Name,
				// 通过 (Dialect).DataTypeOf() 转换为数据库的字段类型
				// Type: d.DataTypeOf(reflect.Indirect(reflect.ValueOf(p.Type))),
				Type: d.DataTypeOf(zeroVal),
			}

			if v, ok := p.Tag.Lookup("orm"); ok {
				field.Tag = v
			}

			schema.Fields = append(schema.Fields, *field)
			schema.FieldNames = append(schema.FieldNames, p.Name)
			schema.fieldMap[p.Name] = *field
		}
	}
	return schema
}

// RecordValues 返回表对应的结构体对象
func (s *Schema) RecordValues(dest interface{}) interface{} {
	destValue := reflect.Indirect(reflect.ValueOf(dest))
	var fieldValues []interface{}
	for _, field := range s.Fields {
		fieldValues = append(fieldValues, destValue.FieldByName(field.Name).Interface())
	}
	return fieldValues
}
