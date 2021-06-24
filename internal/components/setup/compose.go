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
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/constant"
	"github.com/apache/skywalking-infra-e2e/internal/logger"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"github.com/testcontainers/testcontainers-go"
)

// ComposeSetup sets up environment according to e2e.yaml.
func ComposeSetup(e2eConfig *config.E2EConfig) error {
	composeConfigPath := e2eConfig.Setup.GetFile()
	if composeConfigPath == "" {
		return fmt.Errorf("no compose config file was provided")
	}

	// build docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	// setup docker compose
	composeFilePaths := []string{
		composeConfigPath,
	}
	identifier := GetIdentity()
	compose := testcontainers.NewLocalDockerCompose(composeFilePaths, identifier)

	// bind wait port
	timeout := e2eConfig.Setup.Timeout
	var waitTimeout time.Duration
	if timeout <= 0 {
		waitTimeout = constant.DefaultWaitTimeout
	} else {
		waitTimeout = time.Duration(timeout) * time.Second
	}
	serviceWithPorts := make(map[string][]int)
	for service, content := range compose.Services {
		serviceConfig := content.(map[interface{}]interface{})
		ports := serviceConfig["ports"]
		if ports == nil {
			continue
		}
		serviceWithPorts[service] = []int{}

		portList := ports.([]interface{})
		for inx := range portList {
			exportPort, err := getExpectPort(portList[inx])
			if err != nil {
				return err
			}
			serviceWithPorts[service] = append(serviceWithPorts[service], exportPort)

			compose.WithExposedService(
				service,
				exportPort,
				wait.NewHostPortStrategy(nat.Port(fmt.Sprintf("%d/tcp", exportPort))).WithStartupTimeout(waitTimeout))
		}
	}

	execError := compose.WithCommand([]string{"up", "-d"}).Invoke()
	if execError.Error != nil {
		return execError.Error
	}

	// find exported port and build env
	for service, portList := range serviceWithPorts {
		container, err := findContainer(cli, fmt.Sprintf("%s_%s", identifier, getInstanceName(service)))
		if err != nil {
			return err
		}
		containerPorts := container.Ports

		for inx := range portList {
			for _, containerPort := range containerPorts {
				if int(containerPort.PrivatePort) != portList[inx] {
					continue
				}

				// expose env config to env
				// format: <service_name>_<port>
				envKey := fmt.Sprintf("%s_%d", service, containerPort.PrivatePort)
				envValue := fmt.Sprintf("%d", containerPort.PublicPort)
				err = os.Setenv(envKey, envValue)
				if err != nil {
					return fmt.Errorf("could not set env for %s:%d, %v", service, portList[inx], err)
				}
				logger.Log.Infof("expose env : %s : %s", envKey, envValue)
				break
			}
		}
	}

	return nil
}

func getExpectPort(portConfig interface{}) (int, error) {
	switch conf := portConfig.(type) {
	case int:
		return conf, nil
	case string:
		portInfo := strings.Split(conf, ":")
		if len(portInfo) > 1 {
			return strconv.Atoi(portInfo[1])
		}
		return strconv.Atoi(portInfo[0])
	}
	return 0, fmt.Errorf("unknown port information: %v", portConfig)
}

func findContainer(c *client.Client, instanceName string) (*types.Container, error) {
	f := filters.NewArgs(filters.Arg("name", instanceName))
	containerListOptions := types.ContainerListOptions{Filters: f}
	containers, err := c.ContainerList(context.Background(), containerListOptions)
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("could not found container: %s", instanceName)
	}
	return &containers[0], nil
}

func getInstanceName(serviceName string) string {
	match, err := regexp.MatchString(".*_[0-9]+", serviceName)
	if err != nil {
		return serviceName
	}
	if !match {
		return serviceName + "_1"
	}
	return serviceName
}
