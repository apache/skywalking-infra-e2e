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
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type archiveEntry struct {
	name string
	data []byte
}

func blobEntry(t *testing.T, v any) (archiveEntry, ocispec.Descriptor) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	d := digest.FromBytes(data)
	return archiveEntry{name: "blobs/sha256/" + d.Encoded(), data: data},
		ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: d, Size: int64(len(data))}
}

func writeArchive(t *testing.T, entries []archiveEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "image.tar")
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: e.name, Mode: 0o644, Size: int64(len(e.data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(e.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readArchive(t *testing.T, path string) map[string][]byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	files := map[string][]byte{}
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return files
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		files[hdr.Name] = data
	}
}

// savedImage is an archive shaped like `docker save --platform` writes with the
// containerd image store: the platform manifest, named, then its attestation
// and the attestation's signature, both marked as referrers.
func savedImage(t *testing.T) (path string, image json.RawMessage, entries []archiveEntry) {
	t.Helper()
	imageBlob, imageDesc := blobEntry(t, ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Layers:    []ocispec.Descriptor{{MediaType: ocispec.MediaTypeImageLayerGzip, Digest: digest.FromString("layer"), Size: 5}},
	})
	attestationBlob, attestation := blobEntry(t, ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Layers:    []ocispec.Descriptor{{MediaType: "application/vnd.in-toto+json", Digest: digest.FromString("provenance"), Size: 10}},
	})
	attestation.Annotations = map[string]string{manifestSubjectAnnotation: imageDesc.Digest.String()}
	signatureBlob, signature := blobEntry(t, ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Layers:    []ocispec.Descriptor{{MediaType: "application/vnd.dev.cosign.simplesigning.v1+json", Digest: digest.FromString("payload"), Size: 7}},
	})
	signature.ArtifactType = "application/vnd.dev.cosign.artifact.sig.v1+json"
	signature.Annotations = map[string]string{manifestSubjectAnnotation: attestation.Digest.String()}

	// A field ocispec.Descriptor does not know, which must survive the rewrite.
	image = json.RawMessage(`{"mediaType":"` + ocispec.MediaTypeImageManifest + `","digest":"` + imageDesc.Digest.String() +
		`","size":` + strconv.FormatInt(imageDesc.Size, 10) + `,"annotations":{"` + imageNameAnnotation + `":"ghcr.io/example/app:1",` +
		`"` + ocispec.AnnotationRefName + `":"1"},"platform":{"architecture":"amd64","os":"linux"},"x-extension":true}`)
	index, err := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     ocispec.MediaTypeImageIndex,
		"manifests":     []any{image, attestation, signature},
	})
	if err != nil {
		t.Fatal(err)
	}
	entries = []archiveEntry{
		imageBlob,
		attestationBlob,
		signatureBlob,
		{name: "oci-layout", data: []byte(`{"imageLayoutVersion":"1.0.0"}`)},
		{name: "manifest.json", data: []byte(`[{"Config":"blobs/sha256/x","RepoTags":["ghcr.io/example/app:1"],"Layers":[]}]`)},
		{name: ociIndexFile, data: index},
	}
	return writeArchive(t, entries), image, entries
}

func TestDropReferrers(t *testing.T) {
	path, image, entries := savedImage(t)
	dropped, err := dropReferrers(path)
	if err != nil {
		t.Fatal(err)
	}
	if dropped != 2 {
		t.Fatalf("dropped = %d, want 2", dropped)
	}
	files := readArchive(t, path)
	var index map[string]json.RawMessage
	if err := json.Unmarshal(files[ociIndexFile], &index); err != nil {
		t.Fatal(err)
	}
	var manifests []json.RawMessage
	if err := json.Unmarshal(index["manifests"], &manifests); err != nil {
		t.Fatal(err)
	}
	if len(manifests) != 1 || !bytes.Equal(manifests[0], image) {
		t.Fatalf("manifests = %s, want only the image %s", index["manifests"], image)
	}
	if string(index["schemaVersion"]) != "2" || string(index["mediaType"]) != `"`+ocispec.MediaTypeImageIndex+`"` {
		t.Errorf("index fields changed: %s", files[ociIndexFile])
	}
	for _, e := range entries {
		if e.name == ociIndexFile {
			continue
		}
		if !bytes.Equal(files[e.name], e.data) {
			t.Errorf("%s changed", e.name)
		}
	}
	left, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 {
		t.Errorf("the rewrite left files behind: %v", left)
	}
}

func TestDropReferrersKeepsNamedImages(t *testing.T) {
	subject := digest.FromString("subject").String()
	descriptor := func(name string, annotations map[string]string) ocispec.Descriptor {
		return ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromString(name), Size: 1, Annotations: annotations}
	}
	named := descriptor("named", map[string]string{imageNameAnnotation: "ghcr.io/example/app:1", manifestSubjectAnnotation: subject})
	refNamed := descriptor("ref-named", map[string]string{ocispec.AnnotationRefName: "2", manifestSubjectAnnotation: subject})
	referrer := descriptor("referrer", map[string]string{manifestSubjectAnnotation: named.Digest.String()})
	index, err := json.Marshal(ocispec.Index{Manifests: []ocispec.Descriptor{named, refNamed, referrer}})
	if err != nil {
		t.Fatal(err)
	}
	path := writeArchive(t, []archiveEntry{{name: ociIndexFile, data: index}})
	dropped, err := dropReferrers(path)
	if err != nil || dropped != 1 {
		t.Fatalf("dropReferrers() = %d, %v, want 1, nil", dropped, err)
	}
	var kept ocispec.Index
	if err := json.Unmarshal(readArchive(t, path)[ociIndexFile], &kept); err != nil {
		t.Fatal(err)
	}
	if len(kept.Manifests) != 2 || kept.Manifests[0].Digest != named.Digest || kept.Manifests[1].Digest != refNamed.Digest {
		t.Fatalf("manifests = %+v, want the two named ones", kept.Manifests)
	}
}

func TestDropReferrersLeavesOtherArchivesUntouched(t *testing.T) {
	imageBlob, image := blobEntry(t, ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Layers:    []ocispec.Descriptor{{MediaType: ocispec.MediaTypeImageLayerGzip, Digest: digest.FromString("layer"), Size: 5}},
	})
	index, err := json.Marshal(ocispec.Index{Manifests: []ocispec.Descriptor{image}})
	if err != nil {
		t.Fatal(err)
	}
	for name, entries := range map[string][]archiveEntry{
		"no referrers":         {imageBlob, {name: ociIndexFile, data: index}},
		"docker legacy format": {{name: "manifest.json", data: []byte(`[]`)}},
	} {
		t.Run(name, func(t *testing.T) {
			path := writeArchive(t, entries)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			dropped, err := dropReferrers(path)
			if err != nil || dropped != 0 {
				t.Fatalf("dropReferrers() = %d, %v, want 0, nil", dropped, err)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Error("the archive changed")
			}
		})
	}
}
