package singleflight

import "sync"

// call 代表正在进行中，或已经结束的请求
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group 管理不同 key 的请求(call)
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// Do 执行给定的函数fn 并确保同一时刻相同的key只会执行一次
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}

	// 已有相同 key 的调用在执行：释放锁后等待完成，直接复用结果
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	// wg.Add(1) 锁加1。
	// wg.Wait() 阻塞，直到锁被释放。
	// wg.Done() 锁减1。

	// 无调用,注册新的 call，当前协程成为首个调用者
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	// 调用 fn，发起请求
	c.val, c.err = fn()
	c.wg.Done() // 请求结束

	// 更新 g.m
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
