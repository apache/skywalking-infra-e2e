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

package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/constant"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"
)

func kindCollect(e2eConfig *config.E2EConfig, collectCfg *config.CollectConfig) error {
	kubeConfigPath := e2eConfig.Setup.GetKubeconfig()
	if kubeConfigPath == "" {
		kubeConfigPath = constant.K8sClusterConfigFilePath
	}

	var errs []string
	for _, item := range collectCfg.Items {
		if err := kindCollectItem(kubeConfigPath, collectCfg.OutputDir, &item); err != nil {
			errs = append(errs, fmt.Sprintf("collect item error: %v", err))
			logger.Log.Warnf("failed to collect item: %v", err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("some collect items failed:\n%s", strings.Join(errs, "\n"))
	}
	return nil
}

func kindCollectItem(kubeConfigPath, outputDir string, item *config.CollectItem) error {
	pods, err := listPods(kubeConfigPath, item)
	if err != nil {
		logger.Log.Warnf("failed to list pods (namespace=%s label-selector=%s resource=%s): %v. "+
			"The cluster or pods may not be ready yet.",
			item.Namespace, item.LabelSelector, item.Resource, err)
		return fmt.Errorf("failed to list pods: %v", err)
	}

	if len(pods) == 0 {
		logger.Log.Warnf("no pods found for namespace=%s label-selector=%s resource=%s. "+
			"Pods may not have been created yet due to setup failure.",
			item.Namespace, item.LabelSelector, item.Resource)
		return nil
	}

	var errs []string
	for _, pod := range pods {
		// Collect kubectl describe output
		if err := collectPodDescribe(kubeConfigPath, outputDir, pod.namespace, pod.name); err != nil {
			logger.Log.Warnf("failed to collect describe for pod %s/%s: %v. Pod may not be ready.", pod.namespace, pod.name, err)
		}

		// Collect specified files
		for _, p := range item.Paths {
			if err := collectPodFile(kubeConfigPath, outputDir, pod.namespace, pod.name, item.Container, p); err != nil {
				errs = append(errs, fmt.Sprintf("pod %s/%s path %s: %v", pod.namespace, pod.name, p, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("some files failed to collect:\n%s", strings.Join(errs, "\n"))
	}
	return nil
}

type podInfo struct {
	namespace string
	name      string
}

func listPods(kubeConfigPath string, item *config.CollectItem) ([]podInfo, error) {
	namespace := item.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// If a specific resource is given (e.g. pod/oap-xxx), use it directly
	if item.Resource != "" {
		parts := strings.SplitN(item.Resource, "/", 2)
		if len(parts) == 2 && parts[0] == "pod" {
			return []podInfo{{namespace: namespace, name: parts[1]}}, nil
		}
		return nil, fmt.Errorf("unsupported resource format: %s (expected pod/<name>)", item.Resource)
	}

	// Use label selector to find pods
	if item.LabelSelector == "" {
		return nil, fmt.Errorf("either resource or label-selector must be specified")
	}

	cmd := fmt.Sprintf("kubectl --kubeconfig %s -n %s get pods -l %s -o jsonpath='{.items[*].metadata.name}'",
		kubeConfigPath, namespace, item.LabelSelector)
	stdout, stderr, err := util.ExecuteCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("kubectl get pods failed: %v, stderr: %s", err, stderr)
	}

	stdout = strings.Trim(stdout, "'")
	names := strings.Fields(stdout)
	pods := make([]podInfo, 0, len(names))
	for _, name := range names {
		if name != "" {
			pods = append(pods, podInfo{namespace: namespace, name: name})
		}
	}
	return pods, nil
}

func collectPodDescribe(kubeConfigPath, outputDir, namespace, podName string) error {
	podDir := filepath.Join(outputDir, namespace, podName)
	if err := os.MkdirAll(podDir, os.ModePerm); err != nil {
		return err
	}

	cmd := fmt.Sprintf("kubectl --kubeconfig %s -n %s describe pod %s",
		kubeConfigPath, namespace, podName)
	stdout, stderr, err := util.ExecuteCommand(cmd)
	if err != nil {
		return fmt.Errorf("kubectl describe failed: %v, stderr: %s", err, stderr)
	}

	descFile := filepath.Join(podDir, "describe.txt")
	if err := os.WriteFile(descFile, []byte(stdout), 0o644); err != nil {
		return fmt.Errorf("failed to write describe output: %v", err)
	}

	logger.Log.Infof("collected describe for pod %s/%s", namespace, podName)
	return nil
}

func collectPodFile(kubeConfigPath, outputDir, namespace, podName, container, srcPath string) error {
	// Preserve the full source path under the pod directory to avoid collisions.
	// e.g. /skywalking/logs/ -> outputDir/namespace/podName/skywalking/logs/
	cleanPath := filepath.Clean(srcPath)
	destPath := filepath.Join(outputDir, namespace, podName, cleanPath)
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return err
	}

	// Build kubectl cp command
	// kubectl cp <namespace>/<pod>:<src> <dest> [-c container]
	src := fmt.Sprintf("%s/%s:%s", namespace, podName, srcPath)
	cmd := fmt.Sprintf("kubectl --kubeconfig %s cp %s %s", kubeConfigPath, src, destPath)
	if container != "" {
		cmd += fmt.Sprintf(" -c %s", container)
	}

	_, stderr, err := util.ExecuteCommand(cmd)
	if err != nil {
		logger.Log.Warnf("failed to collect %s from pod %s/%s: pod/container may not be ready, stderr: %s",
			srcPath, namespace, podName, stderr)
		return fmt.Errorf("kubectl cp failed: %v, stderr: %s", err, stderr)
	}

	logger.Log.Infof("collected %s from pod %s/%s to %s", srcPath, namespace, podName, destPath)
	return nil
}
