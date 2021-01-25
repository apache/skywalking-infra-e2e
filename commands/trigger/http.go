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
	"net/http"
	"strings"
	"time"

	"github.com/apache/skywalking-infra-e2e/internal/flags"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

type httpAction struct {
	interval string
	times    int
	url      string
	method   string
}

func NewHTTPAction() Action {
	return &httpAction{
		interval: flags.Interval,
		times:    flags.Times,
		url:      flags.HttpUrl,
		method:   strings.ToUpper(flags.HttpMethod),
	}
}

func (h *httpAction) Do() error {

	t := time.NewTicker(time.Second)
	c := 1
	client := &http.Client{}
	request, err := http.NewRequest(h.method, h.url, nil)
	if err != nil {
		logger.Log.Errorf("new request error %v", err)
		return err
	}

	for range t.C {
		response, err := client.Do(request)
		if err != nil {
			logger.Log.Errorf("do request error %v", err)
			return err
		}
		response.Body.Close()

		logger.Log.Infof("do request %v response http code %v", h.url, response.StatusCode)

		if h.times > 0 {
			if h.times <= c {
				break
			}
			c++
		}
	}

	return nil
}
