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

	//response info
	StatusCode int
}

// NewContext 构造函数，创建一个新的 Context 实例
func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
	}
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
