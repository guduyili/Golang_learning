// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"
// )

// func main() {
// 	http.HandleFunc("/", indexHandler)
// 	http.HandleFunc("/hello", helloHandler)
// 	log.Fatal(http.ListenAndServe(":8090", nil))
// }

// func indexHandler(w http.ResponseWriter, req *http.Request) {
// 	fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
// }

// func helloHandler(w http.ResponseWriter, req *http.Request) {
// 	for k, v := range req.Header {
// 		fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
// 	}
// }

// package main

// import (
// 	"fmt"
// 	"net/http"
// 	"log"
// )

// type Engine struct{}

// func (engine *Engine)ServerHTTP(w http.ResponseWritr, req *http.Request){
// 	switch req.URL.Path{
// 	case: "/":
// 	// fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
// 	fmt.Fprintf(w)

// 	default:
// 	fmt.Fprintf(w, "404 NOT FOUND %s\n", req.URL)
// }
// }

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"gee"
)

func onlyForV2() gee.HandlerFunc {
	return func(c *gee.Context) {
		t := time.Now()
		c.Fail(500, "Internal Server Error")
		log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}

// func main() {
// 	r := gee.New()
// 	r.Use(gee.Logger())

// 	r.GET("/", func(c *gee.Context) {
// 		c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
// 	})

// 	v1 := r.Group("/v1")
// 	{
// 		v1.GET("/", func(c *gee.Context) {
// 			c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
// 		})

// 		v1.GET("/hello/:name", func(c *gee.Context) {
// 			c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
// 		})

// 	}

// 	// r.POST("/login", func(c *gee.Context) {
// 	// 	c.JSON(http.StatusOK, gee.H{
// 	// 		"username": c.PostForm("username"),
// 	// 		"password": c.PostForm("password"),
// 	// 	})
// 	// })

// 	v2 := r.Group("/v2")
// 	v2.Use(onlyForV2())
// 	{
// 		v2.GET("/hello/:name", func(c *gee.Context) {
// 			// expect /hello/geektutu
// 			c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
// 		})
// 		v2.POST("/login", func(c *gee.Context) {
// 			c.JSON(http.StatusOK, gee.H{
// 				"username": c.PostForm("username"),
// 				"password": c.PostForm("password"),
// 			})
// 		})

// 	}

// 	r.GET("/assets/*filepath", func(c *gee.Context) {
// 		c.JSON(http.StatusOK, gee.H{"filepath": c.Param("filepath")})
// 	})
// 	r.Run(":3000")
// }

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
	r := gee.New()
	// Logger 会在请求处理前后打印耗时等信息
	r.Use(gee.Logger())

	// 注册模板函数映射，模板中可使用 {{ FormatAsDate .now }}
	r.SetFuncMap(template.FuncMap{
		"FormatAsDate": FormatAsDate,
	})

	// 加载 templates 目录下的所有模板文件
	r.LoadHTMLGlob("templates/*")

	// 挂载静态文件目录：把 ./static 映射到 /assets
	r.Static("/assets", "./static")

	// 示例数据
	stu1 := &student{Name: "Geektutu", Age: 20}
	stu2 := &student{Name: "Jack", Age: 22}

	// 路由：渲染单个模板文件
	r.GET("/", func(c *gee.Context) {
		// css.tmpl 会被渲染并返回
		c.HTML(http.StatusOK, "css.tmpl", nil)
	})

	// 路由：渲染数组/列表模板
	r.GET("/students", func(c *gee.Context) {
		// 将学生数组传递给模板，模板里可以遍历 stuArr
		c.HTML(http.StatusOK, "arr.tmpl", gee.H{
			"title":  "gee",
			"stuArr": [2]*student{stu1, stu2},
		})
	})

	// 路由：演示自定义模板函数的使用
	r.GET("/date", func(c *gee.Context) {
		c.HTML(http.StatusOK, "custom_func.tmpl", gee.H{
			"title": "gee",
			"now":   time.Date(2025, 10, 29, 0, 0, 0, 0, time.UTC),
		})
	})

	r.Run(":3000")
}
