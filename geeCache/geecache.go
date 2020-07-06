package geeCache

import (
	"fmt"
	"geeCache/singleflight"
	"log"
	"sync"
)

// 提供与用户交互的结构和接口

type Getter interface {
	Get(key string) ([]byte, error) // 回调函数
}
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// 核心结构
type Group struct {
	name      string
	getter    Getter // 缓存未命中时获取源数据的回调函数
	mainCache cache
	peers     PeerPicker
	loader    *singleflight.Group
}

var (
	mux    sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mux.Lock() // 写锁
	defer mux.Unlock()
	if _, ok := groups[name]; ok {
		log.Printf("group %s has existed\n", name)
		return nil
	}
	group := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = group
	return group
}
func GetGroup(name string) *Group {
	mux.RLock() // 读锁
	defer mux.RUnlock()
	group := groups[name]
	return group
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	// mainCache中已经实现了单机并发访问 所以这里不需要再加锁
	if value, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return value, nil
	}
	// 本机无该缓存 从远程节点或者数据源获取
	return g.load(key)
}
func (g *Group) load(key string) (value ByteView, err error) {
	view, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil { // 有远程节点
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key) // 执行回调函数
	})
	if err == nil {
		return view.(ByteView), nil
	}
	return
}

// 从远程节点获取缓存值
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{data: cloneBytes(bytes)}, nil
}
func (g *Group) getLocally(key string) (ByteView, error) {
	// 执行回调函数获取源数据
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	// 缓存对保存在本地 并 返回一份缓存值的拷贝
	value := ByteView{data: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}
