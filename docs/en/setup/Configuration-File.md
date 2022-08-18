# SkyWalking Infra E2E Configuration Guide

The configuration file is used to integrate all the step configuration content.
You can see the sample configuration files for different environments in the [examples directory](../../../examples).

There is a quick view about the configuration file, and using the `yaml` format.
```yaml
setup:
  # set up the environment
cleanup:
  # clean up the environment
trigger:
  # generate traffic
verify:
  # test cases
```

## Setup

Support two kinds of the environment to set up the system.

### KinD

```yaml
setup:
  env: kind
  file: path/to/kind.yaml               # Specified kinD manifest file path
  timeout: 20m                          # timeout duration
  init-system-environment: path/to/env  # Import environment file
  steps:                                # customize steps for prepare the environment
    - name: customize setups            # step name
      # one of command line or kinD manifest file
      command: command lines            # use command line to setup 
      path: /path/to/manifest.yaml      # the manifest file path
      wait:                             # how to verify the manifest is set up finish
        - namespace:                    # The pod namespace
          resource:                     # The pod resource name
          label-selector:               # The resource label selector
          for:                          # The wait condition
  kind:
     import-images:                     # import docker images to KinD
        - image:version                 # support using env to expand image, such as `${env_key}` or `$env_key`
     expose-ports:                      # Expose resource for host access
        - namespace:                    # The resource namespace
          resource:                     # The resource name, such as `pod/foo` or `service/foo`
          port:                         # Want to expose port from resource
```

The `KinD` environment follow these steps:
1. Start the `KinD` cluster according to the config file, expose `KUBECONFIG` to environment for help execute `kubectl` in the steps.
1. Load docker images from `kind.import-images` if needed.
1. Apply the resources files (`--manifests`) or/and run the custom init command (`--commands`) by steps.
1. Wait until all steps are finished and all services are ready with the timeout(second).
1. Expose all resource ports for host access.

#### Import docker image

