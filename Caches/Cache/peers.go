package cache

// PeerPicker 定义根据 key 选择对应节点的能力
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 表示具体节点的客户端能力，可通过 HTTP 等方式拉取远程数据
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
