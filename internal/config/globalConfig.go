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
	"path"
	"path/filepath"

	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"

	"gopkg.in/yaml.v2"
)

// GlobalE2EConfig store E2EConfig which can be used globally.
type GlobalE2EConfig struct {
	Error     error
	E2EConfig E2EConfig
}

var GlobalConfig GlobalE2EConfig
var CfgFile string

func ReadGlobalConfigFile() {
	if !util.PathExist(CfgFile) {
		GlobalConfig.Error = fmt.Errorf("e2e config file %s not exist", CfgFile)
		return
	}

	data, err := ioutil.ReadFile(CfgFile)
	if err != nil {
		GlobalConfig.Error = fmt.Errorf("read e2e config file %s error: %s", CfgFile, err)
		return
	}

	e2eConfigObject := E2EConfig{}
	if err := yaml.Unmarshal(data, &e2eConfigObject); err != nil {
		GlobalConfig.Error = fmt.Errorf("unmarshal e2e config file %s error: %s", CfgFile, err)
		return
	}

	GlobalConfig.E2EConfig = e2eConfigObject
	GlobalConfig.Error = nil
	logger.Log.Info("load the e2e config successfully")
}

// ResolveAbs resolves the relative path (relative to CfgFile) to an absolute file path.
func ResolveAbs(p string) string {
	if p == "" {
		return p
	}

	if path.IsAbs(p) {
		return p
	}

	abs, err := filepath.Abs(CfgFile)
	if err != nil {
		logger.Log.Warnf("failed to resolve the absolute file path of %v\n", CfgFile)
		return p
	}

	return filepath.Join(filepath.Dir(abs), p)
}
