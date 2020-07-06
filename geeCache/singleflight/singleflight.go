package singleflight

import "sync"

// 表示一个请求
type call struct {
	wg    sync.WaitGroup
	value interface{}
	err   error
}
type Group struct {
	mux sync.Mutex // 用于并发修改map
	m   map[string]*call
}

func (g *Group) Do(key string, fun func() (interface{}, error)) (interface{}, error) {
	g.mux.Lock()
	if c, ok := g.m[key]; ok {
		g.mux.Unlock()
		c.wg.Wait()
		return c.value, c.err
	}
	c := new(call)
	g.m[key] = c
	c.wg.Add(1)
	g.mux.Unlock()

	c.value, c.err = fun()
	c.wg.Done()

	g.mux.Lock()
	delete(g.m, key)
	g.mux.Unlock()

	return c.value, c.err
}
