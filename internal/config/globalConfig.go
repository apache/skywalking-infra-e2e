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
	"io/ioutil"
	"os"

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
}

func ReadGlobalConfigFile() {
	if !util.PathExist(util.CfgFile) {
		GlobalConfig.Error = fmt.Errorf("e2e config file %s not exist", util.CfgFile)
		return
	}

	data, err := ioutil.ReadFile(util.CfgFile)
	if err != nil {
		GlobalConfig.Error = fmt.Errorf("read e2e config file %s error: %s", util.CfgFile, err)
		return
	}

	if err := yaml.Unmarshal(data, &GlobalConfig.E2EConfig); err != nil {
		GlobalConfig.Error = fmt.Errorf("unmarshal e2e config file %s error: %s", util.CfgFile, err)
		return
	}

	GlobalConfig.Error = nil
	logger.Log.Info("load the e2e config successfully")
}
