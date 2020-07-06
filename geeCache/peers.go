package geeCache

type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// 每个节点必须实现的接口 用对应group查找缓存值 对应于一个HTTP客户端
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
