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

package config

import (
	"os"
	"testing"

	"github.com/apache/skywalking-infra-e2e/internal/util"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestSetup_GetFile(t *testing.T) {
	randomVersion := rand.String(10)
	type fields struct {
		File string
	}
	tests := []struct {
		name       string
		fields     fields
		beforeEach func()
		want       string
	}{{
		name: "Should keep file path still",
		fields: fields{
			File: "fake/file/path.yaml",
		},
		want: util.ResolveAbs("fake/file/path.yaml"),
	}, {
		name: "Should expand file path",
		fields: fields{
			File: "kind/$VERSION.yaml",
		},
		beforeEach: func() {
			os.Clearenv()
			os.Setenv("VERSION", randomVersion)
		},
		want: util.ResolveAbs("kind/" + randomVersion + ".yaml"),
	}, {
		name: "Should never expand file path",
		fields: fields{
			File: "kind/$VERSION.yaml",
		},
		beforeEach: func() {
			os.Clearenv()
			os.Setenv("INVAD_VERSION", randomVersion)
		},
		want: util.ResolveAbs("kind/.yaml"),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.beforeEach != nil {
				tt.beforeEach()
			}
			s := &Setup{
				File: tt.fields.File,
			}
			if got := s.GetFile(); got != tt.want {
				t.Errorf("Setup.GetFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
