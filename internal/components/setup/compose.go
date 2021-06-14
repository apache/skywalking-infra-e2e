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
	"net"
	"os"
	"regexp"
	"syscall"
	"time"

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
	execError := compose.WithCommand([]string{"up", "-d"}).Invoke()
	if execError.Error != nil {
		return execError.Error
	}

	// record time now
	timeNow := time.Now()
	timeout := e2eConfig.Setup.Timeout
	var waitTimeout time.Duration
	if timeout <= 0 {
		waitTimeout = constant.DefaultWaitTimeout
	} else {
		waitTimeout = time.Duration(timeout) * time.Second
	}
	logger.Log.Debugf("wait timeout is %d seconds", int(waitTimeout.Seconds()))

	// find exported port and build env
	for service, content := range compose.Services {
		serviceConfig := content.(map[interface{}]interface{})
		ports := serviceConfig["ports"]
		if ports == nil {
			continue
		}
		portList := ports.([]interface{})
		container, err := findContainer(cli, fmt.Sprintf("%s_%s", identifier, getInstanceName(service)))
		if err != nil {
			return err
		}
		containerPorts := container.Ports

		for inx := range portList {
			for _, containerPort := range containerPorts {
				if int(containerPort.PrivatePort) != portList[inx].(int) {
					continue
				}

				// calculate max wait time
				waitTimeout = NewTimeout(timeNow, waitTimeout)
				timeNow = time.Now()

				// wait port and export
				err := waitTCPPortStarted(context.Background(), cli, container, int(containerPort.PublicPort), int(containerPort.PrivatePort), waitTimeout)
				if err != nil {
					return fmt.Errorf("could wait port exported: %s:%d, %v", service, portList[inx], err)
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
			}
		}
	}

	return nil
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

func waitTCPPortStarted(ctx context.Context, c *client.Client, container *types.Container, publicPort, interPort int, timeout time.Duration) error {
	// limit context to startupTimeout
	ctx, cancelContext := context.WithTimeout(ctx, timeout)
	defer cancelContext()

	var waitInterval = 100 * time.Millisecond

	// external check
	dialer := net.Dialer{}
	address := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", publicPort))
	for {
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			if v, ok := err.(*net.OpError); ok {
				if v2, ok := (v.Err).(*os.SyscallError); ok {
					if isConnRefusedErr(v2.Err) {
						time.Sleep(waitInterval)
						continue
					}
				}
			}
			return err
		}
		conn.Close()
		break
	}

	// internal check
	command := buildInternalCheckCommand(interPort)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		response, err := c.ContainerExecCreate(ctx, container.ID, types.ExecConfig{
			Cmd:          []string{"/bin/sh", "-c", command},
			AttachStderr: true,
			AttachStdout: true,
		})
		if err != nil {
			return err
		}

		err = c.ContainerExecStart(ctx, response.ID, types.ExecStartCheck{
			Detach: false,
		})
		if err != nil {
			return err
		}

		var exitCode int
		for {
			execResp, err := c.ContainerExecInspect(ctx, response.ID)
			if err != nil {
				return err
			}

			if !execResp.Running {
				exitCode = execResp.ExitCode
				break
			}

			time.Sleep(waitInterval)
		}

		if exitCode == 0 {
			return nil
		}
	}
}

func buildInternalCheckCommand(internalPort int) string {
	command := `(
					nc -vz -w 1 localhost %d || 
					cat /proc/net/tcp | awk '{print $2}' | grep -i :%d || 
					</dev/tcp/localhost/%d
				)
				`
	return "true && " + fmt.Sprintf(command, internalPort, internalPort, internalPort)
}

func isConnRefusedErr(err error) bool {
	return err == syscall.ECONNREFUSED
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
