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
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	ociIndexFile = "index.json"
	// manifestSubjectAnnotation marks a top-level descriptor of an image archive
	// as a referrer of another manifest, such as an attestation or a signature.
	manifestSubjectAnnotation = "io.containerd.manifest.subject"
	imageNameAnnotation       = "io.containerd.image.name"
)

// dropReferrers removes the referrers from the top-level index of an image
// archive saved by docker, and leaves the archive untouched when it has none.
//
// docker's containerd image store saves the referrers of an image, such as its
// attestations and their signatures, beside it in the top-level index, marked
// with manifestSubjectAnnotation. kind imports the archive with `ctr images
// import --all-platforms --digests`: containerd 1.7 (node images built by kind
// v0.20 to v0.26) creates an image for every top-level descriptor and unpacks
// it, and as a referrer has no filesystem layers, the import fails with
// "mismatched image rootfs and manifest layers"; containerd 2 only unpacks
// manifests that have filesystem layers. A descriptor that names an image is
// kept even when marked. The blobs of the referrers stay in the archive,
// unreferenced.
func dropReferrers(path string) (int, error) {
	data, err := readArchiveFile(path, ociIndexFile)
	if err != nil || data == nil {
		return 0, err
	}
	var index map[string]json.RawMessage
	if err := json.Unmarshal(data, &index); err != nil {
		return 0, fmt.Errorf("parse %s of the image archive error: %w", ociIndexFile, err)
	}
	var manifests []json.RawMessage
	if raw, ok := index["manifests"]; ok {
		if err := json.Unmarshal(raw, &manifests); err != nil {
			return 0, fmt.Errorf("parse %s of the image archive error: %w", ociIndexFile, err)
		}
	}

	// The kept descriptors are copied as saved, so no field of theirs is lost.
	kept := make([]json.RawMessage, 0, len(manifests))
	for _, m := range manifests {
		var desc struct {
			Annotations map[string]string `json:"annotations"`
		}
		if err := json.Unmarshal(m, &desc); err != nil {
			return 0, fmt.Errorf("parse %s of the image archive error: %w", ociIndexFile, err)
		}
		_, referrer := desc.Annotations[manifestSubjectAnnotation]
		_, named := desc.Annotations[imageNameAnnotation]
		_, refNamed := desc.Annotations[ocispec.AnnotationRefName]
		if !referrer || named || refNamed {
			kept = append(kept, m)
		}
	}
	dropped := len(manifests) - len(kept)
	if dropped == 0 {
		return 0, nil
	}

	if index["manifests"], err = json.Marshal(kept); err != nil {
		return 0, err
	}
	newIndex, err := json.Marshal(index)
	if err != nil {
		return 0, err
	}
	return dropped, rewriteArchiveFile(path, ociIndexFile, newIndex)
}

// readArchiveFile returns the content of a file in a tar archive, or nil when
// the archive does not have it.
func readArchiveFile(path, name string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open image archive error: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read image archive error: %w", err)
		}
		if strings.TrimPrefix(hdr.Name, "./") == name {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("read %s of the image archive error: %w", name, err)
			}
			return data, nil
		}
	}
}

// rewriteArchiveFile replaces the content of one file of a tar archive, copying
// every other entry as it is.
func rewriteArchiveFile(path, name string, content []byte) error {
	dst, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("create image archive error: %w", err)
	}
	err = copyArchive(path, dst, name, content)
	if closeErr := dst.Close(); err == nil && closeErr != nil {
		err = fmt.Errorf("write image archive error: %w", closeErr)
	}
	// Both archives are closed by now: Windows cannot replace an open file.
	if err == nil {
		err = os.Rename(dst.Name(), path)
	}
	if err != nil {
		_ = os.Remove(dst.Name())
	}
	return err
}

func copyArchive(path string, dst io.Writer, name string, content []byte) error {
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open image archive error: %w", err)
	}
	defer func() {
		_ = src.Close()
	}()
	tr := tar.NewReader(src)
	tw := tar.NewWriter(dst)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read image archive error: %w", err)
		}
		var body io.Reader = tr
		if strings.TrimPrefix(hdr.Name, "./") == name {
			hdr.Size = int64(len(content))
			body = bytes.NewReader(content)
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("write image archive error: %w", err)
		}
		if _, err := io.Copy(tw, body); err != nil {
			return fmt.Errorf("write image archive error: %w", err)
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("write image archive error: %w", err)
	}
	return nil
}