If you want to import docker image from private registries, there are several ways to do this:
1. Using `imagePullSecrets` to pull images, [please take reference from document](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#registry-secret-existing-credentials).
2. Using `kind.import-images` to load images from host.
   ```yaml
   kind:
      import-images:
        - skywalking/oap:${OAP_HASH} # support using environment to expand the image name
   ```

#### Resource Export

If you want to access the resource from host, should follow these steps:
1. Declare which resource and ports need to be accessible from host.
   ```yaml
   setup:
      kind:
         expose-ports:
           - namespace: default  # Need to expose resource namespace
             resource: pod/foo   # Resource description, such as `pod/foo` or `service/foo`
             port: 8080          # Resource port want to expose, support `<resource_port>`, `<bind_to_host_port>:<resource_port>`
   ```
2. Follow this format to get the host and port mapping by the environment, and it's available in steps(trigger, verify).
   ```yaml
   trigger:
      # trigger with specified mapped port, the resource name replace all `/` or `-` as `_`
      # host format: <resource_name>_host
      # port format: <resource_name>_<container_port>
      url: http://${pod_foo_host}:${pod_foo_8080}/
   ```

#### Log

The console output of each pod could be found in `${workDir}/logs/${namespace}/${podName}.log`.

### Compose

```yaml
setup:
  env: compose
  file: path/to/compose.yaml            # Specified docker-compose file path
  timeout: 20m                          # Timeout duration
  init-system-environment: path/to/env  # Import environment file
  steps:                                # Customize steps for prepare the environment
    - name: customize setups            # Step name
      command: command lines            # Use command line to setup 
```

The `docker-compose` environment follow these steps:
1. Import `init-system-environment` file for help build service and execute steps. 
Each line of the file content is an environment variable, and the key value is separate by "=".
1. Start the `docker-compose` services.
1. Check the services' healthiness.
1. Wait until all services are ready according to the interval, etc.
1. Execute command to set up the testing environment or help verify.

#### Service Export
If you want to get the service host and port mapping, should follow these steps:
1. declare the port in the `docker-compose` service `ports` config.
   ```yaml
   oap:
    image: xx.xx:1.0.0
    ports:
        # define the port
        - 8080
   ```
1. Follow this format to get the host and port mapping by the environment, and it's available in steps(trigger, verify).
   ```yaml
   trigger:
      # trigger with specified mappinged port
      url: http://${oap_host}:${oap_8080}/
   ```

#### Log

The console output of each service could be found in `${workDir}/logs/{serviceName}/std.log`.

## Trigger

After the `Setup` step is finished, use the `Trigger` step to generate traffic.

```yaml
trigger:
  action: http      # The action of the trigger. support HTTP invoke.
  interval: 3s      # Trigger the action every 3 seconds.
  times: 5          # The retry count before the request success.
  url: http://apache.skywalking.com/ # Http trigger url link.
  method: GET       # Http trigger method.
  headers:
    "Content-Type": "application/json"
    "Authorization": "Basic whatever"
  body: '{"k1":"v1", "k2":"v2"}'
```

The Trigger executed successfully at least once, after success, the next stage could be continued. Otherwise, there is an error and exit.

## Verify

After the `Trigger` step is finished, running test cases.

```yaml
verify:
  retry:            # verify with retry strategy
    count: 10       # max retry count
    interval: 10s   # the interval between two attempts, e.g. 10s, 1m.
  fail-fast: true  # when a case fails, whether to stop verifying other cases. This property defaults to true.
  concurrency: false # whether to verify cases concurrently. This property defaults to false.
  cases:            # verify test cases
    - actual: path/to/actual.yaml       # verify by actual file path
      expected: path/to/expected.yaml   # excepted content file path
    - query: echo 'foo'                 # verify by command execute output
      expected: path/to/expected.yaml   # excepted content file path
    - includes:      # including cases
        - path/to/cases.yaml            # cases file path
```

The test cases are executed in the order of declaration from top to bottom. When the execution of a case fails and the retry strategy is exceeded, it will stop verifying other cases if `fail-fast` is `true`. Otherwise,  the process will continue to verify other cases.

### Retry strategy

The retry strategy could retry automatically on the test case failure, and restart by the failed test case.

### Case source

Support two kind source to verify, one case only supports one kind source type:

1. source file: verify by generated `yaml` format file.
2. command: use command line output as they need to verify content, also only support `yaml` format.

### Excepted verify template

After clarifying the content that needs to be verified, you need to write content to verify the real content and ensure that the data is correct.

You need to use the form of [Go Template](https://pkg.go.dev/text/template#pkg-overview) to write the verification file, and the data content to be rendered comes from the real data. By verifying whether the rendered data is consistent with the real data, it is verified whether the content is consistent.
You could see [many test cases in this directory](../../../test/verify).

We use [go-cmp](https://pkg.go.dev/github.com/google/go-cmp/cmp#Diff) to show the parts where excepted do not match the actual data. `-` prefix represents the expected data content, `+` prefix represents the actual data content.

We have done a lot of extension functions for verification functions on the original Go Template.

#### Extension functions

Extension functions are used to help users quickly locate the problem content and write test cases that are easier to use.

##### Basic Matches

Verify that the number fits the range.

|Function|Description|Grammar|Verify success|Verify failure|
|-------|------------|-------|-------------|-------------|
|gt|Verify the first param is greater than second param |{{gt param1 param2}}|param1|<wanted gt $param2, but was $param1>|
|ge|Verify the first param is greater than or equals second param |{{ge param1 param2}}|param1|<wanted gt $param2, but was $param1>|
|lt|Verify the first param is less than second param |{{lt param1 param2}}|param1|<wanted gt $param2, but was $param1>|
|le|Verify the first param is less than or equals second param |{{le param1 param2}}|param1|<wanted gt $param2, but was $param1>|
|regexp|Verify the first param matches the second regular expression|{{regexp param1 param2}}|param1|<"$param1" does not match the pattern $param2">|
|notEmpty|Verify The param is not empty|{{notEmpty param}}|param|<"" is empty, wanted is not empty>|
|hasPrefix|Verify The string param has the same prefix.|{{hasPrefix param1 param2}}|true|false|
|hasSuffix|Verify The string param has the same suffix.|{{hasSuffix param1 param2}}|true|false|

##### List Matches

Verify the data in the condition list, Currently, it is only supported when all the conditions in the list are executed, it is considered as successful.

Here is an example, It's means the list values must have value is greater than 0, also have value greater than 1, Otherwise verify is failure.
```yaml
{{- contains .list }}
- key: {{ gt .value 0 }}
- key: {{ gt .value 1 }}
        {{- end }}
```

##### Encoding

In order to make the program easier for users to read and use, some code conversions are provided.

|Function|Description|Grammar|Result|
|-------|------------|-------|------|
|b64enc|Base64 encode|{{ b64enc "Foo" }}|Zm9v|
|sha256enc|Sha256 encode|{{ sha256enc "Foo" }}|1cbec737f863e4922cee63cc2ebbfaafcd1cff8b790d8cfd2e6a5d550b648afa|
|sha512enc|Sha512 encode|{{ sha512enc "Foo" }}|4abcd2639957cb23e33f63d70659b602a5923fafcfd2768ef79b0badea637e5c837161aa101a557a1d4deacbd912189e2bb11bf3c0c0c70ef7797217da7e8207|

### Reuse cases

You could include multiple cases into one single E2E verify, It's helpful for reusing the same verify cases.

Here is the reused verify cases, and using `includes` configuration item to include this into E2E config.

```yaml
cases:
   - actual: path/to/actual.yaml       # verify by actual file path
     expected: path/to/expected.yaml   # excepted content file path
   - query: echo 'foo'                 # verify by command execute output
     expected: path/to/expected.yaml   # excepted content file path
```

## Cleanup

After the E2E finished, how to clean up the environment.

```yaml
cleanup:
   on: always     # Clean up strategy
```

If the `on` option under `cleanup` is not set, it will be automatically set to `always` if there is environment
variable `CI=true`, which is present on many popular CI services, such as GitHub Actions, CircleCI, etc., otherwise it
will be set to `success`, so the testing environment can be preserved when tests failed in your local machine.

All available strategies:
1. `always`: No matter the execution result is success or failure, cleanup will be performed.
1. `success`: Only when the execution succeeds.
1. `failure`: Only when the execution failed.
1. `never`: Never clean up the environment.

