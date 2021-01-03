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
	"github.com/apache/skywalking-infra-e2e/internal/components/setup"
	"github.com/apache/skywalking-infra-e2e/internal/logger"

	"github.com/spf13/cobra"

	"github.com/apache/skywalking-infra-e2e/internal/flags"
)

func init() {
	Setup.Flags().StringVar(&flags.Env, "env", "kind", "specify the run environment")
	Setup.Flags().StringVar(&flags.File, "file", "kind.yaml", "specify configuration file")
}

var Setup = &cobra.Command{
	Use:   "setup",
	Short: "",
	Run: func(cmd *cobra.Command, args []string) {
		// check if env commands in PATH
		if flags.Env == setup.COMPOSE {
			if setup.Which(setup.COMPOSECOMMAND) != nil {
				logger.Log.Errorf("command %s not found, is it in the PATH?", setup.COMPOSECOMMAND)
			}
			logger.Log.Info("env for docker-compose not implemented")
		} else if flags.Env == setup.KIND {
			if setup.Which(setup.KINDCOMMAND) != nil {
				logger.Log.Errorf("command %s not found, is it in the PATH?", setup.COMPOSECOMMAND)
			}
			setup.KindSetupInCommand()
		} else {
			logger.Log.Errorf("No such env for setup: [%s]. Should use kind or compose instead", flags.Env)
		}
	},
}
