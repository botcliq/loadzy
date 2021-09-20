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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"reflect"
	"strings"
	"time"

	"github.com/botcliq/loadzy/internal/pkg/result"
	"github.com/botcliq/loadzy/internal/pkg/runtime"
	"github.com/botcliq/loadzy/internal/pkg/stats"
	"github.com/botcliq/loadzy/internal/pkg/testdef"
	"github.com/botcliq/loadzy/internal/pkg/util"
	"github.com/oliveagle/jsonpath"
	"gopkg.in/xmlpath.v2"
)

// Accepts a Httpaction and a one-way channel to write the results to.
func DoHttpRequest(httpAction HttpAction, resultsChannel chan result.HttpReqResult, sessionMap map[string]string) {
	req := buildHttpRequest(httpAction, sessionMap)

	start := time.Now()
	r := stats.Result{Attack: "HTTP load"}
	var DefaultTransport http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	stats.AddRequest(1, fmt.Sprintf("[%s:%s]->", httpAction.Url, httpAction.Method))
	dumpedBody, err := httputil.DumpRequest(req, true)

	if err != nil {
		fmt.Println(err)
	} else {
		r.BytesOut = uint64(len(dumpedBody))
	}

	resp, err := DefaultTransport.RoundTrip(req)

	if err != nil {
		log.Printf("HTTP request failed: %s", err)
	} else {
		elapsed := time.Since(start)
		responseBody, err := ioutil.ReadAll(resp.Body)
		r.Timestamp = time.Now()
		r.Code = fmt.Sprintf("[%s:%d]->", httpAction.Url, resp.StatusCode)
		r.Status = resp.StatusCode
		r.BytesIn = uint64(len(responseBody))
		r.Latency = elapsed

		stats.Add(1, &r)

		if err != nil {
			//log.Fatal(err)
			log.Printf("Reading HTTP response failed: %s\n", err)
			httpReqResult := buildHttpResult(0, resp.StatusCode, elapsed.Nanoseconds(), httpAction.Title)

			resultsChannel <- httpReqResult
		} else {
			defer resp.Body.Close()

			if httpAction.StoreCookie != "" {
				for _, cookie := range resp.Cookies() {

					if cookie.Name == httpAction.StoreCookie {
						sessionMap["____"+cookie.Name] = cookie.Value
					}
				}
			}

			// if action specifies response action, parse using regexp/jsonpath
			processResult(httpAction, sessionMap, responseBody)

			httpReqResult := buildHttpResult(len(responseBody), resp.StatusCode, elapsed.Nanoseconds(), httpAction.Title)

			resultsChannel <- httpReqResult
		}
	}
}

func buildHttpResult(contentLength int, status int, elapsed int64, title string) result.HttpReqResult {
	httpReqResult := result.HttpReqResult{
		Type:    "HTTP",
		Latency: elapsed,
		Size:    contentLength,
		Status:  status,
		Title:   title,
		When:    time.Since(runtime.SimulationStart).Nanoseconds(),
	}
	return httpReqResult
}

func buildHttpRequest(httpAction HttpAction, sessionMap map[string]string) *http.Request {
	var req *http.Request
	var err error
	if httpAction.Body != "" {
		reader := strings.NewReader(util.SubstParams(sessionMap, httpAction.Body))
		req, err = http.NewRequest(httpAction.Method, util.SubstParams(sessionMap, httpAction.Url), reader)
	} else if httpAction.Template != "" {
		reader := strings.NewReader(util.SubstParams(sessionMap, httpAction.Template))
		req, err = http.NewRequest(httpAction.Method, util.SubstParams(sessionMap, httpAction.Url), reader)
	} else {
		req, err = http.NewRequest(httpAction.Method, util.SubstParams(sessionMap, httpAction.Url), nil)
	}
	if err != nil {
		log.Fatal(err)
	}

	// Add headers
	req.Header.Add("Accept", httpAction.Accept)
	if httpAction.ContentType != "" {
		req.Header.Add("Content-Type", httpAction.ContentType)
	}

	for key, value := range httpAction.Headers {
		req.Header.Add(key, value)
	}

	if hostHeader, found := httpAction.Headers["host"]; found {
		req.Host = hostHeader
	}

	if hostHeader, found := httpAction.Headers["Host"]; found {
		req.Host = hostHeader
	}

	// Add cookies stored by subsequent requests in the sessionMap having the kludgy ____ prefix
	for key, value := range sessionMap {
		if strings.HasPrefix(key, "____") {

			cookie := http.Cookie{
				Name:  key[4:],
				Value: value,
			}

			req.AddCookie(&cookie)
		}
	}

	return req
}

