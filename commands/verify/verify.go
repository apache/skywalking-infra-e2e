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
	"fmt"
	"sync"
	"time"

	"github.com/apache/skywalking-infra-e2e/internal/components/verifier"
	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"

	"github.com/pterm/pterm"
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
	return nil
}

// verifyInfo contains necessary information about verification
type verifyInfo struct {
	caseNumber int
	retryCount int
	interval   time.Duration
	failFast   bool
	outputChan chan caseInfo
}

type summary struct {
	errNum     int
	successNum int
}

type caseInfo struct {
	IfSuccess bool
	msg       string
	err       error
}

func concurrentOutput(outputChan chan caseInfo, stopChan chan bool, failFast bool, wg *sync.WaitGroup, caseNumber int, s *summary, errExist *bool) {
	defer wg.Done()
	var number int
	spinnerLiveText, _ := pterm.DefaultSpinner.WithShowTimer(false).Start()
	pterm.Error.Prefix = pterm.Prefix{
		Text:  "DETAILS",
		Style: &pterm.ThemeDefault.ErrorPrefixStyle,
	}

	for caseInfo := range outputChan {
		number++

		if caseInfo.IfSuccess {
			s.successNum++
			spinnerLiveText.Success(caseInfo.msg)
		} else {
			*errExist = true
			s.errNum++
			spinnerLiveText.Warning(caseInfo.msg)
			spinnerLiveText.Fail(caseInfo.err)
			if failFast {
				stopChan <- true
				return
			}
		}

		if number == caseNumber {
			if failFast {
				stopChan <- false
			}
			return
		}
	}
}

func concurrentVerifySingleCase(idx int, v *config.VerifyCase, verify *verifyInfo, wg *sync.WaitGroup) {
	var msg string
	var err error
	defer wg.Done()

	if v.GetExpected() == "" {
		msg = fmt.Sprintf("failed to verify %v:", caseName(v.Name, idx))
		err = fmt.Errorf("the expected data file for %v is not specified", caseName(v.Name, idx))
		info := caseInfo{
			false,
			msg,
			err,
		}
		verify.outputChan <- info
		return
	}

	for current := 0; current <= verify.retryCount; current++ {
		if err = verifySingleCase(v.GetExpected(), v.GetActual(), v.Query); err == nil {
			if current == 0 {
				msg = fmt.Sprintf("verified %v\n", caseName(v.Name, idx))
			} else {
				msg = fmt.Sprintf("verified %v, retried %d time(s)\n", caseName(v.Name, idx), current)
			}
			info := caseInfo{
				true,
				msg,
				nil,
			}
			verify.outputChan <- info
			return
		} else if current != verify.retryCount {
			time.Sleep(verify.interval)
		} else {
			msg := fmt.Sprintf("failed to verify %v, retried %d time(s):", caseName(v.Name, idx), current)
			info := caseInfo{
				false,
				msg,
				err,
			}
			verify.outputChan <- info
		}
	}
}

func checkForRetryCount(retryCount int) int {
	if retryCount <= 0 {
		return 0
	}
	return retryCount
}

// outputSummary outputs a summary of verify result
func outputSummary(summary *summary, total int) {
	pterm.Info.Prefix = pterm.Prefix{
		Text:  "SUMMARY",
		Style: &pterm.ThemeDefault.InfoPrefixStyle,
	}
	pterm.Info.WithMessageStyle(&pterm.Style{pterm.FgGreen}).Println(fmt.Sprintf("%d passed", summary.successNum))
	pterm.Info.Prefix = pterm.Prefix{
		Text:  "       ",
		Style: &pterm.ThemeDefault.InfoPrefixStyle,
	}
	pterm.Info.WithMessageStyle(&pterm.Style{pterm.FgLightRed}).Println(fmt.Sprintf("%d failed", summary.errNum))
	pterm.Info.WithMessageStyle(&pterm.Style{pterm.FgYellow}).Println(fmt.Sprintf("%d skipped", total-summary.errNum-summary.successNum))
	fmt.Println()
}

