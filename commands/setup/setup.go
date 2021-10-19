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
	"github.com/apache/skywalking-infra-e2e/internal/util"

	"github.com/spf13/cobra"
)

var Setup = &cobra.Command{
	Use:   "setup",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := util.CheckDockerDaemon(); err != nil {
			return err
		}

		if err := DoSetupAccordingE2E(); err != nil {
			return fmt.Errorf("[Setup] %s", err)
		}
		return nil
	},
}

func DoSetupAccordingE2E() error {
	if config.GlobalConfig.Error != nil {
		return config.GlobalConfig.Error
	}

	e2eConfig := config.GlobalConfig.E2EConfig
	useCommand := config.GlobalConfig.UseCommand

	if e2eConfig.Setup.Env == constant.Kind {
		err := setup.KindSetup(&e2eConfig, useCommand)
		if err != nil {
			return err
		}
	} else if e2eConfig.Setup.Env == constant.Compose {
		err := setup.ComposeSetup(&e2eConfig)
		if err != nil {
			return err
		}
		return nil
	} else {
		return fmt.Errorf("no such env for setup: [%s]. should use kind or compose instead", e2eConfig.Setup.Env)
	}

	return nil
}
