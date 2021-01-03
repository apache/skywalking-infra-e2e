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
// Kind, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//

package setup

import (
	"strings"

	kind "sigs.k8s.io/kind/cmd/kind/app"
	kindcmd "sigs.k8s.io/kind/pkg/cmd"

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
	Kind        = "kind"
	KindCommand = "kind"
)

var (
	// kind cluster create config
	kindConfigFile string
)

// setup for kind, invoke from command line
func KindSetupInCommand() error {
	kindConfigFile = flags.File

	if err := createKindCluster(); err != nil {
		return err
	}
	return nil
}

func createKindCluster() error {
	args := []string{"create", "cluster", "--config", kindConfigFile}
	// quiet mode to suppress status output, so that the error is not logged repeatedly
	args = append(args, "--quiet")

	logger.Log.Info("creating kind cluster...")
	logger.Log.Debugf("cluster create commands: %s %s", KindCommand, strings.Join(args, " "))
	if err := kind.Run(kindcmd.NewLogger(), kindcmd.StandardIOStreams(), args); err != nil {
		return err
	}
	logger.Log.Info("create kind cluster succeed")
	return nil
}
