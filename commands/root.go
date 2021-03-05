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
//
package commands

import (
	"github.com/apache/skywalking-infra-e2e/internal/util"
	"github.com/spf13/cobra"

	"github.com/apache/skywalking-infra-e2e/commands/cleanup"
	"github.com/apache/skywalking-infra-e2e/commands/run"
	"github.com/apache/skywalking-infra-e2e/commands/setup"
	"github.com/apache/skywalking-infra-e2e/commands/trigger"
	"github.com/apache/skywalking-infra-e2e/commands/verify"
	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/constant"
)

// Root represents the base command when called without any subcommands
var Root = &cobra.Command{
	Use:           "e2e command [flags]",
	Short:         "The next generation End-to-End Testing framework",
	Version:       version,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.ReadGlobalConfigFile()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	Root.AddCommand(run.Run)
	Root.AddCommand(setup.Setup)
	Root.AddCommand(trigger.Trigger)
	Root.AddCommand(verify.Verify)
	Root.AddCommand(cleanup.Cleanup)

	Root.PersistentFlags().StringVarP(&util.CfgFile, "config", "c", constant.E2EDefaultFile, "the config file")

	return Root.Execute()
}
