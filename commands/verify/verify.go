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

// verifyInfo contains necessary information about verification
type verifyInfo struct {
	caseNumber int
	retryCount int
	interval   time.Duration
	failFast   bool
}

type Summary struct {
	errNum     int
	successNum int
}

type CaseInfo struct {
	msg string
	err error
}

type OutputInfo struct {
	writeLock sync.Mutex
	casesInfo []CaseInfo
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

func concurrentlyVerifySingleCase(v *config.VerifyCase, verifyInfo *verifyInfo, wg *sync.WaitGroup, outputInfo *OutputInfo, stopChan chan bool) {
	var msg string
	var err error
	var caseInfo CaseInfo
	defer func() {
		caseInfo = CaseInfo{
			msg,
			err,
		}
		outputInfo.writeLock.Lock()
		outputInfo.casesInfo = append(outputInfo.casesInfo, caseInfo)
		outputInfo.writeLock.Unlock()
		if verifyInfo.failFast {
			if err != nil {
				stopChan <- true
			} else {
				stopChan <- false
			}
		}
		wg.Done()
	}()

	if v.GetExpected() == "" {
		msg = fmt.Sprintf("failed to verify %v:", caseName(v))
		err = fmt.Errorf("the expected data file for %v is not specified", caseName(v))
		return
	}

	for current := 0; current <= verifyInfo.retryCount; current++ {
		if err = verifySingleCase(v.GetExpected(), v.GetActual(), v.Query); err == nil {
			if current == 0 {
				msg = fmt.Sprintf("verified %v\n", caseName(v))
			} else {
				msg = fmt.Sprintf("verified %v, retried %d time(s)\n", caseName(v), current)
			}
			return
		} else if current != verifyInfo.retryCount {
			time.Sleep(verifyInfo.interval)
		} else {
			msg = fmt.Sprintf("failed to verify %v, retried %d time(s):", caseName(v), current)
		}
	}
}

// verifyCasesConcurrently verifies the cases concurrently.
func verifyCasesConcurrently(verify *config.Verify, verifyInfo *verifyInfo) error {
	summary := Summary{}
	var waitGroup sync.WaitGroup
	stopChan := make(chan bool)
	waitGroup.Add(verifyInfo.caseNumber)
	outputInfo := OutputInfo{}
	for idx := range verify.Cases {
		go concurrentlyVerifySingleCase(&verify.Cases[idx], verifyInfo, &waitGroup, &outputInfo, stopChan)
	}

	if verifyInfo.failFast {
		if shouldExit(stopChan, verifyInfo.caseNumber) {
			outputResult(&outputInfo, &summary)
			outputSummary(&summary, verifyInfo.caseNumber)
			return fmt.Errorf("failed to verify one case")
		}
	}
	waitGroup.Wait()
	outputResult(&outputInfo, &summary)
	outputSummary(&summary, verifyInfo.caseNumber)
	if summary.errNum > 0 {
		return fmt.Errorf("failed to verify %d case(s)", summary.errNum)
	}
	return nil
}

// verifyCasesSerially verifies the cases serially.
func verifyCasesSerially(verify *config.Verify, verifyInfo *verifyInfo) error {
	summary := Summary{}
	for idx := range verify.Cases {
		v := &verify.Cases[idx]
		spinnerLiveText, _ := pterm.DefaultSpinner.WithShowTimer(false).Start()
		spinnerLiveText.MessageStyle = &pterm.Style{pterm.FgCyan}
		pterm.Error.Prefix = pterm.Prefix{
			Text:  "DETAILS",
			Style: &pterm.ThemeDefault.ErrorPrefixStyle,
		}

		if v.GetExpected() == "" {
			errMsg := fmt.Sprintf("failed to verify %v", caseName(v))
			spinnerLiveText.Warning(errMsg)
			spinnerLiveText.Fail(fmt.Sprintf("the expected data file for %v is not specified\n", caseName(v)))
			summary.errNum++
			if verifyInfo.failFast {
				outputSummary(&summary, verifyInfo.caseNumber)
				return fmt.Errorf("failed to verify one case")
			}
			continue
		}

		for current := 0; current <= verifyInfo.retryCount; current++ {
			if err := verifySingleCase(v.GetExpected(), v.GetActual(), v.Query); err == nil {
				summary.successNum++
				if current == 0 {
					spinnerLiveText.Success(fmt.Sprintf("verified %v \n", caseName(v)))
				} else {
					spinnerLiveText.Success(fmt.Sprintf("verified %v, retried %d time(s)\n", caseName(v), current))
				}
				break
			} else if current != verifyInfo.retryCount {
				if current == 0 {
					spinnerLiveText.UpdateText(fmt.Sprintf("failed to verify %v, will continue retry:", caseName(v)))
					time.Sleep(time.Second * 2)
				} else {
					spinnerLiveText.UpdateText(fmt.Sprintf("failed to verify %v, retry [%d/%d]", caseName(v), current, verifyInfo.retryCount))
					time.Sleep(verifyInfo.interval)
				}
			} else {
				summary.errNum++
				spinnerLiveText.UpdateText(fmt.Sprintf("failed to verify %v, retry [%d/%d]", caseName(v), current, verifyInfo.retryCount))
				time.Sleep(time.Second)
				spinnerLiveText.Warning(fmt.Sprintf("failed to verify %v, retried %d time(s):", caseName(v), current))
				spinnerLiveText.Fail(err)
				fmt.Println()
				if verifyInfo.failFast {
					outputSummary(&summary, verifyInfo.caseNumber)
					return fmt.Errorf("failed to verify one case, an error occurred")
				}
			}
		}
	}

	outputSummary(&summary, verifyInfo.caseNumber)
	if summary.errNum > 0 {
		return fmt.Errorf("failed to verify %d case(s)", summary.errNum)
	}
	return nil
}

// outputSummary outputs a summary of verify result. The summary shows the number of the successful, failed and skipped cases.
func outputSummary(summary *Summary, total int) {
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

// outputResult outputs the result of cases.
func outputResult(outputInfo *OutputInfo, summary *Summary) {
	pterm.Error.Prefix = pterm.Prefix{
		Text:  "DETAILS",
		Style: &pterm.ThemeDefault.ErrorPrefixStyle,
	}
	for _, caseInfo := range outputInfo.casesInfo {
		if caseInfo.err == nil {
			summary.successNum++
			pterm.DefaultSpinner.Success(caseInfo.msg)
		} else {
			summary.errNum++
			pterm.DefaultSpinner.Warning(caseInfo.msg)
			pterm.DefaultSpinner.Fail(caseInfo.err)
		}
	}
}

func caseName(v *config.VerifyCase) string {
	if v.Name == "" {
		if v.Actual != "" {
			return fmt.Sprintf("case[%s]", v.Actual)
		}
		return fmt.Sprintf("case[%s]", v.Query)
	}
	return v.Name
}

func shouldExit(stopChan chan bool, goroutineNum int) bool {
	count := 0
	for shouldExit := range stopChan {
		count++

		if shouldExit {
			return true
		}

		if count == goroutineNum {
			break
		}
	}
	return false
}

// DoVerifyAccordingConfig reads cases from the config file and verifies them.
func DoVerifyAccordingConfig() error {
	if config.GlobalConfig.Error != nil {
		return config.GlobalConfig.Error
	}
	e2eConfig := config.GlobalConfig.E2EConfig
	retryCount := e2eConfig.Verify.RetryStrategy.Count
	if retryCount <= 0 {
		retryCount = 0
	}
	interval, err := parseInterval(e2eConfig.Verify.RetryStrategy.Interval)
	if err != nil {
		return err
	}
	failFast := e2eConfig.Verify.FailFast
	caseNumber := len(e2eConfig.Verify.Cases)

	VerifyInfo := verifyInfo{
		caseNumber,
		retryCount,
		interval,
		failFast,
	}

	concurrency := e2eConfig.Verify.Concurrency
	if concurrency {
		return verifyCasesConcurrently(&e2eConfig.Verify, &VerifyInfo)
	}

	return verifyCasesSerially(&e2eConfig.Verify, &VerifyInfo)
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
