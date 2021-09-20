package workers

import (
	"sync"

	"github.com/botcliq/loadzy/internal/pkg/action"
	"github.com/botcliq/loadzy/internal/pkg/result"
)

type Task struct {
	Err error
	c   action.Action
	rc  chan result.HttpReqResult
	sm  *map[string]string
	wg  *sync.WaitGroup
}

func NewTask(a action.Action, rc chan result.HttpReqResult, sm *map[string]string, wg *sync.WaitGroup) *Task {
	return &Task{c: a, rc: rc, sm: sm, wg: wg}
}

func process(workerID int, task *Task) {
	task.c.Execute(task.rc, *task.sm)
	task.wg.Done()
}
