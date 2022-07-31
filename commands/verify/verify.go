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

// ErrContain has two fields. The errAddr field is a pointer points to Err. Err's type is "multierror.Error".
// When an error occurs during concurrently verifying cases, the process will append the new error to Err.Errors.
// The Err.Errors is []error, which contains all errors occurring during the verification. errAddr points to Err.
type ErrContain struct {
	errAddr *multierror.Error
	mutex   sync.Mutex
}

// verifyInfo contains necessary information about verification
type verifyInfo struct {
	retryCount int
	interval   time.Duration
	failFast   bool
}

func concurrentSafeErrAppend(errContain *ErrContain, err error) {
	errContain.mutex.Lock()
	errContain.errAddr = multierror.Append(errContain.errAddr, err)
	errContain.mutex.Unlock()
}

func concurrentVerifySingleCase(idx int, v config.VerifyCase, errContain *ErrContain, verify verifyInfo, wg *sync.WaitGroup, chanBool chan bool) {
	if v.GetExpected() == "" {
		errMsg := fmt.Sprintf("the expected data file for case[%v] is not specified\n", idx)
		logger.Log.Warnf(errMsg)
		concurrentSafeErrAppend(errContain, errors.New(errMsg))
		if verify.failFast {
			chanBool <- true
		}
		wg.Done()
		return
	}

	for current := 1; current <= verify.retryCount; current++ {
		if err := verifySingleCase(v.GetExpected(), v.GetActual(), v.Query); err == nil {
			break
		} else if current != verify.retryCount {
			logger.Log.Warnf("verify case[%v] failure, will continue retry, %v", idx, err)
			time.Sleep(verify.interval)
		} else {
			concurrentSafeErrAppend(errContain, err)
			if verify.failFast {
				chanBool <- true
			}
		}
	}
	wg.Done()
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

	var errCollection multierror.Error

	errContain := ErrContain{
		errAddr: &errCollection,
	}

	if concurrency {
		var waitGroup sync.WaitGroup
		VerifyInfo := verifyInfo{
			retryCount,
			interval,
			failFast,
		}
		chanBool := make(chan bool, 1)
		waitGroup.Add(len(e2eConfig.Verify.Cases))

		for idx, v := range e2eConfig.Verify.Cases {
			go concurrentVerifySingleCase(idx, v, &errContain, VerifyInfo, &waitGroup, chanBool)
		}

		if failFast {
			for {
				result := <-chanBool
				if result {
					return errContain.errAddr.ErrorOrNil()
				}
			}
		}

		waitGroup.Wait()
	} else {
		for idx, v := range e2eConfig.Verify.Cases {
			if v.GetExpected() == "" {
				errMsg := fmt.Sprintf("the expected data file for case[%v] is not specified\n", idx)
				if failFast {
					return errors.New(errMsg)
				}
				logger.Log.Warnf(errMsg)
				errContain.errAddr = multierror.Append(errContain.errAddr, errors.New(errMsg))
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
					errContain.errAddr = multierror.Append(errContain.errAddr, err)
				}
			}
		}
	}

	return errContain.errAddr.ErrorOrNil()
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
