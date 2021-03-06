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
package action

import (
	"fmt"
	"net"
	"time"

	"github.com/botcliq/loadzy/internal/pkg/result"
	"github.com/botcliq/loadzy/internal/pkg/runtime"
	"github.com/botcliq/loadzy/internal/pkg/util"
)

var conn net.Conn

// Accepts a TcpAction and a one-way channel to write the results to.
func DoTcpRequest(tcpAction TcpAction, resultsChannel chan result.HttpReqResult, sessionMap map[string]string) {

	address := util.SubstParams(sessionMap, tcpAction.Address)
	payload := util.SubstParams(sessionMap, tcpAction.Payload)

	if conn == nil {
		var err error
		conn, err = net.Dial("tcp", address)
		if err != nil {
			fmt.Printf("TCP socket closed, error: %s\n", err)
			conn = nil
			return
		}
		// conn.SetDeadline(time.Now().Add(100 * time.Millisecond))
	}

	start := time.Now()

	_, err := fmt.Fprintf(conn, payload+"\r\n")
	if err != nil {
		fmt.Printf("TCP request failed with error: %s\n", err)
		conn = nil
	}

	elapsed := time.Since(start)
	resultsChannel <- buildTcpResult(0, 200, elapsed.Nanoseconds(), tcpAction.Title)

}

func buildTcpResult(contentLength int, status int, elapsed int64, title string) result.HttpReqResult {
	httpReqResult := result.HttpReqResult{
		Type:    "TCP",
		Latency: elapsed,
		Size:    contentLength,
		Status:  status,
		Title:   title,
		When:    time.Since(runtime.SimulationStart).Nanoseconds(),
	}
	return httpReqResult
}