/**
 * If the httpAction specifies a Jsonpath in the Response, try to extract value(s)
 * from the responseBody.
 *
 * TODO extract both Jsonpath handling and Xmlpath handling into separate functions, and write tests for them.
 */
func processResult(httpAction HttpAction, sessionMap map[string]string, responseBody []byte) {
	if httpAction.ResponseHandler.Jsonpath != "" {
		jsonPattern, err := jsonpath.Compile(httpAction.ResponseHandler.Jsonpath)
		if err != nil {
			log.Fatal(err)
		}

		var jsonData interface{}
		json.Unmarshal(responseBody, &jsonData)

		res, err := jsonPattern.Lookup(jsonData)
		if err != nil {
			log.Fatal(err)
		}

		var resultArray []string
		v := reflect.ValueOf(res)
		switch v.Kind() {
		case reflect.String:
			resultArray = []string{res.(string)}
		case reflect.Slice:
			a := res.([]interface{})
			resultArray = make([]string, len(a))
			for idx, val := range a {
				resultArray[idx] = fmt.Sprintf("%s", val)
			}
		default:
			log.Printf("Unknown type [%T]", reflect.TypeOf(res))
		}
		passResultIntoSessionMap(resultArray, httpAction, sessionMap)
	}

	if httpAction.ResponseHandler.Xmlpath != "" {
		path := xmlpath.MustCompile(httpAction.ResponseHandler.Xmlpath)
		r := bytes.NewReader(responseBody)
		root, err := xmlpath.Parse(r)

		if err != nil {
			log.Fatal(err)
		}

		iterator := path.Iter(root)
		hasNext := iterator.Next()
		if hasNext {
			resultsArray := make([]string, 0, 10)
			for {
				if hasNext {
					node := iterator.Node()
					resultsArray = append(resultsArray, node.String())
					hasNext = iterator.Next()
				} else {
					break
				}
			}
			passResultIntoSessionMap(resultsArray, httpAction, sessionMap)
		}
	}

	// log.Println(string(responseBody))
}

/**
 * Trims leading and trailing byte r from string s
 */
func trimChar(s string, r byte) string {
	sz := len(s)

	if sz > 0 && s[sz-1] == r {
		s = s[:sz-1]
	}
	sz = len(s)
	if sz > 0 && s[0] == r {
		s = s[1:sz]
	}
	return s
}

func passResultIntoSessionMap(resultsArray []string, httpAction HttpAction, sessionMap map[string]string) {
	resultCount := len(resultsArray)

	if resultCount > 0 {
		switch httpAction.ResponseHandler.Index {
		case testdef.FIRST:
			sessionMap[httpAction.ResponseHandler.Variable] = resultsArray[0]
			break
		case testdef.LAST:
			sessionMap[httpAction.ResponseHandler.Variable] = resultsArray[resultCount-1]
			break
		case testdef.RANDOM:
			if resultCount > 1 {
				rand.Seed(time.Now().UnixNano())
				sessionMap[httpAction.ResponseHandler.Variable] = resultsArray[rand.Intn(resultCount-1)]
			} else {
				sessionMap[httpAction.ResponseHandler.Variable] = resultsArray[0]
			}
			break
		}

	} else {
		// TODO how to handle requested, but missing result?
	}
}
