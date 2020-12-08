package helpers

import (
	"sync"
)

func SignalWhenAnyErr(channels ...<-chan error) <-chan error {
	result := make(chan error)
	done := make(chan struct{}, 1)
	for i := 0; i < len(channels); i++ {
		go func(idx int) {
			ch := channels[idx]
			select {
			case err := <-ch:
				select {
				case result <- err:
					close(done)
				case <-done:
				}
			case <-done:
			}
		}(i)
	}
	return result
}
func SignalWhenAllErr(channels ...<-chan error) <-chan []error {
	mu := sync.Mutex{}
	wg := new(sync.WaitGroup)
	result := make(chan []error, 1)
	errors := make([]error, len(channels))
	for i := 0; i < len(channels); i++ {
		go func(idx int) {
			err := <-channels[idx]
			// while theorically our memory does not overlap with anybody and we must be able to set result
			// in the array directly, we can't do this because of MEMORY BASELINE and sharing problem
			mu.Lock()
			errors[idx] = err
			mu.Unlock()
			wg.Done()
		}(i)
	}
	go func() {
		wg.Wait()
		result <- errors
	}()
	return result
}
