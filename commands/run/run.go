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
package run

import (
	"github.com/apache/skywalking-infra-e2e/commands/cleanup"
	"github.com/apache/skywalking-infra-e2e/commands/setup"
	"github.com/apache/skywalking-infra-e2e/commands/trigger"
	"github.com/apache/skywalking-infra-e2e/commands/verify"
	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/logger"

	"github.com/spf13/cobra"
)

var Run = &cobra.Command{
	Use:   "run",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := runAccordingE2E()
		if err != nil {
			return err
		}

		return nil
	},
}

func runAccordingE2E() error {
	if config.GlobalConfig.Error != nil {
		return config.GlobalConfig.Error
	}

	// setup part
	err := setup.DoSetupAccordingE2E()
	if err != nil {
		return err
	}
	logger.Log.Infof("setup part finished successfully")

	// cleanup part
	defer func() {
		err = cleanup.DoCleanupAccordingE2E()
		if err != nil {
			logger.Log.Errorf("cleanup part error: %s", err)
		}
		logger.Log.Infof("cleanup part finished successfully")
	}()

	// trigger part
	err = trigger.DoActionAccordingE2E()
	if err != nil {
		return err
	}
	logger.Log.Infof("trigger part finished successfully")

	// verify part
	err = verify.DoVerifyAccordingConfig()
	if err != nil {
		return err
	}
	logger.Log.Infof("verify part finished successfully")

	return nil
}
