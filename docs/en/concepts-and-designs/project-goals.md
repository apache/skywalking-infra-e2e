# Design Goals
The document outlines the core design goals for the SkyWalking Infra E2E project.

- Support various E2E testing requirements in SkyWalking main repository with other ecosystem repositories.
- Support both [docker-compose](https://docs.docker.com/compose/) and [KinD](https://kind.sigs.k8s.io/) to orchestrate the tested services 
  under different environments.
- Be language-independent as much as possible, users only need to configure YAMLs and run COmmands, without writing codes.

## Non-Goal

- This framework is not involved with the build process, i.e. it won’t do something like `mvn package` or `docker build`, 
  the artifacts (`.tar`, docker images) should be ready in an earlier process before this;
- This project doesn’t take the plugin tests into account, at least for now;
