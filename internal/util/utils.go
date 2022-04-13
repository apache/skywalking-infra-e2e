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
	_ "embed"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/apache/skywalking-infra-e2e/internal/logger"
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
		content, err := os.ReadFile(filename)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
	return "", errors.New("the file does not exist")
}

// ExecuteCommand executes the given command and returns the result.
func ExecuteCommand(cmd string) (stdout, stderr string, err error) {
	hookScript, err := hookScript()
	if err != nil {
		return "", "", err
	}

	// Propagate the env vars from sub-process back to parent process
	defer ExportEnvVars(filepath.Join(WorkDir, ".env"))

	cmd = hookScript + "\n" + cmd

	command := exec.Command("bash", "-ec", cmd)
	sout, serr := bytes.Buffer{}, bytes.Buffer{}
	command.Stdout, command.Stderr = &sout, &serr

	if err := command.Start(); err != nil {
		return sout.String(), serr.String(), err
	}
	if err := command.Wait(); err != nil {
		return sout.String(), serr.String(), err
	}
	return sout.String(), serr.String(), nil
}

//go:embed hook.sh
var hookScriptTemplate string

type HookScriptTemplate struct {
	EnvFile string
}

func hookScript() (string, error) {
	hookScript := bytes.Buffer{}

	parse, err := template.New("hookScriptTemplate").Parse(hookScriptTemplate)
	if err != nil {
		return "", err
	}

	envFile := filepath.Join(WorkDir, ".env")
	scriptData := HookScriptTemplate{EnvFile: envFile}
	if err := parse.Execute(&hookScript, scriptData); err != nil {
		return "", err
	}
	return hookScript.String(), nil
}

func ExportEnvVars(envFile string) {
	b, err := os.ReadFile(envFile)
	if err != nil {
		logger.Log.Warnf("failed to export environment variables, %v", err)
		return
	}
	s := string(b)

	lines := strings.Split(s, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := kv[0], kv[1]
		// should only export env vars that are not already exist in parent process (Go process)
		if err := os.Setenv(key, val); err != nil {
			logger.Log.Warnf("failed to export environment variable %v=%v, %v", key, val, err)
		}
	}
}
