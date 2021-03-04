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
package verifier

import "testing"

func TestVerify(t *testing.T) {
	type args struct {
		actualData       string
		expectedTemplate string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "should contain two elements",
			args: args{
				actualData: `
metrics:
  - name: business-zone::projectA
    id: YnVzaW5lc3Mtem9uZTo6cHJvamVjdEE=.1
    value: 1
  - name: system::load balancer1
    id: c3lzdGVtOjpsb2FkIGJhbGFuY2VyMQ==.1
    value: 0
  - name: system::load balancer2
    id: WW91cl9BcHBsaWNhdGlvbk5hbWU=.1
    value: 2
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
    value: {{ gt .value 0 }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
    value: {{ gt .value 1 }}
{{- end }}
`,
			},
			wantErr: false,
		},
		{
			name: "fail to contain two elements",
			args: args{
				actualData: `
metrics:
  - name: business-zone::projectA
    id: YnVzaW5lc3Mtem9uZTo6cHJvamVjdEE=.1
    value: 1
  - name: system::load balancer1
    id: c3lzdGVtOjpsb2FkIGJhbGFuY2VyMQ==.1
    value: 0
  - name: system::load balancer2
    id: WW91cl9BcHBsaWNhdGlvbk5hbWU=.1
    value: 1
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
    value: {{ gt .value 0 }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
    value: {{ gt .value 1 }}
{{- end }}
`,
			},
			wantErr: true,
		},
		{
			name: "should contain one element",
			args: args{
				actualData: `
metrics:
  - name: business-zone::projectA
    id: YnVzaW5lc3Mtem9uZTo6cHJvamVjdEE=.1
    value: 1
  - name: system::load balancer1
    id: c3lzdGVtOjpsb2FkIGJhbGFuY2VyMQ==.1
    value: 0
  - name: system::load balancer2
    id: WW91cl9BcHBsaWNhdGlvbk5hbWU=.1
    value: 2
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
    value: {{ gt .value 1 }}
{{- end }}
`,
			},
			wantErr: false,
		},
		{
			name: "fail to contain one element",
			args: args{
				actualData: `
metrics:
  - name: business-zone::projectA
    id: YnVzaW5lc3Mtem9uZTo6cHJvamVjdEE=.1
    value: 1
  - name: system::load balancer1
    id: c3lzdGVtOjpsb2FkIGJhbGFuY2VyMQ==.1
    value: 0
  - name: system::load balancer2
    id: WW91cl9BcHBsaWNhdGlvbk5hbWU=.1
    value: 2
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
    value: {{ gt .value 3 }}
{{- end }}
`,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Verify(tt.args.actualData, tt.args.expectedTemplate); (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
