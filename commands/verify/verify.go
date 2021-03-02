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
	"github.com/apache/skywalking-infra-e2e/internal/components/verifier"
	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"

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

var Verify = &cobra.Command{
	Use:   "verify",
	Short: "verify if the actual data match the expected data",
	RunE: func(cmd *cobra.Command, args []string) error {
		if expected != "" {
			return verifySingleCase(expected, actual, query)
		}
		// If there is no given flags.
		return verifyAccordingConfig()
	},
}

func verifySingleCase(expectedFile, actualFile, query string) error {
	expectedData, err := util.ReadFileContent(expectedFile)
	if err != nil {
		logger.Log.Error("failed to read the expected data file")
		return err
	}

	if actualFile != "" {
		if err = verifier.VerifyDataFile(actualFile, expectedData); err != nil {
			logger.Log.Warnf("failed to verify the output: %s\n", actualFile)
		} else {
			logger.Log.Infof("verified the output: %s\n", actualFile)
		}
	} else if query != "" {
		if err = verifier.VerifyQuery(query, expectedData); err != nil {
			logger.Log.Warnf("failed to verify the output: %s\n", query)
		} else {
			logger.Log.Infof("verified the output: %s\n", query)
		}
	}
	return nil
}

func verifyAccordingConfig() error {
	if config.GlobalConfig.Error != nil {
		return config.GlobalConfig.Error
	}

	e2eConfig := config.GlobalConfig.E2EConfig

	for _, v := range e2eConfig.Verify {
		if v.Expected != "" {
			verifySingleCase(v.Expected, v.Actual, v.Query)
		} else {
			logger.Log.Error("the expected data file is not specified")
		}
	}
	return nil
}
