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
//

package config

import (
	"fmt"
	"os"
	"time"

	"github.com/apache/skywalking-infra-e2e/internal/constant"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"
)

// E2EConfig corresponds to configuration file e2e.yaml.
type E2EConfig struct {
	Setup   Setup   `yaml:"setup"`
	Cleanup Cleanup `yaml:"cleanup"`
	Trigger Trigger `yaml:"trigger"`
	Verify  Verify  `yaml:"verify"`
}

type Setup struct {
	Env                   string    `yaml:"env"`
	File                  string    `yaml:"file"`
	Steps                 []Step    `yaml:"steps"`
	Timeout               any       `yaml:"timeout"`
	InitSystemEnvironment string    `yaml:"init-system-environment"`
	Kind                  KindSetup `yaml:"kind"`

	timeout time.Duration
}

func (s *Setup) Finalize() error {
	interval, err := parseInterval(s.Timeout, "setup.timeout")
	if err != nil {
		return err
	}
	if interval <= 0 {
		interval = constant.DefaultWaitTimeout
	}
	s.timeout = interval
	return nil
}

func (s *Setup) GetTimeout() time.Duration {
	return s.timeout
}

type Cleanup struct {
	On string `yaml:"on"`
}

type Step struct {
	Name    string `yaml:"name"`
	Path    string `yaml:"path"`
	Command string `yaml:"command"`
	Waits   []Wait `yaml:"wait"`
}

type KindSetup struct {
	ImportImages []string         `yaml:"import-images"`
	ExposePorts  []KindExposePort `yaml:"expose-ports"`
}

type KindExposePort struct {
	Namespace string `yaml:"namespace"`
	Resource  string `yaml:"resource"`
	Port      string `yaml:"port"`
}

type Verify struct {
	RetryStrategy VerifyRetryStrategy `yaml:"retry"`
	Cases         []VerifyCase        `yaml:"cases"`
	FailFast      bool                `yaml:"fail-fast"`
	Concurrency   bool                `yaml:"concurrency"`
}

func (s *Setup) GetFile() string {
	// expand the file path with system environment
	file := os.ExpandEnv(s.File)
	file = util.ResolveAbs(file)
	return file
}

type Manifest struct {
	Path  string `yaml:"path"`
	Waits []Wait `yaml:"wait"`
}

type Run struct {
	Command string `yaml:"command"`
	Waits   []Wait `yaml:"wait"`
}

type Wait struct {
	Namespace     string `yaml:"namespace"`
	Resource      string `yaml:"resource"`
	LabelSelector string `yaml:"label-selector"`
	For           string `yaml:"for"`
}

type Trigger struct {
	Action   string            `yaml:"action"`
	Interval string            `yaml:"interval"`
	Times    int               `yaml:"times"`
	URL      string            `yaml:"url"`
	Method   string            `yaml:"method"`
	Body     string            `yaml:"body"`
	Headers  map[string]string `yaml:"headers"`
}

type VerifyCase struct {
	Name     string   `yaml:"name"`
	Query    string   `yaml:"query"`
	Actual   string   `yaml:"actual"`
	Expected string   `yaml:"expected"`
	Includes []string `yaml:"includes"`
}

type VerifyRetryStrategy struct {
	Count    int `yaml:"count"`
	Interval any `yaml:"interval"`
}

type ReusingCases struct {
	Cases []VerifyCase `yaml:"cases"`
}

// GetActual resolves the absolute file path of the actual data file.
func (v *VerifyCase) GetActual() string {
	return util.ResolveAbs(v.Actual)
}

// GetExpected resolves the absolute file path of the expected data file.
func (v *VerifyCase) GetExpected() string {
	return util.ResolveAbs(v.Expected)
}

// parseInterval parses a Duration field with number and string content for compatibility,
// only use this when we previously allow configuring number like 120 and now string like 2m.
// TODO remove this in 2.0
func parseInterval(retryInterval any, name string) (time.Duration, error) {
	var interval time.Duration
	var err error
	switch itv := retryInterval.(type) {
	case int:
		logger.Log.Warnf("configuring %v with number %v is deprecated and will be removed in future version,"+
			" please use Duration style instead, such as 10s, 1m.", name, itv)
		interval = time.Duration(itv) * time.Second
	case string:
		if interval, err = time.ParseDuration(itv); err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("failed to parse %v: %v", name, retryInterval)
	}
	if interval < 0 {
		interval = constant.DefaultWaitTimeout
	}
	return interval, nil
}
