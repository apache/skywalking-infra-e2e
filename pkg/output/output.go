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

package output

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

var (
	SummaryOnly      bool
	Format           string
	SupportedFormats = map[string]struct{}{
		"yaml": {},
	}
)

type YamlCaseResult struct {
	Passed       []string
	Failed       []string
	Skipped      []string
	PassedCount  int `yaml:"passedCount"`
	FailedCount  int `yaml:"failedCount"`
	SkippedCount int `yaml:"skippedCount"`
}

func HasFormat() bool {
	_, ok := SupportedFormats[Format]
	return ok
}

func PrintResult(caseRes []*CaseResult) {
	if Format == "yaml" {
		printResultInYAML(caseRes)
	}
}

func printResultInYAML(caseRes []*CaseResult) {
	var yamlCaseResult YamlCaseResult
	for _, cr := range caseRes {
		if !cr.Skip {
			if cr.Err == nil {
				yamlCaseResult.Passed = append(yamlCaseResult.Passed, cr.Name)
			} else {
				yamlCaseResult.Failed = append(yamlCaseResult.Failed, cr.Name)
			}
		} else {
			yamlCaseResult.Skipped = append(yamlCaseResult.Skipped, cr.Name)
		}
	}

	yamlCaseResult.PassedCount = len(yamlCaseResult.Passed)
	yamlCaseResult.FailedCount = len(yamlCaseResult.Failed)
	yamlCaseResult.SkippedCount = len(yamlCaseResult.Skipped)

	yamlCaseResultData, _ := yaml.Marshal(yamlCaseResult)
	fmt.Println(string(yamlCaseResultData))
}
