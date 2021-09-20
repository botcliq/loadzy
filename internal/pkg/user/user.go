package user

import (
	"sync"
	"time"

	"github.com/botcliq/loadzy/internal/pkg/action"
	"github.com/botcliq/loadzy/internal/pkg/feeder"
	"github.com/botcliq/loadzy/internal/pkg/result"
	"github.com/botcliq/loadzy/internal/pkg/testdef"
	"github.com/botcliq/loadzy/internal/pkg/workers"
)

type User struct {
	Client  string
	Id      int
	Actions []*action.Action
	Limiter chan *workers.Task
}

func New(Id int, c chan *workers.Task) *User {
	return &User{Id: Id, Limiter: c}
}

func (u *User) LaunchActions(t *testdef.TestDef, resultsChannel chan result.HttpReqResult, wg *sync.WaitGroup, actions []action.Action, UID string) {
	var sessionMap = make(map[string]string)

	for i := 0; i < t.Iterations; i++ {
		// Make sure the sessionMap is cleared before each iteration - except for the UID which stays
		cleanSessionMapAndResetUID(UID, sessionMap)
		// If we have feeder data, pop an item and push its key-value pairs into the sessionMap
		feedSession(t, sessionMap)
		// Iterate over the actions. Note the use of the command-pattern like Execute method on the Action interface
		for _, action := range actions {
			if action != nil {
				t := workers.NewTask(action, resultsChannel, &sessionMap, wg)
				u.Limiter <- t
			}
		}
		var waitDuration float32 = (float32(t.Users) / float32(t.Rampup)) * float32(len(t.Actions))
		time.Sleep(time.Duration(int(1000*waitDuration)) * time.Millisecond)
	}
}

func cleanSessionMapAndResetUID(UID string, sessionMap map[string]string) {
	// Optimization? Delete all entries rather than reallocate map from scratch for each new iteration.
	for k := range sessionMap {
		delete(sessionMap, k)
	}
	sessionMap["UID"] = UID
}

func feedSession(t *testdef.TestDef, sessionMap map[string]string) {
	if t.Feeder.Type != "" {
		go feeder.NextFromFeeder()       // Do async
		feedData := <-feeder.FeedChannel // Will block here until feeder delivers value over the FeedChannel
		for item := range feedData {
			sessionMap[item] = feedData[item]
		}
	}
}
