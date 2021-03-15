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
package trigger

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/apache/skywalking-infra-e2e/internal/components/trigger"
	"github.com/apache/skywalking-infra-e2e/internal/config"

	"github.com/apache/skywalking-infra-e2e/internal/constant"
)

var Trigger = &cobra.Command{
	Use:   "trigger",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := DoActionAccordingE2E(); err != nil {
			return fmt.Errorf("[Trigger] %s", err)
		}

		return nil
	},
}

func DoActionAccordingE2E() error {
	if config.GlobalConfig.Error != nil {
		return config.GlobalConfig.Error
	}

	e2eConfig := config.GlobalConfig.E2EConfig
	if e2eConfig.Trigger.Action == constant.ActionHTTP {
		action := trigger.NewHTTPAction(e2eConfig.Trigger.Interval,
			e2eConfig.Trigger.Times,
			e2eConfig.Trigger.URL,
			e2eConfig.Trigger.Method)
		if action == nil {
			return fmt.Errorf("trigger [%+v] parse error", e2eConfig.Trigger)
		}

		err := action.Do()
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no such action for trigger: %s", e2eConfig.Trigger.Action)
	}

	return nil
}
