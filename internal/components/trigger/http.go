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
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

type httpAction struct {
	interval time.Duration
	times    int
	url      string
	method   string
}

func NewHTTPAction(intervalStr string, times int, url, method string) Action {
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
		interval: interval,
		times:    times,
		url:      url,
		method:   strings.ToUpper(method),
	}
}

func (h *httpAction) Do() error {
	t := time.NewTicker(h.interval)
	c := 1
	client := &http.Client{}
	request, err := http.NewRequest(h.method, h.url, nil)
	if err != nil {
		logger.Log.Errorf("new request error %v", err)
		return err
	}

	logger.Log.Infof("Trigger will request URL %s %d times, %s seconds apart each time.", h.url, h.times, h.interval)

	for range t.C {
		logger.Log.Debugf("request URL %s the %d time.", h.url, c)

		response, err := client.Do(request)
		if err != nil {
			logger.Log.Errorf("do request error %v", err)
			return err
		}
		response.Body.Close()

		logger.Log.Infof("do request %v response http code %v", h.url, response.StatusCode)
		if response.StatusCode == http.StatusOK {
			logger.Log.Debugf("do http action %+v success.", *h)
			break
		}

		if h.times > 0 {
			if h.times <= c {
				return fmt.Errorf("do request %d times, but still failed", c)
			}
			c++
		}
	}

	return nil
}
