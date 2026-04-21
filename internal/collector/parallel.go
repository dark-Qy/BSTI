package collector

import (
	"context"
	"sync"
)

const (
	mainDomainConcurrency  = 4
	extraFetchConcurrency  = 3
	chatHistoryConcurrency = 3
)

func runBoundedOrdered[T any, R any](ctx context.Context, inputs []T, limit int, fn func(context.Context, T) R) []R {
	if len(inputs) == 0 {
		return nil
	}
	if limit <= 0 {
		limit = 1
	}
	if limit > len(inputs) {
		limit = len(inputs)
	}

	type job struct {
		index int
		input T
	}

	jobs := make(chan job)
	results := make([]R, len(inputs))

	var wg sync.WaitGroup
	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				results[job.index] = fn(ctx, job.input)
			}
		}()
	}

	for index, input := range inputs {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return results
		case jobs <- job{index: index, input: input}:
		}
	}
	close(jobs)
	wg.Wait()
	return results
}
