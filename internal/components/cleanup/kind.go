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

package cleanup

import (
	"io/ioutil"
	"os"

	"github.com/apache/skywalking-infra-e2e/internal/config"

	"gopkg.in/yaml.v2"
	kind "sigs.k8s.io/kind/cmd/kind/app"
	kindcmd "sigs.k8s.io/kind/pkg/cmd"

	"strings"

	"github.com/apache/skywalking-infra-e2e/internal/constant"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

type KindClusterNameConfig struct {
	Name string
}

func KindCleanUp(e2eConfig *config.E2EConfig) error {
	kindConfigFilePath := e2eConfig.Setup.File

	k8sConfigFile := constant.K8sClusterConfigFile

	logger.Log.Infof("deleting k8s cluster config file:%s", k8sConfigFile)
	err := os.Remove(k8sConfigFile)
	if err != nil {
		logger.Log.Errorf("delete k8s cluster config file failed")
		return err
	}

	logger.Log.Info("deleting kind cluster...")
	if err := cleanKindCluster(kindConfigFilePath); err != nil {
		logger.Log.Error("delete kind cluster failed")
		return err
	}
	logger.Log.Info("delete kind cluster succeeded")

	return nil
}

func getKindClusterName(kindConfigFilePath string) (name string, err error) {
	data, err := ioutil.ReadFile(kindConfigFilePath)
	if err != nil {
		return "", err
	}

	nameConfig := KindClusterNameConfig{}
	err = yaml.Unmarshal(data, &nameConfig)
	if err != nil {
		return "", err
	}

	if nameConfig.Name == "" {
		nameConfig.Name = constant.KindClusterDefaultName
	}

	return nameConfig.Name, nil
}

func cleanKindCluster(kindConfigFilePath string) error {
	clusterName, err := getKindClusterName(kindConfigFilePath)
	if err != nil {
		return err
	}

	args := []string{"delete", "cluster", "--name", clusterName}

	logger.Log.Debugf("cluster delete commands: %s %s", constant.KindCommand, strings.Join(args, " "))
	if err := kind.Run(kindcmd.NewLogger(), kindcmd.StandardIOStreams(), args); err != nil {
		return err
	}

	return nil
}
