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

package os

import "testing"

func TestExpandFromSpecificEnv(t *testing.T) {
	env := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	tests := []struct {
		name     string
		content  string
		excepted string
	}{
		{
			name:     "normal ${var}",
			content:  "test ${key1} content",
			excepted: "test value1 content",
		},
		{
			name:     "normal $var",
			content:  "test $key1 content",
			excepted: "test value1 content",
		},
		{
			name:     "multiple var",
			content:  "test $key1 $key2 content",
			excepted: "test value1 value2 content",
		},
		{
			name:     "not exists ${var}",
			content:  "test ${not_exists} content",
			excepted: "test ${not_exists} content",
		},
		{
			name:     "not exists $var",
			content:  "test $not_exists content",
			excepted: "test $not_exists content",
		},
		{
			name:     "wrong content",
			content:  "test ${not_exists content",
			excepted: "test ${not_exists content",
		},
		{
			name:     "empty ${}",
			content:  "test ${} content",
			excepted: "test ${} content",
		},
		{
			name:     "empty $",
			content:  "test $ content",
			excepted: "test $ content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded := ExpandFromSpecificEnv(tt.content, env)
			if tt.excepted != expanded {
				t.Fatalf("except: %s, got: %s", tt.excepted, expanded)
			}
		})
	}
}
