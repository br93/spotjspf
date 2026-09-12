package download

import "spotjspf/internal/domain/job"

type Dispatcher struct {
	queue   chan *job.Job
	service *Service
}

func NewDispatcher(service *Service, concurrency, queueSize int) *Dispatcher {
	d := &Dispatcher{
		queue:   make(chan *job.Job, queueSize),
		service: service,
	}
	for i := 0; i < concurrency; i++ {
		go d.worker()
	}
	return d
}

func (d *Dispatcher) Enqueue(j *job.Job) {
	d.queue <- j
}

func (d *Dispatcher) worker() {
	for j := range d.queue {
		d.service.Execute(j)
	}
}
