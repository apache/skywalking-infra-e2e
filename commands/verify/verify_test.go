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
package verify

import (
	"testing"
	"time"
)

func Test_parseInterval(t *testing.T) {
	type args struct {
		retryInterval any
	}
	tests := []struct {
		name    string
		args    args
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "Backward compatibility, should parse numeric value",
			args:    args{retryInterval: 1000},
			want:    time.Second,
			wantErr: false,
		},
		{
			name:    "Should parse duration like 10s",
			args:    args{retryInterval: "10s"},
			want:    10 * time.Second,
			wantErr: false,
		},
		{
			name:    "Should have default value if < 0",
			args:    args{retryInterval: "-10s"},
			want:    1 * time.Second,
			wantErr: false,
		},
		{
			name:    "Should fail in other cases",
			args:    args{retryInterval: "abcdef"},
			wantErr: true,
		}, {
			name:    "Should parse interval without setting value",
			args:    args{},
			want:    0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInterval(tt.args.retryInterval)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInterval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseInterval() got = %v, want %v", got, tt.want)
			}
		})
	}
}
