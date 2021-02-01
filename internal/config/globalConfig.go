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

	"github.com/apache/skywalking-infra-e2e/internal/logger"

	"gopkg.in/yaml.v2"

	"github.com/apache/skywalking-infra-e2e/internal/util"
)

// GlobalE2EConfig store E2EConfig which can be used globally.
type GlobalE2EConfig struct {
	Ready     bool
	E2EConfig E2EConfig
}

var GlobalConfig GlobalE2EConfig

func ReadGlobalConfigFile(configFilePath string) error {
	if GlobalConfig.Ready {
		logger.Log.Info("e2e config has been initialized")
		return nil
	}

	e2eFile := configFilePath
	if util.PathExist(e2eFile) {
		// other command should check if global config is ready.
		data, err := ioutil.ReadFile(e2eFile)
		if err != nil {
			return fmt.Errorf("read e2e config file %s error: %s", e2eFile, err)
		}
		e2eConfigObject := E2EConfig{}
		err = yaml.Unmarshal(data, &e2eConfigObject)
		if err != nil {
			return fmt.Errorf("unmarshal e2e config file %s error: %s", e2eFile, err)
		}
		GlobalConfig.E2EConfig = e2eConfigObject
		GlobalConfig.Ready = true
	} else {
		return fmt.Errorf("e2e config file %s not exist", e2eFile)
	}

	if !GlobalConfig.Ready {
		return fmt.Errorf("e2e config read failed")
	}

	return nil
}
