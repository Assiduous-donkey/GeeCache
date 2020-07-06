package geeCache

// 一个“只读”的数据结构
type ByteView struct {
	data []byte // 使用byte是为了支持任意的数据类型存储
}

func (view ByteView) Len() int {
	return len(view.data)
}
func (view ByteView) ByteSlice() []byte {
	// 返回一份数据拷贝 避免对数据的意外修改
	return cloneBytes(view.data)
}
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
func (view ByteView) String() string {
	return string(view.data)
}
