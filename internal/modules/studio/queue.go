package studio

import "time"

type JobQueue interface {
	EnqueueDispatch(jobID string, delay time.Duration) error
	EnqueueTimeout(jobID string, delay time.Duration) error
	Start() error
	Shutdown()
}

type noopQueue struct{}

func newNoopQueue() JobQueue { return noopQueue{} }

func (noopQueue) EnqueueDispatch(string, time.Duration) error { return nil }
func (noopQueue) EnqueueTimeout(string, time.Duration) error  { return nil }
func (noopQueue) Start() error                                { return nil }
func (noopQueue) Shutdown()                                   {}
