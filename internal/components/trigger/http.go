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
	"fmt"
	"io"
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
	stopCh        chan struct{}
	client        *http.Client
}

func NewHTTPAction(intervalStr string, times int, url, method, body string, headers map[string]string) (Action, error) {
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, err
	}

	if interval <= 0 {
		return nil, fmt.Errorf("trigger interval should be > 0, but was %s", interval)
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
		stopCh:        make(chan struct{}),
		client:        &http.Client{},
	}, nil
}

func (h *httpAction) Do() chan error {
	t := time.NewTicker(h.interval)

	logger.Log.Infof("trigger will request URL %s %d times with interval %s.", h.url, h.times, h.interval)

	result := make(chan error)
	sent := false
	go func() {
		for {
			select {
			case <-t.C:
				err := h.execute()

				// `send == false && h.times == h.executedCount` makes sure to only send firstly the error and
				// ignore errors before.
				if !sent && (err == nil || h.times == h.executedCount) {
					result <- err
					sent = true
				}
			case <-h.stopCh:
				t.Stop()
				result <- nil
				return
			}
		}
	}()

	return result
}

func (h *httpAction) Stop() {
	h.stopCh <- struct{}{}
}

func (h *httpAction) request() (*http.Request, error) {
	request, err := http.NewRequest(h.method, h.url, strings.NewReader(h.body))
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	for k, v := range h.headers {
		headers[k] = []string{v}
	}
	request.Header = headers
	return request, err
}

func (h *httpAction) execute() error {
	req, err := h.request()
	if err != nil {
		logger.Log.Errorf("failed to create new request %v", err)
		return err
	}
	logger.Log.Debugf("request URL %s the %d time.", h.url, h.executedCount)
	response, err := h.client.Do(req)
	h.executedCount++
	if err != nil {
		logger.Log.Errorf("do request error %v", err)
		return err
	}
	_, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()

	logger.Log.Debugf("do request %v response http code %v", h.url, response.StatusCode)
	if response.StatusCode == http.StatusOK {
		logger.Log.Debugf("do http action %+v success.", *h)
		return nil
	}
	return fmt.Errorf("do request failed, response status code: %d", response.StatusCode)
}
