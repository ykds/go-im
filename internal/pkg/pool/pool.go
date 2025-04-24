package pool

import (
	"math"
	"math/bits"
	"sync"
	"unsafe"
)

var defaultPool *bufferPool

type bufferPool struct {
	pool [32]sync.Pool
}

func InitBufferPool() {
	defaultPool = NewBufferPool()
}

func Get(size uint32) []byte {
	return defaultPool.Get(size)
}

func Put(b []byte) {
	defaultPool.Put(b)
}

func NewBufferPool() *bufferPool {
	p := &bufferPool{}
	return p
}

func (bp *bufferPool) Get(size uint32) []byte {
	if size == 0 {
		return nil
	}
	if size > math.MaxInt32 {
		return nil
	}
	// pool[i] 存放的是容量 1<<i 大小地 buffer
	// 所以通过 bits.Len32(size - 1) 计算出 size 的位数，相当于找到大于等于 size 的最小 2 的幂的内存。
	idx := bits.Len32(size - 1)
	ptr, _ := bp.pool[idx].Get().(*byte)
	if ptr == nil {
		return make([]byte, size, 1<<idx)
	}
	return unsafe.Slice(ptr, 1<<idx)[:size]
}

func (bp *bufferPool) Put(buf []byte) {
	size := cap(buf)
	if size == 0 || size > math.MaxInt32 {
		return
	}
	// size - 1是保证 1 << idx 后的结果一样。
	// 假设 size 是 2 的幂，那么减1后就少一位；而如果不是 2 的幂，减1后位数不位；
	// 如 size=4 (100)，减1后为 3 (11)，1<<idx 后为 4 (100); size=3 (11)，减1后为 2 (10)，1<<idx 后为 4 (100)
	// 这是计算大于等于 size 的最小 2 的幂
	idx := bits.Len32(uint32(size) - 1)
	// 在设计上，从内存池返回的容量总是 2 的幂，不相等，说明 size 不是2次幂，不能放到当前 index 的池子。
	// 因为 n 位字节的最小值是 n-1 位字节的最大值，如 n=3, 最小值是 100, 而n=2的最大值是11.
	// 所以，把这个buf放到上一个池子总能满足申请需求；而放到当前池子可能会导致空间不足。比如n=3，但是大小是100=4, 我申请5的话，buf不够大了。
	if size != 1<<idx {
		idx--
	}
	bp.pool[idx].Put(unsafe.SliceData(buf))
}
