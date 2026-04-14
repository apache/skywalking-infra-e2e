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
	"strconv"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
	"gopkg.in/yaml.v2"

	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"
)

// ComposeSetup sets up environment according to e2e.yaml.
func ComposeSetup(e2eConfig *config.E2EConfig) error {
	composeConfigPath := e2eConfig.Setup.GetFile()
	if composeConfigPath == "" {
		return fmt.Errorf("no compose config file was provided")
	}

	// parse compose file to extract services and their ports
	services, err := parseComposeFile(composeConfigPath)
	if err != nil {
		return fmt.Errorf("parse compose file error: %v", err)
	}

	// load environment variables from env file
	if e2eConfig.Setup.InitSystemEnvironment != "" {
		profilePath := util.ResolveAbs(e2eConfig.Setup.InitSystemEnvironment)
		util.ExportEnvVars(profilePath)
	}

	// create compose stack
	identifier := util.GetIdentity()
	stack, err := compose.NewDockerComposeWith(
		compose.WithStackFiles(composeConfigPath),
		compose.StackIdentifier(identifier),
	)
	if err != nil {
		return fmt.Errorf("create compose stack error: %v", err)
	}

	// register wait strategies for each service port
	waitTimeout := e2eConfig.Setup.GetTimeout()
	for _, svc := range services {
		for _, port := range svc.ports {
			stack.WaitForService(svc.name,
				wait.ForListeningPort(fmt.Sprintf("%d/tcp", port)).
					WithStartupTimeout(waitTimeout),
			)
		}
	}

	// pass current environment to compose
	stack.WithOsEnv()

	// bring up the compose stack
	ctx := context.Background()
	if err := stack.Up(ctx, compose.Wait(true)); err != nil {
		return fmt.Errorf("compose up error: %v", err)
	}

	// expose ports and logs for each service
	for _, svc := range services {
		ctr, err := stack.ServiceContainer(ctx, svc.name)
		if err != nil {
			logger.Log.Warnf("could not get container for service %s: %v", svc.name, err)
			continue
		}

		// start log streaming
		if err := startLogStreaming(ctx, ctr, svc.name); err != nil {
			logger.Log.Warnf("could not start log streaming for %s: %v", svc.name, err)
		}

		// export host env
		host, err := ctr.Host(ctx)
		if err != nil {
			return fmt.Errorf("get host for %s error: %v", svc.name, err)
		}
		if err := exportComposeEnv(fmt.Sprintf("%s_host", svc.name), host, svc.name); err != nil {
			return err
		}

		// export port mappings
		for _, port := range svc.ports {
			portStr := fmt.Sprintf("%d/tcp", port)
			mappedPort, err := ctr.MappedPort(ctx, portStr)
			if err != nil {
				return fmt.Errorf("get mapped port %d for %s error: %v", port, svc.name, err)
			}
			if err := exportComposeEnv(
				fmt.Sprintf("%s_%d", svc.name, port),
				strconv.Itoa(int(mappedPort.Num())),
				svc.name,
			); err != nil {
				return err
			}
		}
	}

	// run post-compose setup steps
	if err := RunStepsAndWait(e2eConfig.Setup.Steps, e2eConfig.Setup.GetTimeout(), nil); err != nil {
		logger.Log.Errorf("execute steps error: %v", err)
		return err
	}

	return nil
}

// composeService holds a service name and its container ports.
type composeService struct {
	name  string
	ports []int
}

// parseComposeFile reads the docker-compose YAML and extracts service names and port mappings.
func parseComposeFile(path string) ([]*composeService, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var composeFile struct {
		Services map[string]struct {
			Ports []any `yaml:"ports"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &composeFile); err != nil {
		return nil, err
	}

	var services []*composeService
	for name, svc := range composeFile.Services {
		cs := &composeService{name: name}
		for _, p := range svc.Ports {
			port, err := parseContainerPort(p)
			if err != nil {
				return nil, fmt.Errorf("service %s: %v", name, err)
			}
			cs.ports = append(cs.ports, port)
		}
		services = append(services, cs)
	}
	return services, nil
}

// parseContainerPort extracts the container port from a port config.
// Supports formats: 8080, "8080", "8080:80", "0.0.0.0:8080:80"
func parseContainerPort(portConfig any) (int, error) {
	switch conf := portConfig.(type) {
	case int:
		return conf, nil
	case string:
		parts := strings.Split(conf, ":")
		// last part is always the container port (possibly with /protocol)
		containerPart := parts[len(parts)-1]
		containerPart = strings.Split(containerPart, "/")[0] // remove /tcp, /udp
		return strconv.Atoi(containerPart)
	}
	return 0, fmt.Errorf("unknown port format: %v", portConfig)
}

// fileLogConsumer writes container logs to a file.
type fileLogConsumer struct {
	writer  *os.File
	service string
}

func (f *fileLogConsumer) Accept(log testcontainers.Log) {
	if _, err := f.writer.Write(log.Content); err != nil {
		logger.Log.Warnf("write %s log error: %v", f.service, err)
	}
}

// startLogStreaming begins streaming container logs to the log directory.
func startLogStreaming(ctx context.Context, ctr *testcontainers.DockerContainer, serviceName string) error {
	writer, err := logFollower.BuildLogWriter(fmt.Sprintf("%s/std.log", serviceName))
	if err != nil {
		return err
	}
	consumer := &fileLogConsumer{writer: writer, service: serviceName}
	//nolint:staticcheck // FollowOutput/StartLogProducer are deprecated but the replacement
	// (ContainerRequest.LogConsumerConfig) is not available for containers obtained from compose stacks.
	ctr.FollowOutput(consumer)
	//nolint:staticcheck
	return ctr.StartLogProducer(ctx)
}

func exportComposeEnv(key, value, service string) error {
	if err := os.Setenv(key, value); err != nil {
		return fmt.Errorf("could not set env for %s, %v", service, err)
	}
	logger.Log.Infof("export %s=%s", key, value)
	return nil
}
