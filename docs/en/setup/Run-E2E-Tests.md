# SkyWalking Infra E2E Execute Guide

There are two ways to perform E2E Testing:
1. Command: Suitable for local debugging and operation.
1. GitHub Action: Suitable for automated execution in GitHub projects.

## Command

Through commands, you can execute a complete Controller.

```shell
# e2e.yaml configuration file in current directory
e2e run

# or 

# Specified the e2e.yaml file path
e2e run -c /path/to/the/test/e2e.yaml
```

Also, could run the separate step in the command line, these commands are all done by reading the configuration.

```shell
e2e setup
e2e trigger
e2e verify
e2e cleanup
```

## GitHub Action

To use skywalking-infra-e2e in GitHub Actions, add a step in your GitHub workflow.

The working directory is uploaded to GitHub Action Artifact after the task is completed, which contains environment variables and container logs in the environment.

```yaml
- name: Run E2E Test
  uses: apache/skywalking-infra-e2e@main      # always prefer to use a revision instead of `main`.
  with:
    e2e-file: e2e.yaml                        # (required)need to run E2E file path
    log-dir: /path/to/log/dir                 # (not required)the container logs path, if not provide it would be auto generation
```

If you want to upload the log directory to the GitHub Action Artifact when this E2E test failure, you could define the below content in your GitHub Action Job.

```yaml
- name: Upload E2E Log
  uses: actions/upload-artifact@v2
  if: ${{ failure() }}                      # Only upload the artifact when E2E testing failure
  with:
    name: e2e-log
    path: "${{ env.SW_INFRA_E2E_LOG_DIR }}" # The SkyWalking Infra E2E action would provide this environment
```