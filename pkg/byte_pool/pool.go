package byte_pool

import "sync"

type Pool struct {
	pool *sync.Pool
}

func NewPool(len int) *Pool {
	p := &sync.Pool{
		New: func() any {
			buffer := make([]byte, 0, len)
			return buffer
		},
	}
	return &Pool{
		pool: p,
	}
}

func (p *Pool) Get() (buffer []byte, put func()) {
	buffer = p.pool.Get().([]byte)
	put = func() { p.pool.Put(buffer) }
	return
}
