package workers

import (
	"sync"
)

// Pool is the worker pool
type Pool struct {
	concurrency int
	Collector   chan *Task
}

// NewPool initializes a new pool with the given tasks and
// at the given concurrency.
func NewPool(concurrency int) *Pool {
	return &Pool{
		concurrency: concurrency,
		Collector:   make(chan *Task, 1000),
	}
}

// Run runs all work within the pool and blocks until it's
// finished.
func (p *Pool) Run(ip string, port int) {
	for i := 1; i <= p.concurrency; i++ {
		worker := NewWorker(p.Collector, i, ip, port)
		worker.Start()
	}
}

var singleton *Pool
var once sync.Once

func GetPool(concurrency int) *Pool {
	once.Do(func() {
		singleton = &Pool{
			concurrency: concurrency,
			Collector:   make(chan *Task, 1000),
		}
	})
	return singleton
}
