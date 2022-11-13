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

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apache/skywalking-infra-e2e/internal/constant"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"

	"gopkg.in/yaml.v2"
)

// GlobalE2EConfig stores E2EConfig which can be used globally.
type GlobalE2EConfig struct {
	Error     error
	E2EConfig E2EConfig
}

var GlobalConfig GlobalE2EConfig

func init() {
	if os.Getenv("CI") == "true" {
		GlobalConfig.E2EConfig.Cleanup.On = constant.CleanUpAlways
	} else {
		GlobalConfig.E2EConfig.Cleanup.On = constant.CleanUpOnSuccess
	}

	GlobalConfig.E2EConfig.Verify.FailFast = true
}

func ReadGlobalConfigFile() {
	if !util.PathExist(util.CfgFile) {
		GlobalConfig.Error = fmt.Errorf("e2e config file %s not exist", util.CfgFile)
		return
	}

	data, err := os.ReadFile(util.CfgFile)
	if err != nil {
		GlobalConfig.Error = fmt.Errorf("read e2e config file %s error: %s", util.CfgFile, err)
		return
	}

	if err := yaml.Unmarshal(data, &GlobalConfig.E2EConfig); err != nil {
		GlobalConfig.Error = fmt.Errorf("unmarshal e2e config file %s error: %s", util.CfgFile, err)
		return
	}

	// convert verify
	if err := convertVerify(&GlobalConfig.E2EConfig.Verify); err != nil {
		GlobalConfig.Error = err
		return
	}

	if err := GlobalConfig.E2EConfig.Setup.Finalize(); err != nil {
		GlobalConfig.Error = err
	}

	GlobalConfig.Error = nil
	logger.Log.Info("load the e2e config successfully")
}

func convertVerify(verify *Verify) error {
	// convert cases
	result := make([]VerifyCase, 0)
	cfgAbsPath, _ := filepath.Abs(util.CfgFile)
	for idx := range verify.Cases {
		cases, err := convertSingleCase(&verify.Cases[idx], cfgAbsPath)
		if err != nil {
			return err
		}
		result = append(result, cases...)
	}
	verify.Cases = result
	return nil
}

func convertSingleCase(verifyCase *VerifyCase, baseFile string) ([]VerifyCase, error) {
	if len(verifyCase.Includes) > 0 && (verifyCase.Expected != "" || verifyCase.Query != "") {
		return nil, fmt.Errorf("include and query/expected only support selecting one of them in a case")
	}
	if len(verifyCase.Includes) == 0 {
		// using base path to resolve case paths
		if verifyCase.Expected != "" {
			verifyCase.Expected = util.ResolveAbsWithBase(verifyCase.Expected, baseFile)
		}
		if verifyCase.Actual != "" {
			verifyCase.Actual = util.ResolveAbsWithBase(verifyCase.Actual, baseFile)
		}
		return []VerifyCase{*verifyCase}, nil
	}
	result := make([]VerifyCase, 0)
	for _, include := range verifyCase.Includes {
		includePath := util.ResolveAbsWithBase(include, baseFile)

		if !util.PathExist(includePath) {
			return nil, fmt.Errorf("reuse case config file %s not exist", includePath)
		}

		data, err := os.ReadFile(includePath)
		if err != nil {
			return nil, fmt.Errorf("reuse case config file %s error: %s", includePath, err)
		}

		r := &ReusingCases{}
		if err := yaml.Unmarshal(data, r); err != nil {
			return nil, fmt.Errorf("unmarshal reuse case config file %s error: %s", includePath, err)
		}

		for idx := range r.Cases {
			// using include file path as base path to resolve cases
			cases, err := convertSingleCase(&r.Cases[idx], includePath)
			if err != nil {
				return nil, err
			}
			result = append(result, cases...)
		}
	}
	return result, nil
}
