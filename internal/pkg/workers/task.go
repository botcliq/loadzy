package workers

type Task struct {
	Err  error
	Data interface{}
	c    string
}

func NewTask(c string, data interface{}) *Task {
	return &Task{c: c, Data: data}
}

func process(workerID int, task *Task) {

}
