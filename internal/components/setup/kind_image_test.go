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
	"errors"
	"fmt"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	kinderrors "sigs.k8s.io/kind/pkg/errors"
	kindexec "sigs.k8s.io/kind/pkg/exec"
)

func TestUsesContainerdImageStore(t *testing.T) {
	tests := []struct {
		name         string
		driverStatus [][2]string
		want         bool
	}{
		{name: "containerd image store", driverStatus: [][2]string{{"driver-type", "io.containerd.snapshotter.v1"}}, want: true},
		{name: "classic overlay2", driverStatus: [][2]string{{"Backing Filesystem", "extfs"}, {"Supports d_type", "true"}}},
		{name: "no driver status", driverStatus: nil},
		{name: "other driver type", driverStatus: [][2]string{{"driver-type", "io.containerd.snapshotter.v2"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := usesContainerdImageStore(tt.driverStatus); got != tt.want {
				t.Errorf("usesContainerdImageStore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChooseImageLoadMode(t *testing.T) {
	tests := []struct {
		name            string
		containerdStore bool
		apiVersion      string
		want            imageLoadMode
	}{
		{name: "docker 28 classic store", apiVersion: "1.48", want: loadDockerImage},
		{name: "docker 29 classic store", apiVersion: "1.54", want: loadDockerImage},
		{name: "docker 29 containerd store", containerdStore: true, apiVersion: "1.54", want: loadPlatformArchive},
		{name: "docker 28 containerd store", containerdStore: true, apiVersion: "1.48", want: loadPlatformArchive},
		{name: "docker 27 containerd store", containerdStore: true, apiVersion: "1.47", want: loadDockerImage},
		{name: "API pinned below 1.48", containerdStore: true, apiVersion: "1.44", want: loadDockerImage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chooseImageLoadMode(tt.containerdStore, tt.apiVersion); got != tt.want {
				t.Errorf("chooseImageLoadMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodePlatform(t *testing.T) {
	tests := []struct {
		arch string
		want ocispec.Platform
	}{
		{arch: "amd64", want: ocispec.Platform{OS: "linux", Architecture: "amd64"}},
		{arch: "x86_64", want: ocispec.Platform{OS: "linux", Architecture: "amd64"}},
		{arch: "arm64", want: ocispec.Platform{OS: "linux", Architecture: "arm64"}},
		{arch: "aarch64", want: ocispec.Platform{OS: "linux", Architecture: "arm64"}},
		{arch: "arm", want: ocispec.Platform{OS: "linux", Architecture: "arm", Variant: "v7"}},
	}
	for _, tt := range tests {
		t.Run(tt.arch, func(t *testing.T) {
			if got := nodePlatform(tt.arch); got.OS != tt.want.OS || got.Architecture != tt.want.Architecture || got.Variant != tt.want.Variant {
				t.Errorf("nodePlatform(%q) = %+v, want %+v", tt.arch, got, tt.want)
			}
		})
	}
}

func TestIsMissingContentError(t *testing.T) {
	runErr := func(output string) error {
		return fmt.Errorf("failed to load image: %w", &kindexec.RunError{
			Command: []string{"ctr", "images", "import"},
			Output:  []byte(output),
			Inner:   errors.New("exit status 1"),
		})
	}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "missing content",
			err:  &kindLoadError{err: runErr(""), output: "ctr: content digest sha256:bfb54eac: not found"},
			want: true,
		},
		{
			name: "other ctr failure",
			err:  &kindLoadError{err: runErr(""), output: "ctr: failed to extract layer: no space left on device"},
		},
		{name: "wrapped missing content", err: fmt.Errorf("import: %w", &kindLoadError{output: "ctr: content digest sha256:1: not found"}), want: true},
		{name: "plain error", err: errors.New("content digest sha256:1: not found")},
		{name: "nil", err: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMissingContentError(tt.err); got != tt.want {
				t.Errorf("isMissingContentError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKindCommandOutput(t *testing.T) {
	err := fmt.Errorf("failed to load image: %w", &kindexec.RunError{
		Command: []string{"ctr", "images", "import"},
		Output:  []byte("ctr: content digest sha256:1: not found\n"),
		Inner:   errors.New("exit status 1"),
	})
	if got, want := kindCommandOutput(err), "ctr: content digest sha256:1: not found"; got != want {
		t.Errorf("kindCommandOutput() = %q, want %q", got, want)
	}
	// kind wraps the failed node command with its own errors package
	kindErr := kinderrors.Wrap(&kindexec.RunError{Output: []byte("ctr: failed\n"), Inner: errors.New("exit status 1")}, "failed to load image")
	if got, want := kindCommandOutput(kindErr), "ctr: failed"; got != want {
		t.Errorf("kindCommandOutput() = %q, want %q", got, want)
	}
	if got := kindCommandOutput(errors.New("no command")); got != "" {
		t.Errorf("kindCommandOutput() = %q, want empty", got)
	}
}
