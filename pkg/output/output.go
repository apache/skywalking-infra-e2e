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
	Format  string
	Formats = []string{"yaml"}
)

type YamlCaseResult struct {
	Passed  []string
	Failed  []string
	Skipped []string
}

func FormatIsNotExist() bool {
	for _, format := range Formats {
		if Format == format {
			return false
		}
	}

	return true
}

func PrintResult(caseRes []*CaseResult) {
	switch Format {
	case "yaml":
		PrintResultInYAML(caseRes)
	}
}

func PrintResultInYAML(caseRes []*CaseResult) {
	var yamlCaseResult YamlCaseResult
	for _, cr := range caseRes {
		if !cr.Skip {
			if cr.Err == nil {
				yamlCaseResult.Passed = append(yamlCaseResult.Passed, cr.CaseName)
			} else {
				yamlCaseResult.Failed = append(yamlCaseResult.Failed, cr.CaseName)
			}
		} else {
			yamlCaseResult.Skipped = append(yamlCaseResult.Skipped, cr.CaseName)
		}
	}

	yaml, _ := yaml.Marshal(yamlCaseResult)
	fmt.Print(string(yaml))
}
