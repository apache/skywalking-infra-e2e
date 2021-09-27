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
	"github.com/apache/skywalking-infra-e2e/internal/util"

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
	serviceWithPorts, err := bindWaitPort(e2eConfig, compose)
	if err != nil {
		return fmt.Errorf("bind wait ports error: %v", err)
	}

	// build command
	cmd := make([]string, 0)
	if e2eConfig.Setup.InitSystemEnvironment != "" {
		profilePath := util.ResolveAbs(e2eConfig.Setup.InitSystemEnvironment)
		cmd = append(cmd, "--env-file", profilePath)
		util.ExportEnvVars(profilePath)
	}
	cmd = append(cmd, "up", "-d")

	// setup
	execError := compose.WithCommand(cmd).Invoke()
	if execError.Error != nil {
		return execError.Error
	}

	// find exported port and build env
	err = exposeServiceEnv(serviceWithPorts, cli, identifier, e2eConfig)
	if err != nil {
		return err
	}

	// run steps
	err = RunStepsAndWait(e2eConfig.Setup.Steps, e2eConfig.Setup.Timeout, nil)
	if err != nil {
		logger.Log.Errorf("execute steps error: %v", err)
		return err
	}

	return nil
}

func exposeServiceEnv(serviceWithPorts map[string][]*hostPortCachedStrategy, cli *client.Client, identity string, e2eConfig *config.E2EConfig) error {
	dockerProvider := &DockerProvider{client: cli}
	// find exported port and build env
	for service, portList := range serviceWithPorts {
		container, err := findContainer(cli, fmt.Sprintf("%s_%s", identity, getInstanceName(service)))
		if err != nil {
			return err
		}
		if len(portList) == 0 {
			continue
		}

		containerPorts := container.Ports

		// get real ip address for access and export to env
		host, err := dockerProvider.daemonHost(context.Background())
		if err != nil {
			return err
		}

		// format: <service_name>_host
		if err := exportComposeEnv(fmt.Sprintf("%s_host", service), host, service); err != nil {
			return err
		}

		for inx := range portList {
			for _, containerPort := range containerPorts {
				if int(containerPort.PrivatePort) != portList[inx].expectPort {
					continue
				}

				if err := waitPortUntilReady(e2eConfig, container, dockerProvider, portList[inx].expectPort); err != nil {
					return err
				}

				// expose env config to env
				// format: <service_name>_<port>
				if err := exportComposeEnv(
					fmt.Sprintf("%s_%d", service, containerPort.PrivatePort),
					fmt.Sprintf("%d", containerPort.PublicPort),
					service); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func exportComposeEnv(key, value, service string) error {
	err := os.Setenv(key, value)
	if err != nil {
		return fmt.Errorf("could not set env for %s, %v", service, err)
	}
	logger.Log.Infof("expose env : %s : %s", key, value)
	return nil
}

func bindWaitPort(e2eConfig *config.E2EConfig, compose *testcontainers.LocalDockerCompose) (map[string][]*hostPortCachedStrategy, error) {
	timeout := e2eConfig.Setup.Timeout
	var waitTimeout time.Duration
	if timeout <= 0 {
		waitTimeout = constant.DefaultWaitTimeout
	} else {
		waitTimeout = time.Duration(timeout) * time.Second
	}
	serviceWithPorts := make(map[string][]*hostPortCachedStrategy)
	for service, content := range compose.Services {
		serviceConfig := content.(map[interface{}]interface{})
		ports := serviceConfig["ports"]
		if ports == nil {
			continue
		}
		serviceWithPorts[service] = []*hostPortCachedStrategy{}

		portList := ports.([]interface{})
		for inx := range portList {
			exportPort, err := getExpectPort(portList[inx])
			if err != nil {
				return nil, err
			}

			strategy := &hostPortCachedStrategy{
				expectPort:       exportPort,
				HostPortStrategy: *wait.NewHostPortStrategy(nat.Port(fmt.Sprintf("%d/tcp", exportPort))).WithStartupTimeout(waitTimeout),
			}
			// temporary don't use testcontainers-go framework wait strategy until fix docker-in-docker bug
			// compose.WithExposedService(service, exportPort, strategy)

			serviceWithPorts[service] = append(serviceWithPorts[service], strategy)
		}
	}
	return serviceWithPorts, nil
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

// hostPortCachedStrategy cached original target
type hostPortCachedStrategy struct {
	wait.HostPortStrategy
	expectPort int
	target     wait.StrategyTarget
}

func (hp *hostPortCachedStrategy) WaitUntilReady(ctx context.Context, target wait.StrategyTarget) error {
	hp.target = target
	return hp.HostPortStrategy.WaitUntilReady(ctx, target)
}

func waitPortUntilReady(e2eConfig *config.E2EConfig, container *types.Container, dockerProvider *DockerProvider, expectPort int) error {
	// wait port
	var waitTimeout time.Duration
	if e2eConfig.Setup.Timeout <= 0 {
		waitTimeout = constant.DefaultWaitTimeout
	} else {
		waitTimeout = time.Duration(e2eConfig.Setup.Timeout) * time.Second
	}
	waitPort := nat.Port(fmt.Sprintf("%d/tcp", expectPort))
	target := &DockerContainer{
		ID:         container.ID,
		WaitingFor: wait.NewHostPortStrategy(waitPort),
		provider:   dockerProvider}
	return WaitPort(context.Background(), target, waitPort, waitTimeout)
}
