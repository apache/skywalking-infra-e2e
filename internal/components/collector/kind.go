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
			paths, err := expandPodGlob(kubeConfigPath, pod.namespace, pod.name, item.Container, p)
			if err != nil {
				errs = append(errs, fmt.Sprintf("pod %s/%s path %s: %v", pod.namespace, pod.name, p, err))
				continue
			}
			for _, expanded := range paths {
				if err := collectPodFile(kubeConfigPath, outputDir, pod.namespace, pod.name, item.Container, expanded); err != nil {
					errs = append(errs, fmt.Sprintf("pod %s/%s path %s: %v", pod.namespace, pod.name, expanded, err))
				}
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

// containsGlob reports whether the path contains glob metacharacters.
func containsGlob(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

// expandPodGlob expands a glob pattern inside a pod. If the path has no glob
// characters it is returned as-is. Otherwise kubectl exec runs sh to expand
// the pattern and returns the matched paths.
func expandPodGlob(kubeConfigPath, namespace, podName, container, pattern string) ([]string, error) {
	if !containsGlob(pattern) {
		return []string{pattern}, nil
	}

	cmd := fmt.Sprintf("kubectl --kubeconfig %s -n %s exec %s", kubeConfigPath, namespace, podName)
	if container != "" {
		cmd += fmt.Sprintf(" -c %s", container)
	}
	cmd += fmt.Sprintf(" -- sh -c 'ls -d %s 2>/dev/null'", pattern)

	stdout, stderr, err := util.ExecuteCommand(cmd)
	if err != nil {
		logger.Log.Warnf("failed to expand glob %s in pod %s/%s: %v, stderr: %s", pattern, namespace, podName, err, stderr)
		return nil, fmt.Errorf("glob expansion failed for %s: %v", pattern, err)
	}

	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}

	if len(paths) == 0 {
		logger.Log.Warnf("glob %s matched no files in pod %s/%s", pattern, namespace, podName)
		return nil, fmt.Errorf("glob %s matched no files", pattern)
	}

	logger.Log.Infof("glob %s expanded to %d path(s) in pod %s/%s", pattern, len(paths), namespace, podName)
	return paths, nil
}

func collectPodFile(kubeConfigPath, outputDir, namespace, podName, container, srcPath string) error {
	// Preserve the full source path under the pod directory to avoid collisions.
	// e.g. /skywalking/logs/ -> outputDir/namespace/podName/skywalking/logs/
	// Strip leading "/" so filepath.Join doesn't discard the prefix.
	cleanPath := filepath.Clean(srcPath)
	cleanPath = strings.TrimLeft(cleanPath, string(filepath.Separator))
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
