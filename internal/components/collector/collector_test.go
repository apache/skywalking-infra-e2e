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
	"testing"

	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/constant"
)

func TestDoCollect_NoItems(t *testing.T) {
	cfg := &config.E2EConfig{
		Setup: config.Setup{Env: constant.Kind},
		Cleanup: config.Cleanup{
			Collect: config.CollectConfig{
				On:    constant.CollectOnFailure,
				Items: []config.CollectItem{},
			},
		},
	}
	// Should return nil with no items
	if err := DoCollect(cfg); err != nil {
		t.Errorf("DoCollect with no items should return nil, got: %v", err)
	}
}

func TestDoCollect_UnsupportedEnv(t *testing.T) {
	cfg := &config.E2EConfig{
		Setup: config.Setup{Env: "unsupported"},
		Cleanup: config.Cleanup{
			Collect: config.CollectConfig{
				On:        constant.CollectOnFailure,
				OutputDir: t.TempDir(),
				Items: []config.CollectItem{
					{Service: "test", Paths: []string{"/tmp"}},
				},
			},
		},
	}
	err := DoCollect(cfg)
	if err == nil {
		t.Error("DoCollect with unsupported env should return error")
	}
}

func TestListPods_ResourceFormat(t *testing.T) {
	tests := []struct {
		name      string
		item      config.CollectItem
		wantPods  int
		wantName  string
		wantError bool
	}{
		{
			name:      "Valid pod resource",
			item:      config.CollectItem{Resource: "pod/oap-0", Namespace: "default"},
			wantPods:  1,
			wantName:  "oap-0",
			wantError: false,
		},
		{
			name:      "Invalid resource format",
			item:      config.CollectItem{Resource: "deployment/oap", Namespace: "default"},
			wantPods:  0,
			wantError: true,
		},
		{
			name:      "No resource or label selector",
			item:      config.CollectItem{Namespace: "default"},
			wantPods:  0,
			wantError: true,
		},
		{
			name:      "Default namespace when empty",
			item:      config.CollectItem{Resource: "pod/oap-0"},
			wantPods:  1,
			wantName:  "oap-0",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Only test static resource parsing (label-selector requires a real cluster)
			if tt.item.LabelSelector != "" {
				t.Skip("skipping label-selector test without cluster")
			}
			pods, err := listPods("", &tt.item)
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(pods) != tt.wantPods {
				t.Errorf("got %d pods, want %d", len(pods), tt.wantPods)
			}
			if tt.wantName != "" && len(pods) > 0 && pods[0].name != tt.wantName {
				t.Errorf("pod name = %v, want %v", pods[0].name, tt.wantName)
			}
		})
	}
}

func TestContainsGlob(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/skywalking/logs/", false},
		{"/tmp/dump.hprof", false},
		{"/tmp/app[1].log", true},  // [1] is a valid shell character class
		{"/tmp/app[].log", false},  // [] is not a valid character class
		{"/skywalking/logs*", true},
		{"/tmp/*.hprof", true},
		{"/tmp/dump-[0-9].hprof", true},
		{"/var/log/?oo", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := containsGlob(tt.path); got != tt.want {
				t.Errorf("containsGlob(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestValidateGlobPattern(t *testing.T) {
	tests := []struct {
		pattern string
		wantErr bool
	}{
		{"/skywalking/logs*", false},
		{"/tmp/*.hprof", false},
		{"/tmp/dump-[0-9].hprof", false},
		{"/var/log/app-?.log", false},
		{"'; rm -rf /; '", true},
		{"/path with spaces/*", true},
		{"/tmp/$(whoami)", true},
		{"/tmp/`id`", true},
		{"/tmp/foo|bar", true},
		{"/tmp/foo;bar", true},
		{"/tmp/foo&bar", true},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			err := validateGlobPattern(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGlobPattern(%q) error = %v, wantErr %v", tt.pattern, err, tt.wantErr)
			}
		})
	}
}

func TestExpandPodGlob_NoGlob(t *testing.T) {
	paths, err := expandPodGlob("", "default", "pod-0", "", "/skywalking/logs/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 1 || paths[0] != "/skywalking/logs/" {
		t.Errorf("expected [/skywalking/logs/], got %v", paths)
	}
}

func TestExpandContainerGlob_NoGlob(t *testing.T) {
	paths, err := expandContainerGlob("abc123", "svc", "/var/log/app.log")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 1 || paths[0] != "/var/log/app.log" {
		t.Errorf("expected [/var/log/app.log], got %v", paths)
	}
}

func TestComposeCollectItem_NoService(t *testing.T) {
	err := composeCollectItem("/fake/compose.yml", "test-project", t.TempDir(), &config.CollectItem{
		Paths: []string{"/tmp"},
	})
	if err == nil {
		t.Error("composeCollectItem with empty service should return error")
	}
}
