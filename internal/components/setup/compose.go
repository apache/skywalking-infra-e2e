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

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/testcontainers/testcontainers-go"
)

const (
	// SeparatorV1 is the separator used in docker-compose v1
	// refer to https://github.com/docker/compose/blob/5becea4ca9f68875334c92f191a13482bcd6e5cf/compose/service.py#L1492-L1498
	SeparatorV1 = "_"
	// SeparatorV2 is the separator used in docker-compose v2
	// refer to https://github.com/docker/compose/blob/981aea674d052ee1ab252f71c3ca1f9f8a7e32de/pkg/compose/convergence.go#L252-L257
	SeparatorV2 = "-"
)

var (
	containerNamePattern = regexp.MustCompile(`.*_(?P<containerNum>\d+)$`)
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
	services, err := buildComposeServices(e2eConfig, compose)
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

	// Listen container create
	listener := NewComposeContainerListener(context.Background(), cli, services)
	defer listener.Stop()
	err = listener.Listen(func(container *ComposeContainer) {
		if err = exposeComposeLog(cli, container.Service, container.ID, logFollower); err == nil {
			container.Service.beenFollowLog = true
		}
	})
	if err != nil {
		return err
	}

	// setup
	execError := compose.WithCommand(cmd).Invoke()
	if execError.Error != nil {
		return execError.Error
	}

	// find exported port and build env
	err = exposeComposeService(services, cli, identifier, e2eConfig)
	if err != nil {
		return err
	}

	// run steps
	err = RunStepsAndWait(e2eConfig.Setup.Steps, e2eConfig.Setup.GetTimeout(), nil)
	if err != nil {
		logger.Log.Errorf("execute steps error: %v", err)
		return err
	}

	return nil
}

type ComposeService struct {
	Name           string
	waitStrategies []*hostPortCachedStrategy
	beenFollowLog  bool
}

func exposeComposeService(services []*ComposeService, cli *client.Client,
	identity string, e2eConfig *config.E2EConfig) error {
	dockerProvider := &DockerProvider{client: cli}

	// find exported port and build env
	for _, service := range services {
		// expose port
		if err := exposeComposePort(dockerProvider, service, cli, identity, e2eConfig); err != nil {
			return err
		}

		// if service log not follow, expose log
		if !service.beenFollowLog {
			c, err := service.FindContainer(cli, identity)
			if err != nil {
				logger.Log.Warn(err)
				continue
			}
			if err := exposeComposeLog(dockerProvider.client, service, c.ID, logFollower); err != nil {
				return err
			}
			service.beenFollowLog = true
		}
	}
	return nil
}

func (c *ComposeService) FindContainer(cli *client.Client, identity string) (*types.Container, error) {
	serviceName, num := getInstanceName(c.Name)
	return findContainer(cli, identity, serviceName, num)
}

func exposeComposePort(dockerProvider *DockerProvider, service *ComposeService, cli *client.Client, identity string,
	e2eConfig *config.E2EConfig) error {
	if len(service.waitStrategies) == 0 {
		return nil
	}

	// get real ip address for access and export to env
	host, err := dockerProvider.daemonHost(context.Background())
	if err != nil {
		return err
	}

	container, err := service.FindContainer(cli, identity)
	if err != nil {
		return err
	}

	// format: <service_name>_host
	if err := exportComposeEnv(fmt.Sprintf("%s_host", service.Name), host, service.Name); err != nil {
		return err
	}

	for inx := range service.waitStrategies {
		for _, containerPort := range container.Ports {
			if int(containerPort.PrivatePort) != service.waitStrategies[inx].expectPort {
				continue
			}

			if err := waitPortUntilReady(e2eConfig, container, dockerProvider, service.waitStrategies[inx].expectPort); err != nil {
				return err
			}

			// expose env config to env
			// format: <service_name>_<port>
			if err := exportComposeEnv(
				fmt.Sprintf("%s_%d", service.Name, containerPort.PrivatePort),
				fmt.Sprintf("%d", containerPort.PublicPort),
				service.Name); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

// export container log to local path
func exposeComposeLog(cli *client.Client, service *ComposeService, containerID string, logFollower *util.ResourceLogFollower) error {
	logs, err := cli.ContainerLogs(logFollower.Ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Details:    false,
	})
	if err != nil {
		return err
	}
	writer, err := logFollower.BuildLogWriter(fmt.Sprintf("%s/std.log", service.Name))
	if err != nil {
		return err
	}

	go func() {
		defer writer.Close()
		if _, err := stdcopy.StdCopy(writer, writer, logs); err != nil {
			logger.Log.Warnf("write %s std log error: %v", service.Name, err)
		}
	}()
	return nil
}

func exportComposeEnv(key, value, service string) error {
	err := os.Setenv(key, value)
	if err != nil {
		return fmt.Errorf("could not set env for %s, %v", service, err)
	}
	logger.Log.Infof("export %s=%s", key, value)
	return nil
}

func buildComposeServices(e2eConfig *config.E2EConfig, compose *testcontainers.LocalDockerCompose) ([]*ComposeService, error) {
	waitTimeout := e2eConfig.Setup.GetTimeout()
	services := make([]*ComposeService, 0)
	for service, content := range compose.Services {
		serviceConfig := content.(map[any]any)
		ports := serviceConfig["ports"]
		serviceContext := &ComposeService{Name: service}
		services = append(services, serviceContext)
		if ports == nil {
			continue
		}

		portList := ports.([]any)
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
			serviceContext.waitStrategies = append(serviceContext.waitStrategies, strategy)
		}
	}
	return services, nil
}

func getExpectPort(portConfig any) (int, error) {
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

func findContainer(c *client.Client, projectName, serviceName string, number int) (*types.Container, error) {
	nameV1 := strings.Join([]string{projectName, serviceName, strconv.Itoa(number)}, SeparatorV1)
	nameV2 := strings.Join([]string{projectName, serviceName, strconv.Itoa(number)}, SeparatorV2)
	// filter either names
	// 1) {project}_{service}_{number}
	// 2) {project}-{service}-{number}
	f := filters.NewArgs(filters.Arg("name", nameV1), filters.Arg("name", nameV2))
	containerListOptions := types.ContainerListOptions{Filters: f}
	containers, err := c.ContainerList(context.Background(), containerListOptions)
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("could not found container: %s(docker-compose v1) or %s(docker-compose v2)", nameV1, nameV2)
	}
	return &containers[0], nil
}

func getInstanceName(serviceName string) (service string, number int) {
	matches := containerNamePattern.FindStringSubmatch(serviceName)
	if len(matches) == 0 {
		return serviceName, 1
	}
	numberStr := matches[0]
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return serviceName, 1
	}
	return serviceName, number
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
	waitTimeout := e2eConfig.Setup.GetTimeout()
	waitPort := nat.Port(fmt.Sprintf("%d/tcp", expectPort))
	target := &DockerContainer{
		ID:         container.ID,
		WaitingFor: wait.NewHostPortStrategy(waitPort),
		provider:   dockerProvider}
	return WaitPort(context.Background(), target, waitPort, waitTimeout)
}
