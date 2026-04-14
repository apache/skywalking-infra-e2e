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
		{
			name: "multiple level attribute and contains greater and equals 2",
			args: args{
				actualData: `
metrics:
  key:
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
  key:
  {{- contains .metrics.key }}
    - name: {{ notEmpty .name }}
      id: {{ notEmpty .id }}
      value: {{ ge .value 2 }}
  {{- end }}
`,
			},
			wantErr: false,
		},
		{
			name: "multiple level attribute and contains greater 2",
			args: args{
				actualData: `
metrics:
  key:
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
  key:
  {{- contains .metrics.key }}
    - name: {{ notEmpty .name }}
      id: {{ notEmpty .id }}
      value: {{ gt .value 2 }}
  {{- end }}
`,
			},
			wantErr: true,
		},
		{
			name: "multiple level attribute and contains greater 2",
			args: args{
				actualData: `
metrics:
  key:
  - name: business-zone::projectA
    id: YnVzaW5lc3Mtem9uZTo6cHJvamVjdEE=.1
    value: 1
`,
				expectedTemplate: `
metrics:
  key:
  {{- contains .metrics.key }}
    - name: {{ notEmpty .name }}
      id: {{ notEmpty .id }}
      value: {{ gt .value 0 }}
    - name: {{ notEmpty .name }}
      id: {{ notEmpty .id }}
      value: {{ gt .value 2 }}
  {{- end }}
`,
			},
			wantErr: true,
		},
		{
			name: "contains unordered slices",
			args: args{
				actualData: `
- id: ZTJlLXNlcnZpY2UtcHJvdmlkZXI=.1_cHJvdmlkZXIx
  name: whatever
  attributes:
  - name: JVM Arguments
    value: abcde
  - name: OS Name
    value: Linux
  - name: hostname
    value: 127.0.0.1
  - name: Process No.
    value: "1"
  - name: Start Time
    value: "12345"
  - name: Jar Dependencies
    value: abcde
  - name: ipv4s
    value: abcde
  language: JAVA
  instanceuuid: ZTJlLXNlcnZpY2UtcHJvdmlkZXI=.1_cHJvdmlkZXIx
`,
				expectedTemplate: `
{{- contains . }}
- id: {{ b64enc "e2e-service-provider" }}.1_{{ b64enc "provider1" }}
  name: {{ notEmpty .name }}
  attributes:
  {{- contains .attributes }}
  - name: Jar Dependencies
    value: '{{ notEmpty .value }}'
  - name: OS Name
    value: Linux
  - name: hostname
    value: {{ notEmpty .value }}
  - name: ipv4s
    value: {{ notEmpty .value }}
  - name: Process No.
    value: "1"
  - name: Start Time
    value: {{ notEmpty .value }}
  - name: JVM Arguments
    value: '{{ notEmpty .value }}'
  {{- end}}
  language: JAVA
  instanceuuid: {{ b64enc "e2e-service-provider" }}.1_{{ b64enc "provider1" }}
{{- end}}
`,
			},
			wantErr: false,
		},		{
			name: "contains should require distinct actual items for each expected entry",
			args: args{
				actualData: `
metrics:
  - name: service-A
    id: abc
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
{{- end }}
`,
			},
			wantErr: true, // expected 2 entries but actual only has 1
		},
		{
			name: "contains should not pass when extra expected entries are added with echo-only conditions",
			args: args{
				actualData: `
metrics:
  - name: service-A
    id: abc
  - name: service-B
    id: def
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
{{- end }}
`,
			},
			wantErr: true, // expected 3 entries but actual only has 2
		},
		{
			name: "contains greedy cannot solve reordered assignment",
			args: args{
				actualData: `
- name: service-A
  language: JAVA
- name: service-B
  language: GO
`,
				expectedTemplate: `
{{- contains . }}
- name: {{ notEmpty .name }}
  language: {{ notEmpty .language }}
- name: {{ notEmpty .name }}
  language: JAVA
{{- end }}
`,
			},
			wantErr: true, // greedy contains can't solve this; containsOnce with backtracking will
		},
		{
			name: "containsOnce should backtrack to find valid assignment",
			args: args{
				actualData: `
- name: service-A
  language: JAVA
- name: service-B
  language: GO
`,
				expectedTemplate: `
{{- containsOnce . }}
- name: {{ notEmpty .name }}
  language: {{ notEmpty .language }}
- name: {{ notEmpty .name }}
  language: JAVA
{{- end }}
`,
			},
			wantErr: false, // backtracking finds: expected[0]→actual[1], expected[1]→actual[0]
		},
		{
			name: "containsOnce should require distinct actual items",
			args: args{
				actualData: `
metrics:
  - name: service-A
    id: abc
`,
				expectedTemplate: `
metrics:
{{- containsOnce .metrics }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
{{- end }}
`,
			},
			wantErr: true, // 1 actual item cannot satisfy 2 expected entries
		},
		{
			name: "containsOnce should pass when enough distinct items match",
			args: args{
				actualData: `
metrics:
  - name: service-A
    id: abc
  - name: service-B
    id: def
  - name: service-C
    id: ghi
`,
				expectedTemplate: `
metrics:
{{- containsOnce .metrics }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
{{- end }}
`,
			},
			wantErr: false, // 3 actual items, 2 expected → enough distinct matches
		},
		{
			name: "containsOnce should fail when specific value missing",
			args: args{
				actualData: `
- name: service-B
  value: "200"
- name: service-C
  value: "300"
`,
				expectedTemplate: `
{{- containsOnce . }}
- name: service-A
  value: "100"
{{- end }}
`,
			},
			wantErr: true, // service-A does not exist
		},
		{
			name: "contains should match specific values in reversed order",
			args: args{
				actualData: `
metrics:
  - name: service-B
    value: "200"
  - name: service-A
    value: "100"
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: service-A
    value: "100"
  - name: service-B
    value: "200"
{{- end }}
`,
			},
			wantErr: false, // both exist, order shouldn't matter
		},
		{
			name: "contains should fail when one specific value is wrong",
			args: args{
				actualData: `
metrics:
  - name: service-A
    value: "100"
  - name: service-B
    value: "200"
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: service-A
    value: "100"
  - name: service-C
    value: "300"
{{- end }}
`,
			},
			wantErr: true, // service-C does not exist in actual
		},
		{
			name: "contains should match unordered actual data correctly",
			args: args{
				actualData: `
metrics:
  - name: service-B
    value: 200
  - name: service-A
    value: 100
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: service-A
    value: 100
{{- end }}
`,
			},
			wantErr: false, // service-A exists in actual, order shouldn't matter
		},
		{
			name: "contains should fail when specific expected value is missing from actual",
			args: args{
				actualData: `
metrics:
  - name: service-B
    value: 200
  - name: service-C
    value: 300
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: service-A
    value: 100
{{- end }}
`,
			},
			wantErr: true, // service-A does not exist in actual
		},
		{
			name: "noDuplicates should pass when list has no duplicates",
			args: args{
				actualData: `
metrics:
  - name: service-A
    value: "100"
  - name: service-B
    value: "200"
`,
				expectedTemplate: `
metrics:
{{- contains (.metrics | noDuplicates) }}
  - name: {{ notEmpty .name }}
    value: {{ notEmpty .value }}
{{- end }}
`,
			},
			wantErr: false,
		},
		{
			name: "noDuplicates should fail when list has duplicate items",
			args: args{
				actualData: `
metrics:
  - name: service-A
    value: "100"
  - name: service-A
    value: "100"
`,
				expectedTemplate: `
metrics:
{{- contains (.metrics | noDuplicates) }}
  - name: {{ notEmpty .name }}
    value: {{ notEmpty .value }}
{{- end }}
`,
			},
			wantErr: true, // duplicate entry exists
		},
		{
			name: "noDuplicates should work with containsOnce",
			args: args{
				actualData: `
- name: service-A
  value: "100"
- name: service-B
  value: "200"
- name: service-A
  value: "100"
`,
				expectedTemplate: `
{{- containsOnce (. | noDuplicates) }}
- name: {{ notEmpty .name }}
  value: {{ notEmpty .value }}
{{- end }}
`,
			},
			wantErr: true, // duplicate entry exists even though containsOnce would pass
		},
		{
			name: "noDuplicates should pass with range and no duplicates",
			args: args{
				actualData: `
metrics:
  - name: service-A
  - name: service-B
`,
				expectedTemplate: `
metrics:
{{- range (.metrics | noDuplicates) }}
  - name: {{ notEmpty .name }}
{{- end }}
`,
			},
			wantErr: false,
		},
		{
			name: "notEmpty with nil",
			args: args{
				actualData: `
- key: 0
  value:
  - key: name
    value: SET TIMESTAMP
  - key: id
    value: "123"
  - key: refid
    value: null
      `,
				expectedTemplate: `
{{- contains . }}
- key: 0
  value:
  {{- contains .value }}
  - key: {{ notEmpty .key }}
    value: {{ notEmpty .value }}
  {{- end }}
{{- end }}
        `,
			},
			wantErr: false,
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
