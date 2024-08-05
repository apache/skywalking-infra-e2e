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

package setup

import (
	"context"
	"fmt"

	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/logger"

	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ComposeV2Setup sets up environment according to e2e.yaml.
func ComposeV2Setup(e2eConfig *config.E2EConfig) error {
	composeConfigPath := e2eConfig.Setup.GetFile()
	if composeConfigPath == "" {
		return fmt.Errorf("no compose config file was provided")
	}

	identifier := GetIdentity()
	compose, err := tc.NewDockerComposeWith(
		tc.StackIdentifier(identifier),
		tc.WithStackFiles(composeConfigPath),
	)
	if err != nil {
		return fmt.Errorf("compose setup error: %v", err)
	}

	if err = compose.Up(context.Background(), tc.Wait(true)); err != nil {
		return err
	}

	for _, service := range compose.Services() {
		container, err := compose.ServiceContainer(context.Background(), service)
		if err != nil {
			return err
		}
		ports, err := container.Ports(context.Background())
		if err != nil {
			return err
		}
		for port := range ports {
			logger.Log.Debugf("waiting for port %v in container: %v/%v", port, service, container.ID)
			if err = wait.ForListeningPort(port).WaitUntilReady(context.Background(), container); err != nil {
				return err
			}
		}
	}

	// run steps
	err = RunStepsAndWait(e2eConfig.Setup.Steps, e2eConfig.Setup.GetTimeout(), nil)
	if err != nil {
		logger.Log.Errorf("execute steps error: %v", err)
		return err
	}

	return nil
}
