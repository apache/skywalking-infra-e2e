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
	"strings"
	"time"

	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"
)

var (
	logFollower *util.ResourceLogFollower
)

func RunStepsAndWait(steps []config.Step, waitTimeout time.Duration, k8sCluster *util.K8sClusterInfo) error {
	logger.Log.Debugf("wait timeout is %v", waitTimeout.String())

	// record time now
	timeNow := time.Now()

	for _, step := range steps {
		logger.Log.Infof("processing setup step [%s]", step.Name)

		if step.Path != "" && step.Command == "" {
			if k8sCluster == nil {
				return fmt.Errorf("not support path")
			}
			manifest := config.Manifest{
				Path:  step.Path,
				Waits: step.Waits,
			}
			err := createManifestAndWait(k8sCluster, manifest, waitTimeout)
			if err != nil {
				return err
			}
		} else if step.Command != "" && step.Path == "" {
			command := config.Run{
				Command: step.Command,
				Waits:   step.Waits,
			}

			err := RunCommandsAndWait(command, waitTimeout, k8sCluster)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("step parameter error, one Path or one Command should be specified, but got %+v", step)
		}

		waitTimeout = NewTimeout(timeNow, waitTimeout)
		timeNow = time.Now()

		if waitTimeout <= 0 {
			return fmt.Errorf("setup timeout")
		}
	}
	return nil
}

// createManifestAndWait creates manifests in k8s cluster and concurrent waits according to the manifests' wait conditions.
func createManifestAndWait(c *util.K8sClusterInfo, manifest config.Manifest, timeout time.Duration) error {
	waitSet := util.NewWaitSet(timeout)

	waits := manifest.Waits
	err := createByManifest(c, manifest)
	if err != nil {
		return err
	}

	// len() for nil slices is defined as zero
	if len(waits) == 0 {
		logger.Log.Info("no wait-for strategy is provided")
		return nil
	}

	for idx := range waits {
		wait := waits[idx]
		logger.Log.Infof("waiting for %+v", wait)

		options, err := getWaitOptions(c, &wait)
		if err != nil {
			return err
		}

		waitSet.WaitGroup.Add(1)
		go concurrentlyWait(&wait, options, waitSet)
	}

	go func() {
		waitSet.WaitGroup.Wait()
		close(waitSet.FinishChan)
	}()

	select {
	case <-waitSet.FinishChan:
		logger.Log.Infof("create and wait for manifest ready success")
	case err := <-waitSet.ErrChan:
		logger.Log.Errorf("failed to wait for manifest to be ready")
		return err
	case <-time.After(waitSet.Timeout):
		return fmt.Errorf("wait for manifest ready timeout after %d seconds", int(timeout.Seconds()))
	}

	return nil
}

// RunCommandsAndWait Concurrently run commands and wait for conditions.
func RunCommandsAndWait(run config.Run, timeout time.Duration, cluster *util.K8sClusterInfo) error {
	waitSet := util.NewWaitSet(timeout)

	commands := run.Command
	if len(commands) < 1 {
		return nil
	}

	waitSet.WaitGroup.Add(1)
	go executeCommandsAndWait(commands, run.Waits, waitSet, cluster)

	go func() {
		waitSet.WaitGroup.Wait()
		close(waitSet.FinishChan)
	}()

	select {
	case <-waitSet.FinishChan:
		logger.Log.Infof("all commands executed successfully")
	case err := <-waitSet.ErrChan:
		logger.Log.Errorf("execute command error")
		return err
	case <-time.After(waitSet.Timeout):
		return fmt.Errorf("wait for commands run timeout after %d seconds", int(timeout.Seconds()))
	}

	return nil
}

func executeCommandsAndWait(commands string, waits []config.Wait, waitSet *util.WaitSet, cluster *util.K8sClusterInfo) {
	defer waitSet.WaitGroup.Done()

	// executes commands
	logger.Log.Infof("executing commands [%s]", strings.ReplaceAll(commands, "\n", "\\n"))
	result, stderr, err := util.ExecuteCommand(commands)
	if err != nil {
		err = fmt.Errorf("commands: [%s] runs error: %s", strings.ReplaceAll(commands, "\n", "\\n"), stderr)
		waitSet.ErrChan <- err
	}
	logger.Log.Infof("executed commands [%s], result: %s", strings.ReplaceAll(commands, "\n", "\\n"), result)

	// waits for conditions meet
	for idx := range waits {
		wait := waits[idx]
		logger.Log.Infof("waiting for %+v", wait)

		options, err := getWaitOptions(cluster, &wait)
		if err != nil {
			err = fmt.Errorf("commands: [%s] get wait options error: %s", commands, err)
			waitSet.ErrChan <- err
		}

		err = options.RunWait()
		if err != nil {
			err = fmt.Errorf("commands: [%s] waits error: %s", commands, err)
			waitSet.ErrChan <- err
			return
		}
		logger.Log.Infof("wait %+v condition met", wait)
	}
}

// NewTimeout calculates new timeout since timeBefore.
func NewTimeout(timeBefore time.Time, timeout time.Duration) time.Duration {
	elapsed := time.Since(timeBefore)
	newTimeout := timeout - elapsed
	return newTimeout
}

func GetIdentity() string {
	runID := os.Getenv("GITHUB_RUN_ID")
	if runID == "" {
		return "skywalking_e2e"
	}
	return runID
}

func InitLogFollower() {
	logFollower = util.NewResourceLogFollower(context.Background(), util.LogDir)
}

func CloseLogFollower() {
	if logFollower != nil {
		logFollower.Close()
	}
}
