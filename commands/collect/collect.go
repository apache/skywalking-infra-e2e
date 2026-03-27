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

package collect

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/apache/skywalking-infra-e2e/internal/components/collector"
	"github.com/apache/skywalking-infra-e2e/internal/config"
)

var Collect = &cobra.Command{
	Use:   "collect",
	Short: "Collect files from pods/containers for debugging",
	RunE: func(cmd *cobra.Command, args []string) error {
		if config.GlobalConfig.Error != nil {
			return config.GlobalConfig.Error
		}
		err := collector.DoCollect(&config.GlobalConfig.E2EConfig)
		if err != nil {
			return fmt.Errorf("[Collect] %s", err)
		}
		return nil
	},
}
