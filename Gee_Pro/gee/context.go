package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// H 是简化使用的JSON数据结构，方便构造键值对相应
type H map[string]interface{}

// Context 封装了当前HTTP请求的上下文信息
type Context struct {
	Writer http.ResponseWriter
	Req    *http.Request

	//request info
	Path   string
	Method string
	Params map[string]string

	//response info
	StatusCode int

	//middleware
	handlers []HandlerFunc
	index    int

	//engine
	engine *Engine
}

// NewContext 构造函数，创建一个新的 Context 实例
func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
		index:  -1,
	}
}

// 依次执行注册的中间件函数/处理函数链 index标识当前执行位置
func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

func (c *Context) Fail(code int, err string) {
	//中止中间件链的执行，直接返回错误信息
	c.index = len(c.handlers)
	c.JSON(code, H{"message": err})
}

// Param 获取路由参数
func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

// PostForm 获取POST请求的表单参数
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

// Query 获取URL查询参数
func (c *Context) Query(key string) string {
	// return c.Req.URL.Query().Get(key)
	return c.Req.URL.Query().Get(key)
}

// Status 设置HTTP响应状态码
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// SetHeader 设置HTTP响应头
func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

// String 返回纯文本相应
func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

// JSON 序列化对象为JSON并返回
func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	// json.NewEncoder(c.Writer).Encode(obj)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

// Data 直接写入字节数据
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

// HTML 返回HTML字符串
func (c *Context) HTML(code int, name string, data interface{}) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
		// http.Error(c.Writer, err.Error(), 500)
		c.Fail(500, err.Error())
	}
}
