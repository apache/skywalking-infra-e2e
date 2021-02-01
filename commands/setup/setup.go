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

	"github.com/apache/skywalking-infra-e2e/internal/config"

	"github.com/apache/skywalking-infra-e2e/internal/constant"

	"github.com/spf13/cobra"

	"github.com/apache/skywalking-infra-e2e/internal/components/setup"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"

	"github.com/apache/skywalking-infra-e2e/internal/flags"
)

func init() {
	Setup.Flags().StringVar(&flags.Env, "env", "", "specify test environment")
	Setup.Flags().StringVar(&flags.File, "file", "", "specify configuration file")
	Setup.Flags().StringVar(&flags.Manifests, "manifests", "", "specify the resources files/directories to apply")
	Setup.Flags().StringVar(&flags.WaitFor, "wait-for", "", "specify the wait-for strategy")
}

var Setup = &cobra.Command{
	Use:   "setup",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		var setUpError error
		if flags.Env == constant.Compose {
			if util.Which(constant.ComposeCommand) != nil {
				setUpError = fmt.Errorf("command %s not found in the PATH", constant.ComposeCommand)
			}
			logger.Log.Info("env for docker-compose not implemented")
		} else if flags.Env == constant.Kind {
			if err := setup.KindSetupInCommand(); err != nil {
				setUpError = err
			}
		} else if flags.Env == "" {
			// setup according e2e.yaml
			err := setupAccordingE2E()
			if err != nil {
				setUpError = err
			}
		} else {
			setUpError = fmt.Errorf("no such env for setup: [%s]. should use kind or compose instead", flags.Env)
		}

		if setUpError != nil {
			setUpError = fmt.Errorf("[Setup] %s", setUpError)
			return setUpError
		}

		return nil
	},
}

func setupAccordingE2E() error {
	err := config.ReadGlobalConfigFile(constant.E2EDefaultFile)
	if err != nil {
		return err
	}

	e2eConfig := config.GlobalConfig.E2EConfig

	if e2eConfig.Setup.Env == constant.Kind {
		err := setup.KindSetup(&e2eConfig)
		return err
	} else if e2eConfig.Setup.Env == constant.Compose {
		logger.Log.Info("env for docker-compose not implemented")
	} else {
		return fmt.Errorf("no such env for setup: [%s]. should use kind or compose instead", e2eConfig.Setup.Env)
	}

	return nil
}
