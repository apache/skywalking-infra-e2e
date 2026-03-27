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

	"github.com/apache/skywalking-infra-e2e/internal/constant"
	"github.com/apache/skywalking-infra-e2e/internal/util"
	"gopkg.in/yaml.v2"
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

func TestCollectConfig_Parse(t *testing.T) {
	tests := []struct {
		name           string
		yamlInput      string
		wantOn         string
		wantOutputDir  string
		wantItemsCount int
	}{
		{
			name: "Full collect config with Kind items",
			yamlInput: `
cleanup:
  on: always
  collect:
    on: failure
    output-dir: /tmp/collect
    items:
      - namespace: default
        label-selector: app=oap
        container: oap
        paths:
          - /skywalking/logs/
          - /tmp/dump.hprof
`,
			wantOn:         "failure",
			wantOutputDir:  "/tmp/collect",
			wantItemsCount: 1,
		},
		{
			name: "Full collect config with Compose items",
			yamlInput: `
cleanup:
  on: always
  collect:
    on: always
    output-dir: /tmp/compose-collect
    items:
      - service: oap-service
        paths:
          - /skywalking/logs/
      - service: ui-service
        paths:
          - /var/log/nginx/
`,
			wantOn:         "always",
			wantOutputDir:  "/tmp/compose-collect",
			wantItemsCount: 2,
		},
		{
			name: "Empty collect config",
			yamlInput: `
cleanup:
  on: always
`,
			wantOn:         "",
			wantOutputDir:  "",
			wantItemsCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg E2EConfig
			if err := yaml.Unmarshal([]byte(tt.yamlInput), &cfg); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if cfg.Cleanup.Collect.On != tt.wantOn {
				t.Errorf("Collect.On = %v, want %v", cfg.Cleanup.Collect.On, tt.wantOn)
			}
			if cfg.Cleanup.Collect.OutputDir != tt.wantOutputDir {
				t.Errorf("Collect.OutputDir = %v, want %v", cfg.Cleanup.Collect.OutputDir, tt.wantOutputDir)
			}
			if len(cfg.Cleanup.Collect.Items) != tt.wantItemsCount {
				t.Errorf("len(Collect.Items) = %v, want %v", len(cfg.Cleanup.Collect.Items), tt.wantItemsCount)
			}
		})
	}
}

func TestCollectConfig_Finalize(t *testing.T) {
	tests := []struct {
		name          string
		collect       CollectConfig
		wantOn        string
		wantOutputDir string
	}{
		{
			name:          "Defaults when empty",
			collect:       CollectConfig{},
			wantOn:        constant.CollectOnFailure,
			wantOutputDir: "",
		},
		{
			name: "Keeps explicit values",
			collect: CollectConfig{
				On:        constant.CollectAlways,
				OutputDir: "/custom/output",
			},
			wantOn:        constant.CollectAlways,
			wantOutputDir: "/custom/output",
		},
		{
			name: "Expands env var in output-dir",
			collect: CollectConfig{
				OutputDir: "$TEST_COLLECT_DIR/collect",
			},
			wantOn:        constant.CollectOnFailure,
			wantOutputDir: "/tmp/test-collect/collect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Expands env var in output-dir" {
				os.Setenv("TEST_COLLECT_DIR", "/tmp/test-collect")
				defer os.Unsetenv("TEST_COLLECT_DIR")
			}
			c := tt.collect
			c.Finalize()
			if c.On != tt.wantOn {
				t.Errorf("Collect.On = %v, want %v", c.On, tt.wantOn)
			}
			if c.OutputDir != tt.wantOutputDir {
				t.Errorf("Collect.OutputDir = %v, want %v", c.OutputDir, tt.wantOutputDir)
			}
		})
	}
}

func TestCollectItem_KindFields(t *testing.T) {
	yamlInput := `
namespace: skywalking
label-selector: app=oap
container: oap-container
resource: pod/oap-0
paths:
  - /skywalking/logs/
  - /tmp/heap-dump.hprof
`
	var item CollectItem
	if err := yaml.Unmarshal([]byte(yamlInput), &item); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if item.Namespace != "skywalking" {
		t.Errorf("Namespace = %v, want skywalking", item.Namespace)
	}
	if item.LabelSelector != "app=oap" {
		t.Errorf("LabelSelector = %v, want app=oap", item.LabelSelector)
	}
	if item.Container != "oap-container" {
		t.Errorf("Container = %v, want oap-container", item.Container)
	}
	if item.Resource != "pod/oap-0" {
		t.Errorf("Resource = %v, want pod/oap-0", item.Resource)
	}
	if len(item.Paths) != 2 {
		t.Errorf("len(Paths) = %v, want 2", len(item.Paths))
	}
}

func TestCollectItem_ComposeFields(t *testing.T) {
	yamlInput := `
service: oap-service
paths:
  - /skywalking/logs/
`
	var item CollectItem
	if err := yaml.Unmarshal([]byte(yamlInput), &item); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if item.Service != "oap-service" {
		t.Errorf("Service = %v, want oap-service", item.Service)
	}
	if len(item.Paths) != 1 {
		t.Errorf("len(Paths) = %v, want 1", len(item.Paths))
	}
}
