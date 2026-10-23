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
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/containerd/platforms"
	docker "github.com/moby/moby/client"
	"github.com/moby/moby/client/pkg/versions"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	kind "sigs.k8s.io/kind/cmd/kind/app"
	kindcmd "sigs.k8s.io/kind/pkg/cmd"
	kindexec "sigs.k8s.io/kind/pkg/exec"

	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"
)

// imageLoadMode is how images on the docker host are loaded into the kind nodes.
type imageLoadMode int

const (
	// loadDockerImage runs `kind load docker-image`: kind saves the whole image
	// and the nodes import it with `ctr images import --all-platforms`.
	loadDockerImage imageLoadMode = iota
	// loadPlatformArchive saves only the platform of the kind nodes, leaves the
	// referrers of the image, such as attestations, out of the archive, then
	// runs `kind load image-archive`. With docker's containerd image store, a
	// pulled multi-platform image only holds the content of the platform it was
	// pulled for, so importing all of its platforms fails with
	// "content digest ... not found" (kubernetes-sigs/kind#3795).
	loadPlatformArchive
)

const (
	// containerdSnapshotterDriverType is the "driver-type" docker reports in
	// its driver status when the containerd image store is in use.
	containerdSnapshotterDriverType = "io.containerd.snapshotter.v1"
	// minPlatformSaveAPIVersion is the first docker API version (docker 28.0)
	// that accepts a platform when saving an image.
	minPlatformSaveAPIVersion = "1.48"
)

// usesContainerdImageStore reports whether the docker driver status says the
// containerd image store is in use.
func usesContainerdImageStore(driverStatus [][2]string) bool {
	for _, status := range driverStatus {
		if status[0] == "driver-type" && status[1] == containerdSnapshotterDriverType {
			return true
		}
	}
	return false
}

// chooseImageLoadMode picks the load mode from the image store and the API
// version the docker client uses. The classic image store keeps the
// `kind load docker-image` path unchanged.
func chooseImageLoadMode(containerdStore bool, apiVersion string) imageLoadMode {
	if containerdStore && !versions.LessThan(apiVersion, minPlatformSaveAPIVersion) {
		return loadPlatformArchive
	}
	return loadDockerImage
}

// nodePlatform is the platform of the kind nodes, which run on the docker
// host's architecture.
func nodePlatform(arch string) ocispec.Platform {
	return platforms.Normalize(ocispec.Platform{OS: "linux", Architecture: arch})
}

// importImages loads the images on the docker host into the kind cluster.
func importImages(ctx context.Context, images []string, clusterName string) error {
	cli, err := docker.New(docker.FromEnv)
	if err != nil {
		return err
	}
	defer func() {
		if err := cli.Close(); err != nil {
			logger.Log.Warnf("failed to close docker client: %v", err)
		}
	}()

	// negotiate first, so ClientVersion is the API version requests really use
	if _, err := cli.Ping(ctx, docker.PingOptions{NegotiateAPIVersion: true}); err != nil {
		return fmt.Errorf("ping docker daemon error: %w", err)
	}
	info, err := cli.Info(ctx, docker.InfoOptions{})
	if err != nil {
		return fmt.Errorf("get docker info error: %w", err)
	}
	version, err := cli.ServerVersion(ctx, docker.ServerVersionOptions{})
	if err != nil {
		return fmt.Errorf("get docker version error: %w", err)
	}

	containerdStore := usesContainerdImageStore(info.Info.DriverStatus)
	apiVersion := cli.ClientVersion()
	mode := chooseImageLoadMode(containerdStore, apiVersion)
	platform := nodePlatform(version.Arch)
	switch {
	case mode == loadPlatformArchive:
		logger.Log.Infof("docker %s (API %s) uses the containerd image store, images are saved for %s before loading into kind",
			version.Version, apiVersion, platforms.Format(platform))
	case containerdStore:
		logger.Log.Warnf("docker %s (API %s) uses the containerd image store, but saving a single platform needs API %s, "+
			"multi-platform images may fail to load into kind", version.Version, apiVersion, minPlatformSaveAPIVersion)
	}

	for _, image := range images {
		logger.Log.Infof("import docker images: %s", image)
		if mode == loadPlatformArchive {
			err = loadImageArchive(ctx, cli, image, platform, clusterName)
		} else {
			err = runKindLoad("load", "docker-image", image, "--name", clusterName)
			if containerdStore && isMissingContentError(err) {
				err = fmt.Errorf("%w (docker's containerd image store holds only part of this image; "+
					"use docker 28 or later, or disable the containerd-snapshotter feature of docker)", err)
			}
		}
		if err != nil {
			return fmt.Errorf("import docker image %s error: %w", image, err)
		}
	}
	return nil
}

