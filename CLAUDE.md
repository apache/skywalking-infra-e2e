# CLAUDE.md - Project Guide for skywalking-infra-e2e

## Project Overview

Apache SkyWalking Infra E2E is a CLI tool for end-to-end testing. It orchestrates test environments
(Kubernetes/Kind or Docker Compose), generates traffic, verifies results, and cleans up.

## Build & Test Commands

```bash
make all              # clean + lint + test + build
make test             # run unit tests with coverage
make lint             # run golangci-lint (auto-installs if missing)
make build            # build for windows/linux/darwin
make darwin           # build for macOS only (use your current OS target)
make e2e-test         # run e2e test with Docker Compose (test/e2e/e2e.yaml)
make e2e-test-kind    # run e2e test with Kind (test/e2e/kind/e2e.yaml)
```

- Go module: `github.com/apache/skywalking-infra-e2e`
- Entry point: `cmd/e2e/main.go`
- Binary output: `bin/<os>/e2e`
- Version injected via ldflags at build time

## Architecture

### CLI Commands (Cobra)

| Command       | File                          | Purpose                        |
|---------------|-------------------------------|--------------------------------|
| `e2e run`     | `commands/run/run.go`         | Full lifecycle orchestration   |
| `e2e setup`   | `commands/setup/setup.go`     | Setup env only (debug mode)    |
| `e2e trigger` | `commands/trigger/trigger.go` | Run trigger only               |
| `e2e verify`  | `commands/verify/verify.go`   | Run verification only          |
| `e2e cleanup` | `commands/cleanup/cleanup.go` | Run cleanup only               |

Global flags defined in `commands/root.go`:
- `-c, --config` (default: `e2e.yaml`)
- `-v, --verbosity` (debug/info/warn/error)
- `-w, --work-dir` (default: `~/.skywalking-infra-e2e`)
- `-l, --log-dir` (default: `~/.skywalking-infra-e2e/logs`)

### Lifecycle (`e2e run`)

```
setup → trigger → verify → cleanup (deferred)
```

Cleanup runs via Go `defer` and is controlled by `cleanup.on`:
- `always` / `success` / `failure` / `never`
- Default: `success` locally, `always` in CI (`CI=true` env var)
- Constants in `internal/constant/cleanup.go`

### Environment Modes

Determined by `setup.env` in e2e.yaml (`"kind"` or `"compose"`).

Constants: `constant.Kind` and `constant.Compose` in `internal/constant/`.

**Kind mode** (`internal/components/setup/kind.go`):
- Creates Kind cluster, loads Docker images, applies K8s manifests
- Pod log streaming via K8s client-go
- Port forwarding via SPDY
- Cleanup: `kind delete cluster` with retry (up to 5x)

**Compose mode** (`internal/components/setup/compose.go`):
- Uses testcontainers-go for Docker Compose
- Container log streaming
- Cleanup: `docker-compose down`

### Configuration

Config struct: `internal/config/e2eConfig.go` → `E2EConfig`

```yaml
setup:
  env: kind|compose
  file: path/to/kind-config.yaml  # or docker-compose.yml
  kubeconfig: path                # alternative to file (use existing cluster)
  timeout: 20m
  steps:
    - name: step-name
      path: manifest.yaml         # or command: "shell cmd"
      wait:
        - namespace: default
          resource: pod
          label-selector: app=foo
          for: condition=Ready
  kind:
    import-images: [image:tag]
    expose-ports:
      - namespace: default
        resource: pod/name
        port: "8080"

cleanup:
  on: always|success|failure|never

trigger:
  action: http
  interval: 3s
  times: 5
  url: http://...
  method: GET

verify:
  retry: { count: 10, interval: 10s }
  fail-fast: true
  concurrency: false
  cases:
    - name: case-name
      query: "shell command"    # or actual: path/to/file
      expected: path/to/expected.yaml
```

### Key Packages

| Package                              | Role                                      |
|--------------------------------------|-------------------------------------------|
| `internal/config/`                   | YAML config parsing, global config state  |
| `internal/components/setup/`         | Kind & Compose setup implementations      |
| `internal/components/trigger/`       | HTTP trigger action                       |
| `internal/components/verifier/`      | Test case verification with retry         |
| `internal/components/cleanup/`       | Kind & Compose cleanup implementations    |
| `internal/util/`                     | K8s client, Docker helpers, env/log utils |
| `internal/constant/`                 | Constants for both modes and cleanup       |
| `internal/logger/`                   | Logrus-based logging                      |
| `pkg/output/`                        | Test result formatting (YAML/summary)     |
| `third-party/go/template/`          | Extended Go template functions for verify |

### Test Structure

**Unit tests** (6 files):
- `internal/config/e2eConfig_test.go`
- `internal/util/config_test.go`, `utils_test.go`
- `commands/verify/verify_test.go`
- `internal/components/verifier/verifier_test.go`
- `third-party/go/template/funcs_test.go`

**E2E tests** (`test/e2e/`):
- `e2e.yaml` — Compose-based e2e test
- `kind/e2e.yaml` — Kind-based e2e test
- Verify scenarios under `concurrency/` and `non-concurrency/` dirs

### Log Collection

- Logs streamed during setup to `LogDir/namespace/podName.log` (Kind) or `LogDir/serviceName/std.log` (Compose)
- `internal/util/env_log.go` — `ResourceLogFollower` manages log writers
- GitHub Actions: log dir defaults to `${runner.temp}/skywalking-infra-e2e/logs`
- **No existing mechanism to copy arbitrary files from containers on failure**

### GitHub Actions Integration

- `action.yaml` at project root defines the composite action
- Inputs: e2e-file, log-dir, plus matrix vars for log isolation