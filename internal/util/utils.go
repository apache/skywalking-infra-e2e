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
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
)

// PathExist checks if a file/directory is exist.
func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// ReadFileContent reads the file content.
func ReadFileContent(filename string) (string, error) {
	if PathExist(filename) {
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
	return "", errors.New("the file does not exist")
}

// ExecuteCommand executes the given command and returns the result.
func ExecuteCommand(cmd string) (string, error) {
	command := exec.Command("bash", "-ec", cmd)
	outinfo := bytes.Buffer{}
	command.Stdout = &outinfo

	if err := command.Start(); err != nil {
		return outinfo.String(), err
	}
	if err := command.Wait(); err != nil {
		return outinfo.String(), err
	}
	return outinfo.String(), nil
}
