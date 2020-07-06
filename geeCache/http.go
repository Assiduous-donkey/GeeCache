package geeCache

import (
	"fmt"
	"geeCache/consistenthash"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// 为单机节点提供HTTP服务 进行分布式节点之间的通信

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 20
)

// HTTP服务端
type HTTPPool struct {
	self     string // 当前节点的地址 ip:port
	basePath string // 节点间通信地址的前缀
	mux      sync.Mutex
	peers    *consistenthash.Map
	// 节点地址对应的HTTP客户端
	// 这里的HTTP客户端 是当前节点要与其他节点交互的客户端
	httpGetters map[string]*httpGetter
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !strings.HasPrefix(req.URL.Path, p.basePath) {
		// 不属于节点间的通信
		panic("HTTPPool serving unexpected path " + req.URL.Path)
	}
	p.Log("%s %s", req.Method, req.URL.Path)
	// 节点间通信的url格式： /basePath/groupName/key
	parts := strings.SplitN(req.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		// 通信url格式不对
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// 添加节点 传入的是节点的地址 ip:port
func (p *HTTPPool) Set(peers ...string) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.peers = consistenthash.NewMap(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// 根据key返回存储该key的节点的HTTP客户端
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mux.Lock()
	defer p.mux.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

// HTTP客户端
type httpGetter struct {
	baseURL string
}

func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	// 从远程节点获取value
	url := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		// 用于转义能用明文正确发送的任何字符  如空格被转为%20
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}
	return bytes, nil
}

// 编译时检查httpGetter是否实现了PeerGetter接口
var _ PeerGetter = (*httpGetter)(nil)
