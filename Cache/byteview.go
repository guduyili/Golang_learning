package cache

// ByteView 提供对底层字节切片的只读视图，避免被外部修改
type ByteView struct {
	b []byte
}

// Len 返回当前视图包含的字节长度，实现lru.Value接口
func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice 返回当前视图的字节切片拷贝，防止外部修改原始数据
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

//String 将字节内容转换成字符串
func (v ByteView) String() string {
	return string(v.b)
}

// cloneBytes 返回一个字节切片的拷贝
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
