# Module Design

## Controller

The controller means composing all the steps declared in the configuration file, it progressive and display which step is currently running.
If it failed in a step, the error message could be shown, as much comprehensive as possible. An example of the output might be.

```
e2e run
✔ Started Kind Cluster - Cluster Name
✔ Checked Pods Readiness - All pods are ready
? Generating Traffic - HTTP localhost:9090/users (progress spinner)
✔ Verified Output - service ls
(progress spinner) Verifying Output - endpoint ls
✘ Failed to Verify Output Data - endpoint ls
  <the diff content>
✔ Clean Up
```

Compared with running the steps one by one, the controller is also responsible for cleaning up the environment (by executing the cleanup command) no matter what status other commands are, even if they are failed, the controller has the following semantics in terms of setup and cleanup.

```
// Java
try {
    setup();
    // trigger step
    // verify step
    // ...
} finally {
    cleanup();
}

// GoLang
func run() {
    setup();
    defer cleanup();
    // trigger step
    // verify step
    // ...
}
```

## Steps

According to the content in the Controller, E2E Testing can be divided into the following steps.

### Setup

Start the environment required for this E2E Testing, such as database, back-end process, API, etc.

Support two ways to set up the environment:
- **compose**:
  1. Start the `docker-compose` services.
  1. Check the services' healthiness.
  1. Wait until all services are ready according to the interval, etc.
  1. Execute command to set up the testing environment or help verify, such as `yq` help to eval the YAML format.
- **kind**:
  1. Start the `KinD` cluster according to the config files or Start on an existing kubernetes cluster.
  1. Apply the resources files (`--manifests`) or/and run the custom init command (`--commands`).
  1. Check the pods' readiness.
  1. Wait until all pods are ready according to the `interval`, etc.

### Trigger

Generate traffic by trigger the action, It could access `HTTP API` or execute `commands` with interval.

It could have these settings:
1. **interval**: How frequency to trigger the action.
1. **times**: How many times the operation is triggered before aborting on the condition that the trigger had failed always. `0=infinite`.
1. **action**: The action of the trigger.

### Verify

Verify that the data content is matching with the expected results. such as unit test assert, etc.

It could have these settings:
1. **actual**: The actual data file.
1. **query**: The query to get the actual data, could run shell commands to generate the data.
1. **expected**: The expected data file, could specify some matching rules to verify the actual content.

### Cleanup

This step requires the same options in the setup step so that it can clean up all things necessarily. Such as destroy the environment, etc.