// loadImageArchive saves the platform of the image to a temporary archive,
// without its referrers, and loads it with `kind load image-archive`.
func loadImageArchive(ctx context.Context, cli *docker.Client, image string, platform ocispec.Platform, clusterName string) error {
	archive, err := os.CreateTemp(util.WorkDir, "kind-image-*.tar")
	if err != nil {
		return fmt.Errorf("create image archive error: %w", err)
	}
	defer func() {
		if err := os.Remove(archive.Name()); err != nil && !os.IsNotExist(err) {
			logger.Log.Warnf("failed to remove image archive %s: %v", archive.Name(), err)
		}
	}()

	if err := saveImage(ctx, cli, image, platform, archive); err != nil {
		_ = archive.Close()
		return err
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("write image archive error: %w", err)
	}
	dropped, err := dropReferrers(archive.Name())
	if err != nil {
		return err
	}
	if dropped > 0 {
		logger.Log.Infof("left %d referrer(s) of %s, such as attestations, out of the archive", dropped, image)
	}
	return runKindLoad("load", "image-archive", archive.Name(), "--name", clusterName)
}

func saveImage(ctx context.Context, cli *docker.Client, image string, platform ocispec.Platform, dst io.Writer) error {
	saved, err := cli.ImageSave(ctx, []string{image}, docker.ImageSaveWithPlatforms(platform))
	if err != nil {
		return fmt.Errorf("save image for %s error, the local image may not hold this platform, "+
			"pull it with `docker pull --platform %s %s`: %w", platforms.Format(platform), platforms.Format(platform), image, err)
	}
	defer func() {
		if err := saved.Close(); err != nil {
			logger.Log.Warnf("failed to close image save output: %v", err)
		}
	}()
	if _, err := io.Copy(dst, saved); err != nil {
		return fmt.Errorf("save image for %s error: %w", platforms.Format(platform), err)
	}
	return nil
}

// runKindLoad runs a kind load command. kind leaves the output of the failed
// node command out of the error, so it is added back.
func runKindLoad(args ...string) error {
	err := kind.Run(kindcmd.NewLogger(), kindcmd.StandardIOStreams(), args)
	if err == nil {
		return nil
	}
	if output := kindCommandOutput(err); output != "" {
		return &kindLoadError{err: err, output: output}
	}
	return err
}

// kindLoadError is a failed kind load with the output of the node command.
type kindLoadError struct {
	err    error
	output string
}

func (e *kindLoadError) Error() string {
	return fmt.Sprintf("%v: %s", e.err, e.output)
}

func (e *kindLoadError) Unwrap() error {
	return e.err
}

func kindCommandOutput(err error) string {
	var runErr *kindexec.RunError
	if !errors.As(err, &runErr) {
		// kind's own errors may only chain through Cause()
		if runErr = kindexec.RunErrorForError(err); runErr == nil {
			return ""
		}
	}
	return strings.TrimSpace(string(runErr.Output))
}

// isMissingContentError reports whether the nodes failed to import an image
// because the archive references content the docker host does not hold.
func isMissingContentError(err error) bool {
	var loadErr *kindLoadError
	if !errors.As(err, &loadErr) {
		return false
	}
	return strings.Contains(loadErr.output, "content digest") && strings.Contains(loadErr.output, "not found")
}
