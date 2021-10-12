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

package util

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestResolveAbs(t *testing.T) {
	configAbsPath, _ := filepath.Abs("config.go")
	tests := []struct {
		CfgPath string
		Path    string
		Exists  bool
	}{
		{
			configAbsPath,
			"./config.go",
			true,
		},
		{
			"not_exist/e2e.yaml",
			configAbsPath,
			true,
		},
		{
			"config.go",
			"./config.go",
			true,
		},
		{
			"./config.go",
			"config.go",
			true,
		},
		{
			"./config.go",
			"go.mod",
			false,
		},
		{
			"./config.go",
			"",
			false,
		},
		{
			"../../examples/compose/e2e.yaml",
			"env",
			true,
		},
		{
			"../../examples/compose/e2e.yaml",
			"./env",
			true,
		},
		{
			"not_exists/e2e.yaml",
			"./config.go",
			false,
		},
		{
			"not_exists/e2e.yaml",
			"config.go",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("cfg path: %s, verify path: %s", tt.CfgPath, tt.Path), func(t *testing.T) {
			CfgFile = tt.CfgPath
			result := ResolveAbs(tt.Path)
			if PathExist(result) != tt.Exists {
				t.Errorf("path %s not exists", result)
			}
		})
	}
}
