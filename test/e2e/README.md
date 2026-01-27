# Use E2E to test itself
## Introduction
After updating the features of e2e, you can use the files in the test/e2e/ directory to perform testing for both new and old features of e2e.
You can perform testing locally. And when you submit a pull request (PR), GitHub Actions will automatically run the tests.
### Files Structure
```
|- test
  |- e2e
    |- concurrency
      |- fail-fast (concurrency & fail-fast mode)
        |- internal
          |- expected.yaml
          |- verify.yaml (configuration file)
        |- expected.yaml
      |- non-fail-fast (concurrency & non-fail-fast mode)
        |- internal
          |- expected.yaml
          |- verify.yaml (configuration file)
        |- expected.yaml
    |- non-concurrency
      |- fail-fast (non-concurrency & fail-fast mode)
        |- internal
          |- expected.yaml
          |- verify.yaml (configuration file )
        |- expected.yaml
      |- non-fail-fast (non-concurrency & non-fail-fast mode)
        |- internal
          |- expected.yaml
          |- verify.yaml (configuration file)
        |- expected.yaml
    |- kind (KinD import-images test)
      |- kind-cluster.yaml (KinD cluster configuration)
      |- e2e.yaml (E2E test configuration with import-images)
      |- deployment.yaml (test Pod definition)
      |- expected.yaml (expected result)
  |- docker-compose.yaml (run a httpbin container, which can return YAML data)
  |- e2e.yaml (configuration file for the outer infra E2E)

```
### How it works
#### USe E2E to test itself
We use the E2E(released version) to test E2E(dev version) of each mode. The external E2E(released version) will compare the summary of the internal E2E(dev version) of each mode with the expected file. If the summary is as expected, the test of the E2E(dev version) of that mode is passed.
#### Basic flow of SkyWalking Infra E2E
1. Set up a environment for testing
2. Spin up the system under test (SUT), prepare necessary dependencies
3. Trigger or give inputs to SUT, get outputs from SUT
4. Compare the actual outputs and the expected outputs
#### Use 'httpbin' to test E2E(dev verison)
From the view of the external E2E(released version), the internal E2E(dev verison) is the SUT. Also, from the view of each internal E2E(dev verison), the 'httpbin' is the SUT. We use the docker container of 'httpbin' as the SUT, which can receive the 'query' of E2E and return YAML data to E2E.After receiving the YAML data from 'httpbin', the E2E will compare the YAML data with the expected YAML file. At last, the E2E will generate a summary of the result.

## How to add new cases
### 1. add cases in '/internal/verify.yaml' of each mode
```
  cases:
    - name: case-1
      query: 'curl -s 127.0.0.1:8080/get?case=success -H "accept: application/json"'
      expected: ./expected.yaml
    - name: case-2
      query: 'curl -s 127.0.0.1:8080/get?case=success -H "accept: application/json"'
      expected: ./expected.yaml
    - name: case-3
      query: 'curl -s 127.0.0.1:8080/get?case=success -H "accept: application/json"'
      expected: ./expected.yaml
    # Add a new case named 'case-4'
    - name: case-4
      query: 'curl -s 127.0.0.1:8080/get?case=success -H "accept: application/json"'
      expected: ./expected.yaml
```
the 'case-4' will be the passed case, because the parameter is 'success'. In the 'concurrency&fail-fast' mode, the name of the cases should begin with 'passed' or 'failed'.
### 2. add cases in 'expected.yaml' of each mode
- non-concurrency & non-fail-fast mode
```
passed:
  - case-1
  - case-2
  - case-3
  - case-4
  - case-5
  - case-7
failed:
  - case-6
  - case-8
  - case-9
skipped: []
passedCount: 6
failedCount: 3
skippedCount: 0
```
add the name of the cases to 'passed' or 'failed'. And add the number of cases on 'passedCount' and 'failedCount'.
- non-concurrency & fail-fast mode
``` 
passed:
  - case-1
  - case-2
  - case-3
  - case-4
  - case-5
failed:
  - case-6
skipped:
  - case-7
  - case-8
  - case-9
passedCount: 5
failedCount: 1
skippedCount: 3
```
add the name of the cases to 'passed','failed' or 'skipped'. And add the number of cases on 'passedCount','failedCount' and 'skippedCount'.
- concurrency & fail-fast mode
```passed:
{{range .passed}}
- {{ if hasPrefix . "passed" }}{{.}}{{ end }}
{{end}}
failed:
{{range .failed}}
- {{ if hasPrefix . "failed" }}{{.}}{{ end }}
{{end}}
skipped:
{{range .skipped}}
- {{.}}
{{end}}
passedCount: {{le .passedCount x}}
failedCount: {{le .failedCount y}}
skippedCount: {{subtractor z .passedCount .failedCount}} 
```
change the number of cases on 'x' of 'passedCount', 'y' of 'failedCount' and 'z' of 'skippedCount'.
- concurrency & non-fail-fast mode
```passed:
{{- contains .passed}}
- passed-case-1
- passed-case-2
- passed-case-4
- passed-case-5
- passed-case-7
- passed-case-8
{{- end}}
failed:
{{- contains .failed }}
- failed-case-3
- failed-case-6
- failed-case-9
{{- end }}
skipped: []
passedCount: 6
failedCount: 3
skippedCount: 0
```
add the name of the cases to 'passed','failed' or 'skipped'. And add the number of cases on 'passedCount','failedCount' and 'skippedCount'.

## Tips
- You can use `make e2e-test` to run the docker-compose test locally
- You can use `make e2e-test-kind` to run the KinD import-images test locally
- You may need to split your PR to pass e2e tests in CI
```
# /internal/expected.yaml
args:
    case: success
headers:
    Accept: application/json
    Host: 127.0.0.1:8080
    # The 'curl' version in the testing environment may vary.
    User-Agent: curl/7.81.0
# The 'origin' in the testing environment may vary
origin: 172.18.0.1
url: http://127.0.0.1:8080/get?case=success
```
If the above issues prevent the e2e tests from being passed, you can follow these steps to address the problem:

- Remove the '--summary-only -o yaml' from the 'query' in /test/e2e/e2e.yaml file.
- Commit and push the changes to create a new commit.
- Trigger the e2e tests in the CI environment.The detailed information of this test will be displayed in the CI.
- Based on the relevant information, make the necessary modifications to the parameters in /internal/expected.yaml.
- Revert the modifications made to /test/e2e/e2e.yaml.
Commit and push the changes as a new commit.
- The e2e tests should pass in the CI now.
