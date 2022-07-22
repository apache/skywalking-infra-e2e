Changes by Version
==================
Release Notes.

1.2.0
------------------
#### Features
* Expand kind file path with system environment.
* Support shutdown service during setup phase in compose mode.
* Expand kind file path with system environment. 
* Support arbitrary os and arch.
* Support `docker-compose` v2 container naming.
* Support installing via `go install` and add install doc.
* Add retry when delete kind cluster.
* Upgrade to go1.18.

#### Bug Fixes
* Fix the problem of parsing `verify.retry.interval` without setting value.

#### Documentation
* Make `trigger.times` parameter doc more clear.

#### Issues and PR
- All issues are [here](https://github.com/apache/skywalking/milestone/111?closed=1)
- All and pull requests are [here](https://github.com/apache/skywalking-infra-e2e/pulls?q=is%3Apr+is%3Aclosed+milestone%3A1.2.0)
