# SkyWalking Infra E2E

<img src="http://skywalking.apache.org/assets/logo.svg" alt="Sky Walking logo" height="90px" align="right" />

SkyWalking Infra E2E is the next generation End-to-End Testing framework that aims to help developers to set up, debug, and verify E2E tests with ease. It’s built based on the lessons learnt from tens of hundreds of test cases in the SkyWalking main repo.

[![Twitter Follow](https://img.shields.io/twitter/follow/asfskywalking.svg?style=for-the-badge&label=Follow&logo=twitter)](https://twitter.com/AsfSkyWalking)

## Documentation

- **[Official documentation](https://skywalking.apache.org/docs/#SkyWalkingInfraE2E).**

## GitHub Actions

To use skywalking-infra-e2e in GitHub Actions, add a step in your GitHub workflow.

```yaml
- name: Run E2E Test
  uses: apache/skywalking-infra-e2e@main      # always prefer to use a revision instead of `main`.
  with:
    e2e-file: e2e.yaml                        # need to run E2E file path
```

## License

[Apache License 2.0](https://github.com/apache/skywalking-infra-e2e/blob/master/LICENSE)

## Contact Us

* Submit [an issue](https://github.com/apache/skywalking/issues/new) by using [INFRA] as title prefix.
* Mail list: **dev@skywalking.apache.org**. Mail to dev-subscribe@skywalking.apache.org, follow the reply to subscribe the mail list.
* Join `skywalking` channel at [Apache Slack](http://s.apache.org/slack-invite). If the link is not working, find the latest one at [Apache INFRA WIKI](https://cwiki.apache.org/confluence/display/INFRA/Slack+Guest+Invites).
* Twitter, [ASFSkyWalking](https://twitter.com/ASFSkyWalking)
