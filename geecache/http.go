package geecache

import (
	"fmt"
	"io"
	"log"
	"mikucache/geecache/consistenthash"
	"mikucache/geecache/geecachepb"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

type HTTPPool struct {
	self        string
	basePath    string
	mu          sync.Mutex
	peers       *consistenthash.Map    // 一致性哈希算法的map，用来根据key选择节点
	httpGetters map[string]*httpGetter // 每一个远程节点对应一个http客户端
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self, // 启动服务器的url
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...any) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		http.Error(w, "Unexpected path: "+r.URL.Path, http.StatusNotFound)
		return
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
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
	body, err := proto.Marshal(&geecachepb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 创建一致性哈希map，虚拟节点数设置为默认的50，算法采用默认的crc32.ChecksumIEEE
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = NewhtthttpGetter(peer, p.basePath)
	}
}

// 提供根据选择的key 创建HTTP客户端从远程节点获取缓存只的能力
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

// ---------------------Add httpGetter 实现http客户端功能--------------------

type httpGetter struct {
	baseURL string // 要访问的远程节点的地址
}

func NewhtthttpGetter(node string, baseUrl string) *httpGetter {
	return &httpGetter{
		baseURL: node + baseUrl,
	}
}
func (h *httpGetter) Get(in *geecachepb.Request, out *geecachepb.Response) error {
	/*
			url.QueryEscape 它的主要作用是：
		1. 将字符串中的特殊字符转换为 URL 编码格式
		2. 确保 URL 中的参数值能够安全传输，避免特殊字符造成的问题
	*/

	u := fmt.Sprintf("%v%v/%v", h.baseURL, url.QueryEscape(in.GetGroup()), url.QueryEscape(in.GetKey()))
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}
	return nil
}

// 验证httpGetter结构体是否实现了PeerGetter接口
var _ PeerGetter = (*httpGetter)(nil)
