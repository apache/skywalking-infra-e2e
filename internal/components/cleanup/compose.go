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

package cleanup

import (
	"github.com/apache/skywalking-infra-e2e/internal/components/setup"
	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/logger"

	"github.com/testcontainers/testcontainers-go"

	"fmt"
)

func ComposeCleanUp(conf *config.E2EConfig) error {
	composeFilePath := conf.Setup.GetFile()
	logger.Log.Infof("deleting docker compose cluster...\n")

	if composeFilePath == "" {
		return fmt.Errorf("no compose config file was provided")
	}
	composeFilePaths := []string{composeFilePath}
	identifier := setup.GetIdentity()
	compose := testcontainers.NewLocalDockerCompose(composeFilePaths, identifier)
	down := compose.Down()
	if down.Error != nil {
		return down.Error
	}

	return nil
}
