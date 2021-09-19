/**
The MIT License (MIT)

Copyright (c) 2015 ErikL

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/botcliq/loadzy/internal/pkg/action"
	"github.com/botcliq/loadzy/internal/pkg/feeder"
	"github.com/botcliq/loadzy/internal/pkg/result"
	"github.com/botcliq/loadzy/internal/pkg/runtime"
	ws "github.com/botcliq/loadzy/internal/pkg/server"
	"github.com/botcliq/loadzy/internal/pkg/testdef"
	"github.com/botcliq/loadzy/internal/pkg/user"
	"github.com/botcliq/loadzy/internal/pkg/workers"
	"gopkg.in/yaml.v2"
	//"github.com/davecheney/profile"
)

func main() {

	spec := parseSpecFile()

	// defer profile.Start(profile.CPUProfile).Stop()

	// Start the web socket server, will not block exit until forced
	go ws.StartWsServer()

	runtime.SimulationStart = time.Now()
	dir, _ := os.Getwd()
	dat, _ := ioutil.ReadFile(dir + "/" + spec)

	var t testdef.TestDef
	err := yaml.Unmarshal([]byte(dat), &t)
	fail(err)

	if !testdef.ValidateTestDefinition(&t) {
		return
	}

	actions, isValid := action.BuildActionList(&t)
	if !isValid {
		return
	}

	if t.Feeder.Type == "csv" {
		feeder.Csv(t.Feeder.Filename, ",")
	} else if t.Feeder.Type != "" {
		log.Fatal("Unsupported feeder type: " + t.Feeder.Type)
	}

	result.OpenResultsFile(dir + "/results/log/latest.log")
	user.SpawnUsers(&t, actions)

	fmt.Printf("Done in %v\n", time.Since(runtime.SimulationStart))
	fmt.Println("Building reports, please wait...")
	result.CloseResultsFile()
	//buildReport()
}

func parseSpecFile() string {
	if len(os.Args) == 1 {
		fmt.Println("No command line arguments, exiting...")
		panic("Cannot start simulation, no YAML simulaton specification supplied as command-line argument")
	}
	var s, sep string
	for i := 1; i < len(os.Args); i++ {
		s += sep + os.Args[i]
		sep = " "
	}
	if s == "" {
		panic(fmt.Sprintf("Specified simulation file '%s' is not a .yml file", s))
	}
	return s
}

var userMap map[int]*user.User
var Limiter chan *workers.Task

func RunTraffic(t *testdef.TestDef) {
	Limiter = make(chan *workers.Task, 1000)
	userMap = make(map[int]*user.User)
	for i := 1; i <= t.Users; i++ {
		u = user.New(i, Limiter)
		userMap[user.Id] = u
	}

}

func fail(err error) {
	if err != nil {
		fmt.Printf("%v\n", err.Error())
		os.Exit(1)
	}
}
