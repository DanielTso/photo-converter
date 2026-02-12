package pipeline

import (
	"context"
	"sync"
)

// Pool manages a pool of workers that execute conversion jobs.
type Pool struct {
	pipeline *Pipeline
	workers  int
}

// NewPool creates a worker pool with the given pipeline and worker count.
func NewPool(p *Pipeline, workers int) *Pool {
	if workers < 1 {
		workers = 1
	}
	return &Pool{pipeline: p, workers: workers}
}

// Run processes all jobs using the worker pool and returns results.
// It respects the context for cancellation.
func (pool *Pool) Run(ctx context.Context, jobs []Job) []Result {
	results := make([]Result, len(jobs))
	work := make(chan int, len(jobs))

	for i := range jobs {
		work <- i
	}
	close(work)

	var wg sync.WaitGroup
	workers := pool.workers
	if workers > len(jobs) {
		workers = len(jobs)
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range work {
				select {
				case <-ctx.Done():
					results[idx] = Result{Job: jobs[idx], Error: ctx.Err()}
					return
				default:
				}
				err := pool.pipeline.Execute(jobs[idx])
				results[idx] = Result{Job: jobs[idx], Error: err}
			}
		}()
	}

	wg.Wait()
	return results
}

// RunWithCallback processes jobs and calls the callback after each job completes.
func (pool *Pool) RunWithCallback(ctx context.Context, jobs []Job, cb func(Result, int, int)) {
	work := make(chan int, len(jobs))
	for i := range jobs {
		work <- i
	}
	close(work)

	total := len(jobs)
	var mu sync.Mutex
	completed := 0

	workers := pool.workers
	if workers > total {
		workers = total
	}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range work {
				select {
				case <-ctx.Done():
					r := Result{Job: jobs[idx], Error: ctx.Err()}
					mu.Lock()
					completed++
					c := completed
					mu.Unlock()
					cb(r, c, total)
					return
				default:
				}
				err := pool.pipeline.Execute(jobs[idx])
				r := Result{Job: jobs[idx], Error: err}
				mu.Lock()
				completed++
				c := completed
				mu.Unlock()
				cb(r, c, total)
			}
		}()
	}

	wg.Wait()
}
