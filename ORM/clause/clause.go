package clause

import "strings"

// Clause 保存一条待构建 SQL 所需的各个片段（子句）以及对应的绑定变量。
// 设计意图：将不同语义的 SQL 片段（INSERT/WHERE/LIMIT 等）分离，按需组合顺序生成最终语句，
// 避免在业务代码里频繁字符串拼接，提高可读性与可维护性。
type Clause struct {
	sql     map[Type]string
	sqlVars map[Type][]interface{}
}

// Type 标识支持的子句类型枚举，通过 iota 自增实现。
// 注意顺序本身无强制语义，最终输出顺序由 Build(orders ...Type) 的参数决定。
type Type int

// 支持的子句类型：
// INSERT  -> INSERT INTO table (fields)
// VALUES  -> VALUES (...), (...)
// SELECT  -> SELECT fields FROM table
// LIMIT   -> LIMIT ?
// WHERE   -> WHERE condition
// ORDERBY -> ORDER BY xxx
const (
	INSERT Type = iota
	VALUES
	SELECT
	WHERE
	ORDERBY
	LIMIT
	UPDATE
	DELETE
	COUNT
)

// Set 根据子句类型生成并存储对应的 SQL 片段及其变量。
// 输入 vars 的含义由具体 generator 决定（详见 generator.go 中的同名函数）。
// 若首次使用，延迟初始化内部 map，避免在构造阶段浪费空间。
func (c *Clause) Set(name Type, vars ...interface{}) {
	if c.sql == nil {
		c.sql = make(map[Type]string)
		c.sqlVars = make(map[Type][]interface{})
	}

	sql, vars := generators[name](vars...)
	c.sql[name] = sql
	c.sqlVars[name] = vars
}

// Build 按照传入的子句类型顺序构建最终的 SQL 语句及其绑定变量列表。
func (c *Clause) Build(orders ...Type) (string, []interface{}) {
	var sqls []string
	var vars []interface{}

	for _, order := range orders {
		// 仅拼接已经存在的子句
		if sql, ok := c.sql[order]; ok {
			sqls = append(sqls, sql)
			vars = append(vars, c.sqlVars[order]...)
		}
	}

	return strings.Join(sqls, " "), vars
}
