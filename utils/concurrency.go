package utils

import "sync"

// https://www.golangtc.com/t/559e97d6b09ecc22f6000053
// thank you very much

type Pool struct {
	queue chan int
	Wg    *sync.WaitGroup
	Size  int				// pool size
}

func NewPool(cap, total int) *Pool {
	if cap < 1 {
		cap = 1
	}
	p := &Pool{
		queue: make(chan int, cap),
		Wg:    new(sync.WaitGroup),
	}
	p.Wg.Add(total)
	p.Size = 0
	return p
}
func (p *Pool) AddOne() {
	p.queue <- 1
	p.Size ++
}
func (p *Pool) DelOne() {
	<-p.queue
	p.Wg.Done()
	p.Size --
}
