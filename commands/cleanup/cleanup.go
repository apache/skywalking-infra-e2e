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
package cleanup

import (
	"fmt"

	"github.com/apache/skywalking-infra-e2e/internal/config"

	"github.com/apache/skywalking-infra-e2e/internal/components/cleanup"

	"github.com/spf13/cobra"

	"github.com/apache/skywalking-infra-e2e/internal/constant"
)

var Cleanup = &cobra.Command{
	Use:   "cleanup",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := DoCleanupAccordingE2E()
		if err != nil {
			err = fmt.Errorf("[Cleanup] %s", err)
			return err
		}
		return nil
	},
}

func DoCleanupAccordingE2E() error {
	e2eConfig := config.GlobalConfig.E2EConfig

	if e2eConfig.Setup.Env == constant.Kind {
		kubeConfigPath := e2eConfig.Setup.GetKubeconfig()
		// if there is an existing kubernetes cluster, don't delete the kind cluster.
		if kubeConfigPath == "" {
			err := cleanup.KindCleanUp(&e2eConfig)
			if err != nil {
				return err
			}
		}
	} else if e2eConfig.Setup.Env == constant.Compose {
		err := cleanup.ComposeCleanUp(&e2eConfig)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no such env for cleanup: [%s]. should use kind or compose instead", e2eConfig.Setup.Env)
	}

	return nil
}
