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

package verify

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/apache/skywalking-infra-e2e/internal/components/verifier"
	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

var (
	query    string
	actual   string
	expected string
)

func init() {
	Verify.Flags().StringVarP(&query, "query", "q", "", "the query to get the actual data, the result of the query should in YAML format")
	Verify.Flags().StringVarP(&actual, "actual", "a", "", "the actual data file, only YAML file format is supported")
	Verify.Flags().StringVarP(&expected, "expected", "e", "", "the expected data file, only YAML file format is supported")
}

// Verify verifies that the actual data satisfies the expected data pattern.
var Verify = &cobra.Command{
	Use:   "verify",
	Short: "verify if the actual data match the expected data",
	RunE: func(cmd *cobra.Command, args []string) error {
		if expected != "" {
			return verifySingleCase(expected, actual, query)
		}
		// If there is no given flags.
		return DoVerifyAccordingConfig()
	},
}

func verifySingleCase(expectedFile, actualFile, query string) error {
	expectedData, err := util.ReadFileContent(expectedFile)
	if err != nil {
		return fmt.Errorf("failed to read the expected data file: %v", err)
	}

	var actualData, sourceName, stderr string
	if actualFile != "" {
		sourceName = actualFile
		actualData, err = util.ReadFileContent(actualFile)
		if err != nil {
			return fmt.Errorf("failed to read the actual data file: %v", err)
		}
	} else if query != "" {
		sourceName = query
		actualData, stderr, err = util.ExecuteCommand(query)
		if err != nil {
			return fmt.Errorf("failed to execute the query: %s, output: %s, error: %v", query, actualData, stderr)
		}
	}

	if err = verifier.Verify(actualData, expectedData); err != nil {
		if me, ok := err.(*verifier.MismatchError); ok {
			return fmt.Errorf("failed to verify the output: %s, error:\n%v", sourceName, me.Error())
		}
		return fmt.Errorf("failed to verify the output: %s, error:\n%v", sourceName, err)
	}
	logger.Log.Infof("verified the output: %s", sourceName)
	return nil
}

// concurrentErrors store errors that occurred when verifying cases in goroutines.
type concurrentErrors struct {
	errs  *multierror.Error
	mutex sync.Mutex
	count int
}

// verifyInfo contains necessary information about verification
type verifyInfo struct {
	retryCount int
	interval   time.Duration
	failFast   bool
}

func concurrentSafeErrAppend(concurrentError *concurrentErrors, err error) {
	concurrentError.mutex.Lock()
	concurrentError.errs = multierror.Append(concurrentError.errs, err)
	concurrentError.mutex.Unlock()
}

func concurrentSafeCountLess(concurrentError *concurrentErrors) {
	concurrentError.mutex.Lock()
	concurrentError.count--
	concurrentError.mutex.Unlock()
}

func check(stopChan chan struct{}, concurrentErrors *concurrentErrors) error {
	for {
		if concurrentErrors.count == 0 {
			break
		}
		_, ok := <-stopChan
		if ok {
			return concurrentErrors.errs.ErrorOrNil()
		}
	}

	return nil
}

func concurrentVerifySingleCase(idx int, v config.VerifyCase, errs *concurrentErrors, verify verifyInfo, wg *sync.WaitGroup, stopChan chan struct{}) {
	var err error

	defer func() {
		if err != nil {
			concurrentSafeErrAppend(errs, err)
			if verify.failFast {
				stopChan <- struct{}{}
			}
		}
		if verify.failFast {
			concurrentSafeCountLess(errs)
		}
		wg.Done()
	}()

	if v.GetExpected() == "" {
		errMsg := fmt.Sprintf("the expected data file for case[%v] is not specified\n", idx)
		logger.Log.Warnf(errMsg)
		err = errors.New(errMsg)
		return
	}

	for current := 1; current <= verify.retryCount; current++ {
		if err = verifySingleCase(v.GetExpected(), v.GetActual(), v.Query); err == nil {
			break
		} else if current != verify.retryCount {
			logger.Log.Warnf("verify case[%v] failure, will continue retry, %v", idx, err)
			time.Sleep(verify.interval)
		}
	}
}

func checkForRetryCount(retryCount int) int {
	if retryCount <= 0 {
		return 1
	}

	return retryCount
}

// DoVerifyAccordingConfig reads cases from the config file and verifies them.
func DoVerifyAccordingConfig() error {
	if config.GlobalConfig.Error != nil {
		return config.GlobalConfig.Error
	}

	e2eConfig := config.GlobalConfig.E2EConfig

	retryCount := e2eConfig.Verify.RetryStrategy.Count
	retryCount = checkForRetryCount(retryCount)

	interval, err := parseInterval(e2eConfig.Verify.RetryStrategy.Interval)
	if err != nil {
		return err
	}

	failFast := e2eConfig.Verify.FailFast
	concurrency := e2eConfig.Verify.Concurrency

	var Errs *multierror.Error

	if concurrency {
		var waitGroup sync.WaitGroup
		ConcurrentErrors := concurrentErrors{
			errs:  Errs,
			count: len(e2eConfig.Verify.Cases),
		}

		VerifyInfo := verifyInfo{
			retryCount,
			interval,
			failFast,
		}
		stopChan := make(chan struct{})
		waitGroup.Add(len(e2eConfig.Verify.Cases))

		for idx, v := range e2eConfig.Verify.Cases {
			go concurrentVerifySingleCase(idx, v, &ConcurrentErrors, VerifyInfo, &waitGroup, stopChan)
		}

		if failFast {
			if err := check(stopChan, &ConcurrentErrors); err != nil {
				return err
			}
		}
		waitGroup.Wait()
		Errs = ConcurrentErrors.errs
	} else {
		for idx, v := range e2eConfig.Verify.Cases {
			if v.GetExpected() == "" {
				errMsg := fmt.Sprintf("the expected data file for case[%v] is not specified\n", idx)
				if failFast {
					return errors.New(errMsg)
				}
				logger.Log.Warnf(errMsg)
				Errs = multierror.Append(Errs, errors.New(errMsg))
				continue
			}

			for current := 1; current <= retryCount; current++ {
				if err := verifySingleCase(v.GetExpected(), v.GetActual(), v.Query); err == nil {
					break
				} else if current != retryCount {
					logger.Log.Warnf("verify case failure, will continue retry, %v", err)
					time.Sleep(interval)
				} else {
					if failFast {
						return err
					}
					Errs = multierror.Append(Errs, err)
				}
			}
		}
	}

	return Errs.ErrorOrNil()
}

// TODO remove this in 2.0.0
func parseInterval(retryInterval any) (time.Duration, error) {
	var interval time.Duration
	var err error
	switch itv := retryInterval.(type) {
	case int:
		logger.Log.Warnf(`configuring verify.retry.interval with number is deprecated
and will be removed in future version, please use Duration style instead, such as 10s, 1m.`)
		interval = time.Duration(itv) * time.Millisecond
	case string:
		if interval, err = time.ParseDuration(itv); err != nil {
			return 0, err
		}
	case nil:
		interval = 0
	default:
		return 0, fmt.Errorf("failed to parse verify.retry.interval: %v", retryInterval)
	}
	if interval < 0 {
		interval = 1 * time.Second
	}
	return interval, nil
}
