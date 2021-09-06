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

```yaml
- name: Run E2E Test
  uses: apache/skywalking-infra-e2e@main      # always prefer to use a revision instead of `main`.
  with:
    e2e-file: e2e.yaml                        # need to run E2E file path
```