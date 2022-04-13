# Install SkyWalking Infra E2E

## Download pre-built binaries

Download the pre-built binaries from [our website](https://skywalking.apache.org/downloads/#SkyWalkingInfraE2E),
currently we have pre-built binaries for macOS, Linux and Windows. Extract the tarball and add `bin/<os>/e2e`
to you `PATH` environment variable.

## Install from source codes

If you want to try some features that are not released yet, you can compile from the source code.

```shell
mkdir skywalking-infra-e2e && cd skywalking-infra-e2e
git clone https://github.com/apache/skywalking-infra-e2e.git .
make build
```

Then add the binary in `bin/<os>/e2e` to your `PATH`.

## Install via `go install`

If you already have Go SDK installed, you can also directly install `e2e` via `go install`.

```shell
go install github.com/apache/skywalking-infra-e2e/cmd/e2e@<revision>
```

Note that installation via `go install` is only supported after Git commit [2a33478](https://github.com/apache/skywalking-infra-e2e/commit/2a3347824633780b9e785d03124709170e1b9f08)
so you can only `go install` a revision afterwards.