func caseName(name string, idx int) string {
	if name == "" {
		return fmt.Sprintf("case[%d]", idx+1)
	}
	return name
}

func outputSuccess(spinnerLiveText *pterm.SpinnerPrinter, current int, caseName string) {
	if current == 0 {
		spinnerLiveText.Success(fmt.Sprintf("verified %v \n", caseName))
	} else {
		spinnerLiveText.Success(fmt.Sprintf("verified %v, retried %d time(s)\n", caseName, current))
	}
}

func outputRetryInfo(spinnerLiveText *pterm.SpinnerPrinter, current int, caseName string, retryCount int, interval time.Duration) {
	if current == 0 {
		spinnerLiveText.UpdateText(fmt.Sprintf("failed to verify %v, will continue retry:", caseName))
		time.Sleep(time.Second * 2)
	} else {
		spinnerLiveText.UpdateText(fmt.Sprintf("failed to verify %v, retry [%d/%d]", caseName, current, retryCount))
		time.Sleep(interval)
	}
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
	summary := &summary{}
	caseNumber := len(e2eConfig.Verify.Cases)
	var errorExist bool

	if concurrency {
		var waitGroup sync.WaitGroup
		stopChan := make(chan bool)
		outputChan := make(chan caseInfo)
		VerifyInfo := verifyInfo{
			caseNumber,
			retryCount,
			interval,
			failFast,
			outputChan,
		}

		waitGroup.Add(caseNumber + 1)
		go concurrentOutput(outputChan, stopChan, failFast, &waitGroup, caseNumber, summary, &errorExist)
		for idx := range e2eConfig.Verify.Cases {
			go concurrentVerifySingleCase(idx, &e2eConfig.Verify.Cases[idx], &VerifyInfo, &waitGroup)
		}

		if failFast {
			if stop := <-stopChan; stop {
				outputSummary(summary, caseNumber)
				return fmt.Errorf("failed to verify all the cases, an error occurred")
			}
		}
		waitGroup.Wait()
	} else {
		for idx, v := range e2eConfig.Verify.Cases {
			spinnerLiveText, _ := pterm.DefaultSpinner.WithShowTimer(false).Start()
			pterm.Error.Prefix = pterm.Prefix{
				Text:  "DETAILS",
				Style: &pterm.ThemeDefault.ErrorPrefixStyle,
			}
			if v.GetExpected() == "" {
				errMsg := fmt.Sprintf("failed to verify %v", caseName(v.Name, idx))
				spinnerLiveText.Warning(errMsg)
				spinnerLiveText.Fail(fmt.Sprintf("the expected data file for %v is not specified\n", caseName(v.Name, idx)))
				summary.errNum++
				if failFast {
					outputSummary(summary, caseNumber)
					return fmt.Errorf("failed to verify all the cases, an error occurred")
				}
				continue
			}
			for current := 0; current <= retryCount; current++ {
				if err := verifySingleCase(v.GetExpected(), v.GetActual(), v.Query); err == nil {
					summary.successNum++
					outputSuccess(spinnerLiveText, current, caseName(v.Name, idx))
					break
				} else if current != retryCount {
					outputRetryInfo(spinnerLiveText, current, caseName(v.Name, idx), retryCount, interval)
				} else {
					errorExist = true
					summary.errNum++
					spinnerLiveText.UpdateText(fmt.Sprintf("failed to verify %v, retry [%d/%d]", caseName(v.Name, idx), current, retryCount))
					time.Sleep(time.Second)
					spinnerLiveText.Warning(fmt.Sprintf("failed to verify %v, retried %d time(s):", caseName(v.Name, idx), current))
					spinnerLiveText.Fail(err)
					fmt.Println()
					if failFast {
						outputSummary(summary, caseNumber)
						return fmt.Errorf("failed to verify all the cases, an error occurred")
					}
				}
			}
		}
	}
	outputSummary(summary, caseNumber)
	if errorExist {
		return fmt.Errorf("failed to verify all the cases")
	}
	return nil
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
