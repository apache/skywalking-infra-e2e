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

func composeCollect(e2eConfig *config.E2EConfig, collectCfg *config.CollectConfig) error {
	composeFile := e2eConfig.Setup.GetFile()
	if composeFile == "" {
		return fmt.Errorf("compose file not configured in setup.file")
	}
	projectName := util.GetIdentity()

	var errs []string
	for _, item := range collectCfg.Items {
		if err := composeCollectItem(composeFile, projectName, collectCfg.OutputDir, &item); err != nil {
			errs = append(errs, fmt.Sprintf("collect item error: %v", err))
			logger.Log.Warnf("failed to collect item for service %s: %v", item.Service, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("some collect items failed:\n%s", strings.Join(errs, "\n"))
	}
	return nil
}

func composeCollectItem(composeFile, projectName, outputDir string, item *config.CollectItem) error {
	if item.Service == "" {
		return fmt.Errorf("service name is required for compose collect items")
	}

	serviceDir := filepath.Join(outputDir, item.Service)
	if err := os.MkdirAll(serviceDir, os.ModePerm); err != nil {
		return err
	}

	// Find container ID using the compose file and project name that setup used
	containerID, err := findComposeContainer(composeFile, projectName, item.Service)
	if err != nil {
		logger.Log.Warnf("failed to find container for service %s: container may not be running yet. %v", item.Service, err)
		return fmt.Errorf("failed to find container for service %s: %v", item.Service, err)
	}

	// Collect docker inspect output
	if err := collectContainerInspect(outputDir, item.Service, containerID); err != nil {
		logger.Log.Warnf("failed to collect inspect for service %s: container may not be ready. %v", item.Service, err)
	}

	// Collect specified files
	var errs []string
	for _, p := range item.Paths {
		paths, err := expandContainerGlob(containerID, item.Service, p)
		if err != nil {
			errs = append(errs, fmt.Sprintf("service %s path %s: %v", item.Service, p, err))
			continue
		}
		for _, expanded := range paths {
			if err := collectContainerFile(outputDir, item.Service, containerID, expanded); err != nil {
				errs = append(errs, fmt.Sprintf("service %s path %s: %v", item.Service, expanded, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("some files failed to collect:\n%s", strings.Join(errs, "\n"))
	}
	return nil
}

// findComposeContainer locates the container ID for a service using the same
// compose file and project name that setup/cleanup use.
func findComposeContainer(composeFile, projectName, service string) (string, error) {
	cmd := fmt.Sprintf("%s -f %s -p %s ps -q %s", constant.ComposeCommand, composeFile, projectName, service)
	stdout, stderr, err := util.ExecuteCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("docker compose ps failed: %v, stderr: %s", err, stderr)
	}

	containerID := strings.TrimSpace(stdout)
	if containerID == "" {
		return "", fmt.Errorf("no container found for service %s (project: %s, file: %s)", service, projectName, composeFile)
	}
	return containerID, nil
}

func collectContainerInspect(outputDir, service, containerID string) error {
	serviceDir := filepath.Join(outputDir, service)
	if err := os.MkdirAll(serviceDir, os.ModePerm); err != nil {
		return err
	}

	cmd := fmt.Sprintf("docker inspect %s", containerID)
	stdout, stderr, err := util.ExecuteCommand(cmd)
	if err != nil {
		return fmt.Errorf("docker inspect failed: %v, stderr: %s", err, stderr)
	}

	inspectFile := filepath.Join(serviceDir, "inspect.json")
	if err := os.WriteFile(inspectFile, []byte(stdout), 0o644); err != nil {
		return fmt.Errorf("failed to write inspect output: %v", err)
	}

	logger.Log.Infof("collected inspect for service %s", service)
	return nil
}

// expandContainerGlob expands a glob pattern inside a Docker container.
// If the path has no glob characters it is returned as-is.
func expandContainerGlob(containerID, service, pattern string) ([]string, error) {
	if !containsGlob(pattern) {
		return []string{pattern}, nil
	}

	if err := validateGlobPattern(pattern); err != nil {
		return nil, err
	}

	cmd := fmt.Sprintf("docker exec %s sh -c 'ls -d -- %s 2>/dev/null || true'", containerID, pattern)
	stdout, stderr, err := util.ExecuteCommand(cmd)
	if err != nil {
		logger.Log.Warnf("failed to expand glob %s in service %s: %v, stderr: %s", pattern, service, err, stderr)
		return nil, fmt.Errorf("glob expansion failed for %s: %v, stderr: %s", pattern, err, stderr)
	}

	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}

	if len(paths) == 0 {
		logger.Log.Warnf("glob %s matched no files in service %s", pattern, service)
		return nil, fmt.Errorf("glob %s matched no files", pattern)
	}

	logger.Log.Infof("glob %s expanded to %d path(s) in service %s", pattern, len(paths), service)
	return paths, nil
}

func collectContainerFile(outputDir, service, containerID, srcPath string) error {
	// Preserve the full source path under the service directory to avoid collisions.
	// e.g. /var/log/nginx/ -> outputDir/serviceName/var/log/nginx/
	// Strip leading "/" so filepath.Join doesn't discard the prefix.
	cleanPath := filepath.Clean(srcPath)
	cleanPath = strings.TrimLeft(cleanPath, string(filepath.Separator))
	destPath := filepath.Join(outputDir, service, cleanPath)
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return err
	}

	cmd := fmt.Sprintf("docker cp %s:%s %s", containerID, srcPath, destPath)
	_, stderr, err := util.ExecuteCommand(cmd)
	if err != nil {
		logger.Log.Warnf("failed to collect %s from service %s: container may not be ready, stderr: %s",
			srcPath, service, stderr)
		return fmt.Errorf("docker cp failed: %v, stderr: %s", err, stderr)
	}

	logger.Log.Infof("collected %s from service %s to %s", srcPath, service, destPath)
	return nil
}
