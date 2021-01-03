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
// Kind, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package setup

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/apache/skywalking-infra-e2e/internal/components/setup"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"

	"github.com/apache/skywalking-infra-e2e/internal/flags"
)

func init() {
	Setup.Flags().StringVar(&flags.Env, "env", "kind", "specify the run environment")
	Setup.Flags().StringVar(&flags.File, "file", "kind.yaml", "specify configuration file")
}

var Setup = &cobra.Command{
	Use:   "setup",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flags.Env == setup.Compose {
			if util.Which(setup.ComposeCommand) != nil {
				return fmt.Errorf("command %s not found in the PATH", setup.ComposeCommand)
			}
			logger.Log.Info("env for docker-compose not implemented")
		} else if flags.Env == setup.Kind {
			if err := setup.KindSetupInCommand(); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("no such env for setup: [%s]. should use kind or compose instead", flags.Env)
		}
		return nil
	},
}
