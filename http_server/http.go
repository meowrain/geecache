package http_server

import (
	"fmt"
	"io"
	"log"
	"miku_cache/consistenthash"
	"miku_cache/geecache"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache"
	defaultReplicas = 50
)

// HTTP池
// HTTPPool 只有 2 个参数，一个是 self，用来记录自己的地址，包括主机名/IP 和端口。
// 另一个是 basePath，作为节点间通讯地址的前缀，默认是 /_geecache/，那么 http://example.com/_geecache/ 开头的请求，就用于节点间的访问。因为一个主机上还可能承载其他的服务，加一段 Path 是一个好习惯。比如，大部分网站的 API 接口，一般以 /api 作为前缀。
type HTTPPool struct { // 承载一个节点的信息
	self        string // 用来记录自己的地址，包括主机名/IP 和端口 e.g: "https://exmaple.net:8000"
	basePath    string
	mu          sync.Mutex
	peers       *consistenthash.Map    //peers，类型是一致性哈希算法的 Map，用来根据具体的 key 选择节点。
	httpGetters map[string]*httpGetter //映射远程节点与对应的 httpGetter。每一个远程节点对应一个 httpGetter，因为 httpGetter 与远程节点的地址 baseURL 有关。
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log info with server name
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http_server requests
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	// 截取路径并去除前导斜杠
	pathPart := r.URL.Path[len(p.basePath):]
	trimmedPath := strings.TrimPrefix(pathPart, "/")
	// 分割路径为两部分（groupname 和 key）
	parts := strings.SplitN(trimmedPath, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]
	group := geecache.GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// 设置响应头,标识传输的内容类型为二进制
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// 添加远程节点，并且配置每个节点对应的请求baseURL（包含默认的实现了PeerGetter的GET函数）
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 实例化一致性哈希算法
	p.peers = consistenthash.New(defaultReplicas, nil)
	// 添加传入的节点
	p.peers.Add(peers...)
	// 为每个节点创建一个 HTTP 客户端
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		// 传入的是节点的地址 + basePath
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// 实现PeerPicker接口,用于 根据 key 选择对应的节点
func (p *HTTPPool) PickPeer(key string) (geecache.PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 如果存在对应的节点，而且不是自己？ 为什么不是自己？ 因为自己也可以作为一个节点
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		// 返回对应的节点的HTTP客户端
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

type httpGetter struct {
	baseURL string
}

// 从远程节点获取数据,实现PeerGetter接口 api服务器
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	// 构建完整的 URL
	u := fmt.Sprintf("%v%v/%v", h.baseURL, url.QueryEscape(group), url.QueryEscape(key))
	// 发送 HTTP GET 请求
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}
	// 从响应中读取数据
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}
	//	返回数据
	return bytes, nil
}

// 在 Go 中，var _ Interface = (*Struct)(nil) 是一种常见的设计模式，用于静态检查接口的实现。这段代码的目的就是确保 *httpGetter 类型实现了 PeerGetter 接口，如果没有正确实现该接口，编译器将在编译时报告错误。
var _ geecache.PeerGetter = (*httpGetter)(nil)
