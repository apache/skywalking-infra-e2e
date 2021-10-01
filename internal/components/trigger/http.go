//
// Licensed to Apache Software Foundation (ASF) under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Apache Software Foundation (ASF) licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package trigger

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

type httpAction struct {
	interval      time.Duration
	times         int
	url           string
	method        string
	body          string
	headers       map[string]string
	executedCount int
}

func NewHTTPAction(intervalStr string, times int, url, method, body string, headers map[string]string) Action {
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		logger.Log.Errorf("interval [%s] parse error: %s.", intervalStr, err)
		return nil
	}

	if interval <= 0 {
		logger.Log.Errorf("interval [%s] is not positive", interval)
		return nil
	}

	// there can be env variables in url, say, "http://${GATEWAY_HOST}:${GATEWAY_PORT}/test"
	url = os.ExpandEnv(url)

	return &httpAction{
		interval:      interval,
		times:         times,
		url:           url,
		method:        strings.ToUpper(method),
		body:          body,
		headers:       headers,
		executedCount: 0,
	}
}

func (h *httpAction) Do() error {
	ctx := context.Background()
	t := time.NewTicker(h.interval)
	h.executedCount = 0
	client := &http.Client{}

	r := strings.NewReader(h.body)
	rc := ioutil.NopCloser(r)

	request, err := http.NewRequest(h.method, h.url, rc)
	headers := http.Header{}
	for k, v := range h.headers {
		headers[k] = []string{v}
	}
	request.Header = headers
	if err != nil {
		logger.Log.Errorf("new request error %v", err)
		return err
	}

	logger.Log.Infof("Trigger will request URL %s %d times, %s seconds apart each time.", h.url, h.times, h.interval)

	// execute until success
	for range t.C {
		err = h.executeOnce(client, request)
		if err == nil {
			break
		}
		if !h.couldContinue() {
			logger.Log.Errorf("do request %d times, but still failed", h.times)
			return err
		}
	}

	// background interval trigger
	go func() {
		for {
			select {
			case <-t.C:
				err = h.executeOnce(client, request)
				if !h.couldContinue() {
					return
				}
			case <-ctx.Done():
				t.Stop()
				return
			}
		}
	}()

	return nil
}

// execute http request once time
func (h *httpAction) executeOnce(client *http.Client, req *http.Request) error {
	logger.Log.Debugf("request URL %s the %d time.", h.url, h.executedCount)
	response, err := client.Do(req)
	h.executedCount++
	if err != nil {
		logger.Log.Errorf("do request error %v", err)
		return err
	}
	_ = response.Body.Close()

	logger.Log.Debugf("do request %v response http code %v", h.url, response.StatusCode)
	if response.StatusCode == http.StatusOK {
		logger.Log.Debugf("do http action %+v success.", *h)
		return nil
	}
	return fmt.Errorf("do request failed, response status code: %d", response.StatusCode)
}

// verify http action could continue
func (h *httpAction) couldContinue() bool {
	if h.times > 0 && h.times <= h.executedCount {
		return false
	}
	return true
}
