package gee

import (
	"log"
	"time"
)

// Logger 中间件 记录请求耗时和状态码
func Logger() HandlerFunc {
	return func(c *Context) {
		//开始记录时间
		t := time.Now()
		//执行后续中间件活最终处理函数
		c.Next()
		// 计算耗时
		log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}
