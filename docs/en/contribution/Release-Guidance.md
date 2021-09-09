# Apache SkyWalking Infra E2E Release Guide

This documentation guides the release manager to release the SkyWalking Infra E2E in the Apache Way, and also helps people to check the release for voting.

## Prerequisites

1. Close (if finished, or move to next milestone otherwise) all issues in the current milestone from [skywalking-infra-e2e](https://github.com/apache/skywalking-infra-e2e/milestones) and [skywalking](https://github.com/apache/skywalking/milestones), create a new milestone if needed.
1. Update [CHANGES.md](../../../CHANGES.md).

## Add your GPG public key to Apache svn

1. Upload your GPG public key to a public GPG site, such as [MIT's site](http://pgp.mit.edu:11371/).

1. Log in [id.apache.org](https://id.apache.org/) and submit your key fingerprint.

1. Add your GPG public key into [SkyWalking GPG KEYS](https://dist.apache.org/repos/dist/release/skywalking/KEYS) file, **you can do this only if you are a PMC member**.  You can ask a PMC member for help. **DO NOT override the existed `KEYS` file content, only append your key at the end of the file.**


## Build and sign the source code package

```shell
export VERSION=<the version to release>

git clone --recurse-submodules git@github.com:apache/skywalking-infra-e2e.git && cd skywalking-infra-e2e
git tag -a "v$VERSION" -m "Release Apache SkyWalking-Infra-E2E $VERSION"
git push --tags

make clean 

make test # this is optional, it runs sanity checks to verify the features

make release
```

## Upload to Apache svn

```bash
svn co https://dist.apache.org/repos/dist/dev/skywalking/infra-e2e release/skywalking/infra-e2e
mkdir -p release/skywalking/infra-e2e/"$VERSION"
cp skywalking-infra-e2e/skywalking*.tgz release/skywalking/infra-e2e/"$VERSION"
cp skywalking-infra-e2e/skywalking*.tgz.asc release/skywalking/infra-e2e/"$VERSION"
cp skywalking-infra-e2e/skywalking*.tgz.sha512 release/skywalking/infra-e2e/"$VERSION"

cd release/skywalking && svn add infra-e2e/$VERSION && svn commit infra-e2e -m "Draft Apache SkyWalking-Infra-E2E release $VERSION"
```

## Call for vote in dev@ mailing list

Call for vote in `dev@skywalking.apache.org`.

```text
Subject: [VOTE] Release Apache SkyWalking Infra E2E version $VERSION

Content:

Hi the SkyWalking Community:
This is a call for vote to release Apache SkyWalking Infra E2E version $VERSION.

Release notes:

 * https://github.com/apache/skywalking-infra-e2e/blob/v$VERSION/CHANGELOG.md

Release Candidate:

 * https://dist.apache.org/repos/dist/dev/skywalking/infra-e2e/$VERSION
 * sha512 checksums
   - sha512xxxxyyyzzz skywalking-e2e-$VERSION-bin.tgz
   - sha512xxxxyyyzzz skywalking-e2e-$VERSION-src.tgz

Release Tag :

 * (Git Tag) v$VERSION

Release Commit Hash :

 * https://github.com/apache/skywalking-infra-e2e/tree/<Git Commit Hash>

Keys to verify the Release Candidate :

 * https://dist.apache.org/repos/dist/release/skywalking/KEYS

Guide to build the release from source :

 * https://github.com/apache/skywalking-infra-e2e/blob/main/docs/en/contribution/Compiling-Guidance.md

Voting will start now and will remain open for at least 72 hours, all PMC members are required to give their votes.

[ ] +1 Release this package.
[ ] +0 No opinion.
[ ] -1 Do not release this package because....

Thanks.

[1] https://github.com/apache/skywalking/blob/master/docs/en/guides/How-to-release.md#vote-check
```

## Vote Check

All PMC members and committers should check these before voting +1:

1. Features test.
1. All artifacts in staging repository are published with `.asc`, and `sha` files.
1. Source codes and distribution packages (`skywalking-e2e-$VERSION-src.tgz`)
   are in `https://dist.apache.org/repos/dist/dev/skywalking/infra-e2e/$VERSION` with `.asc`, `.sha512`.
1. `LICENSE` and `NOTICE` are in source codes and distribution package.
1. Check `shasum -c skywalking-e2e-$VERSION-src.tgz.sha512`.
1. Check `gpg --verify skywalking-e2e-$VERSION-src.tgz.asc skywalking-e2e-$VERSION-src.tgz`.
1. Build distribution from source code package by following this [the build guide](#build-and-sign-the-source-code-package).

Vote result should follow these:

1. PMC vote is +1 binding, all others is +1 no binding.

1. Within 72 hours, you get at least 3 (+1 binding), and have more +1 than -1. Vote pass.

1. **Send the closing vote mail to announce the result**. When count the binding and no binding votes, please list the names of voters. An example like this:

   ```
   [RESULT][VOTE] Release Apache SkyWalking Infra E2E version $VERSION
   
   72+ hours passed, we’ve got ($NUMBER) +1 bindings (and ... +1 non-bindings):
   
   (list names)
   +1 bindings:
   xxx
   ...
   
   +1 non-bindings:
   xxx
   ...
    
   Thank you for voting, I’ll continue the release process.
   ```

## Publish release

1. Move source codes tar balls and distributions to `https://dist.apache.org/repos/dist/release/skywalking/`, **you can do this only if you are a PMC member**.

    ```shell
    svn mv https://dist.apache.org/repos/dist/dev/skywalking/infra-e2e/"$VERSION" https://dist.apache.org/repos/dist/release/skywalking/infra-e2e/"$VERSION"
    ```

1. Refer to the previous [PR](https://github.com/apache/skywalking-website/pull/132), update news and links on the website. There are several files need to modify.

1. Update [Github release page](https://github.com/apache/skywalking-infra-e2e/releases), follow the previous convention.

1. Send ANNOUNCE email to `dev@skywalking.apache.org` and `announce@apache.org`, the sender should use his/her Apache email account.

    ```
    Subject: [ANNOUNCEMENT] Apache SkyWalking Infra E2E $VERSION Released

    Content:

    Hi the SkyWalking Community

    On behalf of the SkyWalking Team, I’m glad to announce that SkyWalking Infra E2E $VERSION is now released.

    SkyWalking Infra E2E: An End-to-End Testing framework that aims to help developers to set up, debug, and verify E2E tests with ease.

    SkyWalking: APM (application performance monitor) tool for distributed systems, especially designed for microservices, cloud native and container-based (Docker, Kubernetes, Mesos) architectures.

    Download Links: http://skywalking.apache.org/downloads/

    Release Notes : https://github.com/apache/skywalking-infra-e2e/blob/v$VERSION/CHANGELOG.md

    Website: http://skywalking.apache.org/

    SkyWalking Infra E2E Resources:
    - Issue: https://github.com/apache/skywalking/issues
    - Mailing list: dev@skywalking.apache.org
    - Documents: https://github.com/apache/skywalking-infra-e2e/blob/v$VERSION/README.md
    
    The Apache SkyWalking Team
```
