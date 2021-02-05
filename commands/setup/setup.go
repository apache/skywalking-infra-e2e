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

package setup

import (
	"fmt"

	"github.com/apache/skywalking-infra-e2e/internal/components/setup"
	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/constant"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"

	"github.com/spf13/cobra"
)

var Setup = &cobra.Command{
	Use:   "setup",
	Short: "",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := util.CheckDockerDaemon()
		if err != nil {
			return err
		}

		err = config.ReadGlobalConfigFile(constant.E2EDefaultFile)
		if err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := setupAccordingE2E()
		if err != nil {
			err = fmt.Errorf("[Setup] %s", err)
			return err
		}
		return nil
	},
}

func setupAccordingE2E() error {
	e2eConfig := config.GlobalConfig.E2EConfig

	if e2eConfig.Setup.Env == constant.Kind {
		err := setup.KindSetup(&e2eConfig)
		if err != nil {
			return err
		}
	} else if e2eConfig.Setup.Env == constant.Compose {
		logger.Log.Warn("env for docker-compose not implemented")
		return nil
	} else {
		return fmt.Errorf("no such env for setup: [%s]. should use kind or compose instead", e2eConfig.Setup.Env)
	}

	return nil
}
