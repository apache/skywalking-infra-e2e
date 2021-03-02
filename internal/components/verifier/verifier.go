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

import (
	"bytes"
	"fmt"

	"github.com/apache/skywalking-infra-e2e/internal/util"
	"github.com/apache/skywalking-infra-e2e/third-party/go/template"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

// MismatchError is the error type returned by the verify functions.
// It contains the diff content.
type MismatchError struct {
	Err  error
	diff string
}

func (e *MismatchError) Unwrap() error { return e.Err }

func (e *MismatchError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.diff
}

// VerifyDataFile reads the actual data from the file and verifies.
func VerifyDataFile(actualFile, expectedData string) error {
	actualData, err := util.ReadFileContent(actualFile)
	if err != nil {
		return fmt.Errorf("failed to read the actual data file: %v", err)
	}

	return verify(actualData, expectedData)
}

// VerifyQuery gets the actual data from the query and then verifies.
func VerifyQuery(query, expectedData string) error {
	queryResult, err := util.ExecuteCommand(query)
	if err != nil {
		return fmt.Errorf("failed to execute the query: %v", err)
	}

	// TODO: ensure that the query result has the same format as expected data

	return verify(queryResult, expectedData)
}

// verify checks if the actual data match the expected template.
func verify(actualData, expectedTemplate string) error {
	var actual interface{}
	if err := yaml.Unmarshal([]byte(actualData), &actual); err != nil {
		return fmt.Errorf("failed to unmarshal actual data: %v", err)
	}

	tmpl, err := template.New("test").Funcs(funcMap()).Parse(expectedTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	var b bytes.Buffer
	if err := tmpl.Execute(&b, actual); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	var expected interface{}
	if err := yaml.Unmarshal(b.Bytes(), &expected); err != nil {
		return fmt.Errorf("failed to unmarshal expected data: %v", err)
	}

	if !cmp.Equal(expected, actual) {
		// TODO: use a custom Reporter (suggested by the comment of cmp.Diff)
		diff := cmp.Diff(expected, actual)
		return &MismatchError{diff: diff}
	}
	return nil
}
