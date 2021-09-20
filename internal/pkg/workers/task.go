package workers

import (
	"github.com/botcliq/loadzy/internal/pkg/action"
	"github.com/botcliq/loadzy/internal/pkg/result"
)

type Task struct {
	Err error
	c   action.Action
	rc  chan result.HttpReqRequest
	sm  *map[string]string
}

func NewTask(a action.Action, rc chan result.HttpReqRequest, sm *map[string]string) *Task {
	return &Task{c: a}
}

func process(workerID int, task *Task) {
	task.c.Execute(task.rc, *task.sm)
}
