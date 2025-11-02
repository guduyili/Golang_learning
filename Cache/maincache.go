// 负责与外部交互，控制缓存存储和获取的主流程

package cache

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

type Group struct {
	name      string
	getter    Getter
	mainCache cache
}


var{
	mu sync.RWMutex
	groups = make(map[string]*Group)
}

// NewGroup 创建一个新的缓存组
func NewGroup(name string, cacheBytes int64, getter Getter) *Group{
	if getter == nil{
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()


	g := &Group{
		name: name,
		getter: getter,
		mainCache:cache{
			cacheBytes: cacheBytes,
		},

	}
}