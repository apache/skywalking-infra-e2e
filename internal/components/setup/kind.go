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

package setup

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/apache/skywalking-infra-e2e/internal/flags"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

type ExecResult struct {
	Command []string
	Error   error
	Stdout  string
	StdErr  string
}

const (
	KIND        = "kind"
	KINDCOMMAND = "kind"
)

var (
	// kind cluster create config
	kindConfigFile string
)

// setup for kind, invoke from command line
func KindSetupInCommand() {
	kindConfigFile = flags.File

	execResult := createKindCluster()
	err := execResult.Error
	if err != nil {
		cmd := strings.Join(execResult.Command, " ")
		logger.Log.Errorf("Kind cluster create exited abnormally whilst running [%s]\n"+
			"err: %s\nstdout: %s\nstderr: %s", cmd, err, execResult.Stdout, execResult.StdErr)
	} else {
		defer cleanupKindCluster()
	}
}

func kindExec(args []string) ExecResult {
	cmd := exec.Command(KINDCOMMAND, args...)
	var stdoutBytes, stderrBytes bytes.Buffer
	cmd.Stdout = &stdoutBytes
	cmd.Stderr = &stderrBytes

	err := cmd.Run()
	execCmd := []string{KINDCOMMAND}
	execCmd = append(execCmd, args...)

	return ExecResult{
		Command: execCmd,
		Error:   err,
		Stdout:  stdoutBytes.String(),
		StdErr:  stderrBytes.String(),
	}
}

func createKindCluster() ExecResult {
	args := []string{"create", "cluster", "--config", kindConfigFile}

	logger.Log.Info("creating kind cluster...")
	return kindExec(args)
}

func cleanupKindCluster() ExecResult {
	args := []string{"delete", "cluster"}

	logger.Log.Info("deleting kind cluster...")
	return kindExec(args)
}
