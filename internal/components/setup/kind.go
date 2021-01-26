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
	"os"
	"strings"
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctlwait "k8s.io/kubectl/pkg/cmd/wait"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/apache/skywalking-infra-e2e/internal/config"

	apiv1 "k8s.io/api/admission/v1"

	"github.com/apache/skywalking-infra-e2e/internal/util"

	kind "sigs.k8s.io/kind/cmd/kind/app"
	kindcmd "sigs.k8s.io/kind/pkg/cmd"

	"github.com/apache/skywalking-infra-e2e/internal/constant"

	"github.com/apache/skywalking-infra-e2e/internal/flags"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

var (
	kindConfigFile string
	kubeConfigPath string
)

// KindSetup sets up environment according to e2e.yaml.
func KindSetup(e2eConfig *config.E2EConfig) error {
	kindConfigFile = e2eConfig.Setup.File

	timeout := e2eConfig.Setup.Timeout
	var waitTimeout time.Duration
	if timeout == 0 {
		waitTimeout = constant.DefaultWaitTimeout
	} else {
		waitTimeout = time.Duration(timeout) * time.Second
	}

	logger.Log.Debugf("wait timeout is %d seconds", int(waitTimeout.Seconds()))

	if kindConfigFile == "" {
		return fmt.Errorf("no kind config file was provided")
	}

	manifests := e2eConfig.Setup.Manifests
	// if no manifests was provided, then no need to create the cluster.
	if manifests == nil {
		logger.Log.Info("no manifests is provided")
		return nil
	}

	if err := createKindCluster(); err != nil {
		return err
	}

	c, dc, err := util.ConnectToK8sCluster(kubeConfigPath)
	if err != nil {
		logger.Log.Errorf("connect to k8s cluster failed according to config file: %s", kubeConfigPath)
		return err
	}

	err = createManifestsAndWait(c, dc, manifests, waitTimeout)
	if err != nil {
		return err
	}
	return nil
}

// KindSetupInCommand hasn't completed yet.
func KindSetupInCommand() error {
	kindConfigFile = flags.File
	manifests := flags.Manifests

	if err := createKindCluster(); err != nil {
		return err
	}

	c, dc, err := util.ConnectToK8sCluster(kubeConfigPath)
	if err != nil {
		logger.Log.Errorf("connect to k8s cluster failed according to config file: %s", kubeConfigPath)
		return err
	}

	files, err := util.GetManifests(manifests)
	if err != nil {
		logger.Log.Error("get manifests from command line argument failed")
		return err
	}

	for _, f := range files {
		logger.Log.Infof("creating manifest %s", f)
		err = util.OperateManifest(c, dc, f, apiv1.Create)
		if err != nil {
			logger.Log.Errorf("create manifest %s in k8s cluster failed", f)
			return err
		}
	}

	return nil
}

func createKindCluster() error {
	// the config file name of the k8s cluster that kind create
	kubeConfigPath = constant.K8sClusterConfigFile
	args := []string{"create", "cluster", "--config", kindConfigFile, "--kubeconfig", kubeConfigPath}

	logger.Log.Info("creating kind cluster...")
	logger.Log.Debugf("cluster create commands: %s %s", constant.KindCommand, strings.Join(args, " "))
	if err := kind.Run(kindcmd.NewLogger(), kindcmd.StandardIOStreams(), args); err != nil {
		return err
	}
	logger.Log.Info("create kind cluster succeeded")
	return nil
}

// createManifestsAndWait creates manifests in k8s cluster and concurrent waits according to the manifests' wait conditions.
func createManifestsAndWait(c *kubernetes.Clientset, dc dynamic.Interface, manifests []config.Manifest, timeout time.Duration) error {
	waitSet := util.NewWaitSet(timeout)

	kubeConfigYaml, err := ioutil.ReadFile(kubeConfigPath)
	if err != nil {
		return err
	}

	for idx := range manifests {
		manifest := manifests[idx]
		waits := manifest.Waits
		err := createByManifest(c, dc, manifest)
		if err != nil {
			return err
		}

		if waits == nil {
			logger.Log.Info("no wait-for strategy is provided")
			continue
		}

		for _, wait := range waits {
			if strings.Contains(wait.Resource, "/") && wait.LabelSelector != "" {
				return fmt.Errorf("when passing resource.group/resource.name in Resource, the labelSelector can not be set at the same time")
			}

			logger.Log.Infof("waiting for %+v", wait)

			restClientGetter := util.NewSimpleRESTClientGetter(wait.Namespace, string(kubeConfigYaml))
			silenceOutput, _ := os.Open(os.DevNull)
			ioStreams := genericclioptions.IOStreams{In: os.Stdin, Out: silenceOutput, ErrOut: os.Stderr}
			waitFlags := ctlwait.NewWaitFlags(restClientGetter, ioStreams)
			// global timeout is set in e2e.yaml
			waitFlags.Timeout = constant.AWeekWaitTimeout
			waitFlags.ForCondition = wait.For

			var args []string
			// resource.group/resource.name OR resource.group
			if wait.Resource != "" {
				args = append(args, wait.Resource)
			} else {
				return fmt.Errorf("resource must be provided in wait block")
			}

			if wait.LabelSelector != "" {
				waitFlags.ResourceBuilderFlags.LabelSelector = &wait.LabelSelector
			} else if !strings.Contains(wait.Resource, "/") {
				// if labelSelector is nil and resource only provide resource.group, check all resources.
				waitFlags.ResourceBuilderFlags.All = &constant.True
			}

			options, err := waitFlags.ToOptions(args)
			if err != nil {
				return err
			}

			waitSet.WaitGroup.Add(1)
			go concurrentlyWait(wait, options, waitSet)
		}
	}

	go func() {
		waitSet.WaitGroup.Wait()
		close(waitSet.FinishChan)
	}()

	select {
	case <-waitSet.FinishChan:
		logger.Log.Infof("create and wait for manifests ready success")
	case err := <-waitSet.ErrChan:
		logger.Log.Errorf("failed to wait for manifests to be ready")
		return err
	case <-time.After(waitSet.Timeout):
		return fmt.Errorf("wait for manifests ready timeout after %d seconds", int(timeout.Seconds()))
	}

	return nil
}

func createByManifest(c *kubernetes.Clientset, dc dynamic.Interface, manifest config.Manifest) error {
	files, err := util.GetManifests(manifest.Path)
	if err != nil {
		logger.Log.Error("get manifests from command line argument failed")
		return err
	}

	for _, f := range files {
		logger.Log.Infof("creating manifest %s", f)
		err = util.OperateManifest(c, dc, f, apiv1.Create)
		if err != nil {
			logger.Log.Errorf("create manifest %s failed", f)
			return err
		}
	}
	return nil
}

func concurrentlyWait(wait config.Wait, options *ctlwait.WaitOptions, waitSet *util.WaitSet) {
	defer waitSet.WaitGroup.Done()

	err := options.RunWait()
	if err != nil {
		err = fmt.Errorf("wait strategy :%+v, err: %s", wait, err)
		waitSet.ErrChan <- err
	}
	logger.Log.Infof("wait %+v condition met", wait)
}
