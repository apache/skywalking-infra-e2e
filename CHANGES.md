Changes by Version
==================
Release Notes.

Unreleased
------------------
#### Bug Fixes

* Upgrade Go dependencies affected by known security advisories: `golang.org/x/crypto`, `github.com/containerd/containerd/v2`, `google.golang.org/grpc`, `github.com/moby/go-archive`, `github.com/moby/spdystream`, `github.com/moby/buildkit`, and the OpenTelemetry SDK and OTLP trace exporters.
* Upgrade `testcontainers-go` to v0.44.0 so Docker Compose no longer imports a BuildKit package removed in the patched release.
* Use `github.com/moby/moby/client` for Docker Engine API calls. `github.com/docker/docker` has no patched release for its advisories.
* Fix `setup.kind.import-images` failing when Docker uses the containerd image store (the default from Docker 29, e.g. on `ubuntu-26.04` runners). Images are now saved for the kind nodes' platform, so a multi-platform image no longer fails with `ctr: content digest ... not found`, and the referrers saved beside an image, such as its attestations and their signatures, are removed from the archive's top-level index: a containerd 1.7 node (node images built by kind v0.20 to v0.26) otherwise imports each one as an image and fails with `ctr: mismatched image rootfs and manifest layers`. The archive is loaded with `kind load image-archive`; Docker with the classic image store keeps `kind load docker-image`.

1.4.0
------------------
#### Features

* Adding `setup.kind.no-wait` to support should wait for the kind cluster to be ready or not.
* Support importing external variables in the `setup.init-system-environment` file.

#### Bug Fixes

* Fix kind load docker-image error
* Fix the wrong judgement when not all the range is including.

#### Issues and PR
- All issues are [here](https://github.com/apache/skywalking/milestone/148?closed=1)
- All pull requests are [here](https://github.com/apache/skywalking-infra-e2e/milestone/4?closed=1)
