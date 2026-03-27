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

package collector

import (
	"fmt"
	"os"

	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/constant"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

// DoCollect collects files from pods/containers based on the collect config.
// It dispatches to Kind or Compose collector based on setup.env.
// Errors are logged but tolerated — partial setup may leave some targets unreachable.
func DoCollect(e2eConfig *config.E2EConfig) error {
	collectCfg := &e2eConfig.Cleanup.Collect
	if len(collectCfg.Items) == 0 {
		logger.Log.Info("no collect items configured, skipping collection")
		return nil
	}

	if collectCfg.OutputDir == "" {
		return fmt.Errorf("collect output-dir is required when collect items are configured")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(collectCfg.OutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create collect output directory %s: %v", collectCfg.OutputDir, err)
	}

	logger.Log.Infof("collecting files to %s", collectCfg.OutputDir)

	switch e2eConfig.Setup.Env {
	case constant.Kind:
		return kindCollect(e2eConfig, collectCfg)
	case constant.Compose:
		return composeCollect(e2eConfig, collectCfg)
	default:
		return fmt.Errorf("unsupported env for collect: %s", e2eConfig.Setup.Env)
	}
}
