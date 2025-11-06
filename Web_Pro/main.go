package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sc"
	"time"
)

func onlyForV2() sc.HandlerFunc {
	return func(c *sc.Context) {
		t := time.Now()
		c.Fail(500, "Internal Server Error")
		log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}

type student struct {
	Name string
	Age  int8
}

func FormatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

// FormatAsDate 是模板中可调用的自定义函数，用于把 time.Time 格式化为 YYYY-MM-DD

func main() {
	// 创建框架并注册日志中间件
	r := sc.New()
	// Logger 会在请求处理前后打印耗时等信息
	r.Use(sc.Logger())

	// 注册模板函数映射，模板中可使用 {{ FormatAsDate .now }}
	r.SetFuncMap(template.FuncMap{
		"FormatAsDate": FormatAsDate,
	})

	// 加载 templates 目录下的所有模板文件
	r.LoadHTMLGlob("templates/*")

	// 挂载静态文件目录：把 ./static 映射到 /assets
	r.Static("/assets", "./static")

	// 示例数据
	stu1 := &student{Name: "scktutu", Age: 20}
	stu2 := &student{Name: "Jack", Age: 22}

	// 路由：渲染单个模板文件
	r.GET("/", func(c *sc.Context) {
		// css.tmpl 会被渲染并返回
		c.HTML(http.StatusOK, "css.tmpl", nil)
	})

	// 路由：渲染数组/列表模板
	r.GET("/students", func(c *sc.Context) {
		// 将学生数组传递给模板，模板里可以遍历 stuArr
		c.HTML(http.StatusOK, "arr.tmpl", sc.H{
			"title":  "sc",
			"stuArr": [2]*student{stu1, stu2},
		})
	})

	r.GET("/panic", func(c *sc.Context) {
		names := []string{"scktutu"}
		// 故意触发 panic
		c.String(http.StatusOK, names[100])
	})

	// 路由：演示自定义模板函数的使用
	r.GET("/date", func(c *sc.Context) {
		c.HTML(http.StatusOK, "custom_func.tmpl", sc.H{
			"title": "sc",
			"now":   time.Date(2025, 10, 29, 0, 0, 0, 0, time.UTC),
		})
	})

	r.Run(":3000")
}
