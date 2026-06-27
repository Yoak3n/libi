package limiter

import (
	"sync"
	"time"
)

type RequestJob struct {
	Execute func(cookie string) error
	Done    chan error
}

type Dispatcher struct {
	limiter *AccountLimiter
	queue   chan *RequestJob
	stop    chan struct{}
	stopped chan struct{}
}

func NewDispatcher(limiter *AccountLimiter, queueSize int) *Dispatcher {
	if queueSize <= 0 {
		queueSize = 64
	}
	return &Dispatcher{
		limiter: limiter,
		queue:   make(chan *RequestJob, queueSize),
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (d *Dispatcher) Start(workerCount int) {
	if workerCount <= 0 {
		workerCount = 1
	}
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.worker()
		}()
	}
	go func() {
		wg.Wait()
		close(d.stopped)
	}()
}

func (d *Dispatcher) Dispatch(fn func(cookie string) error) error {
	job := &RequestJob{
		Execute: fn,
		Done:    make(chan error, 1),
	}
	d.queue <- job
	return <-job.Done
}

func (d *Dispatcher) Stop() {
	close(d.stop)
	<-d.stopped
}

func (d *Dispatcher) worker() {
	for {
		select {
		case <-d.stop:
			return
		case job := <-d.queue:
			d.executeJob(job)
		}
	}
}

func (d *Dispatcher) executeJob(job *RequestJob) {
	for {
		id, cookie, waitTime := d.limiter.PickAccount()
		if id == 0 {
			// No account available at all, wait and retry
			time.Sleep(5 * time.Second)
			continue
		}
		if waitTime > 0 {
			time.Sleep(waitTime)
		}

		err := job.Execute(cookie)
		if err != nil {
			d.limiter.Penalize(id)
		} else {
			d.limiter.Reward(id)
		}
		job.Done <- err
		return
	}
}
