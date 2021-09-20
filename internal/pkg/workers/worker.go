package workers

import (
	"fmt"
)

// Worker handles all the work
type Worker struct {
	ID       int
	taskChan chan *Task
	QuitChan chan bool
}

// NewWorker returns new instance of worker
func NewWorker(channel chan *Task, ID int) *Worker {
	return &Worker{
		ID:       ID,
		taskChan: channel,
		QuitChan: make(chan bool),
	}
}

// Start starts the worker
func (wr *Worker) Start() {
	fmt.Printf("Starting worker %d\n", wr.ID)

	go func() {
		for {
			select {
			case task := <-wr.taskChan:
				process(wr.ID, task)
			case <-wr.QuitChan:
				// We have been asked to stop.
				fmt.Printf("worker%d stopping\n", wr.ID)
				return
			}
		}
	}()
}

// Note that the worker will only stop *after* it has finished its work.
func (w *Worker) Stop() {
	go func() {
		w.QuitChan <- true
	}()
}
