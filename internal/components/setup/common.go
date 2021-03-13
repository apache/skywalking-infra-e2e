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
	"fmt"
	"io/ioutil"
	"time"

	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"
)

// RunCommandsAndWait Concurrently run commands and wait for conditions.
func RunCommandsAndWait(runs []config.Run, timeout time.Duration) error {
	waitSet := util.NewWaitSet(timeout)

	for idx := range runs {
		run := runs[idx]
		commands := run.Command
		if len(commands) < 1 {
			continue
		}

		waitSet.WaitGroup.Add(1)
		go executeCommandsAndWait(commands, run.Waits, waitSet)
	}

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

func executeCommandsAndWait(commands string, waits []config.Wait, waitSet *util.WaitSet) {
	defer waitSet.WaitGroup.Done()

	// executes commands
	logger.Log.Infof("executing commands [%s]", commands)
	result, err := util.ExecuteCommand(commands)
	if err != nil {
		err = fmt.Errorf("commands: [%s] runs error: %s", commands, err)
		waitSet.ErrChan <- err
	}
	logger.Log.Infof("executed commands [%s], result: %s", commands, result)

	// waits for conditions meet
	for idx := range waits {
		wait := waits[idx]
		logger.Log.Infof("waiting for %+v", wait)

		kubeConfigYaml, err := ioutil.ReadFile(kubeConfigPath)
		if err != nil {
			err = fmt.Errorf("read kube config failed: %s", err)
			waitSet.ErrChan <- err
		}

		options, err := getWaitOptions(kubeConfigYaml, &wait)
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
