---
stage: Verify
group: Runner
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

# Docker Autoscaler executor (alpha)

> Introduced in GitLab Runner 15.10.0.

The Docker Autoscaler executor is an autoscale-enabled Docker executor that creates instances on-demand to
accommodate the jobs that the runner manager processes.

The autoscaler uses [fleeting](https://gitlab.com/gitlab-org/fleeting/fleeting) plugins. `fleeting` is an abstraction
for a group of autoscaled instances and uses plugins that support different cloud providers (such as GCP, AWS and
Azure). This allows instances to be created on-demand to accommodate the jobs that the runner manager processes.

## Prepare the environment

To prepare your environment for autoscaling, install the AWS fleeting plugin. The fleeting plugin
targets the platform that you want to autoscale on.

The AWS fleeting plugin is in alpha. Support for Google Cloud Platform and Azure fleeting plugins
are proposed in <this-issue>.

To install the AWS plugin:

1. [Download the binary](https://gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws/-/releases) for your host platform.
1. Ensure that the plugin binaries are discoverable through the PATH environment variable.

## Configuration

The Docker Autoscaler executor wraps the [Docker executor](docker.md), which means that all Docker Executor options and
features are supported. To enable the autoscaler, the executor `docker-autoscaler` must be used.

- [Docker Executor configuration](../configuration/advanced-configuration.md#the-runnersdocker-section)
- [Autoscaler configuration](../configuration/advanced-configuration.md#the-runnersautoscaler-section)

## Examples

### 1 job per instance using AWS Autoscaling Group

Prerequisites:

- An AMI with [Docker Engine](https://docs.docker.com/engine/) installed.
- An AWS Autoscaling group. For the scaling policy use "none", as Runner handles the scaling.
- An IAM Policy with the [correct permissions](https://gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws#recommended-iam-policy)

This configuration supports:

- A capacity per instance of 1
- A use count of 1
- An idle scale of 5
- An idle time of 20 minutes
- A maximum instance count of 10

By setting the capacity and use count to both 1, each job is given a secure ephemeral instance that cannot be
affected by other jobs. As soon the job is complete the instance it was executed on is immediately deleted.

With an idle scale of 5, the runner will try to keep 5 whole instances (because the capacity per instance is 1)
available for future demand. These instances will stay for at least 20 minutes.

The runner `concurrent` field is set to 10 (maximum number instances * capacity per instance).

```toml
concurrent = 10

[[runners]]
  name = "instance autoscaler example"
  url = "https://gitlab.com"
  token = "<token>"
  shell = "sh"                                        # use powershell or pwsh for Windows AMIs

  # uncomment for Windows AMIs when the Runner manager is hosted on Linux
  # environment = ["FF_USE_POWERSHELL_PATH_RESOLVER=1"]

  executor = "docker-autoscaler"

  # Docker Executor config
  [runners.docker]
    image = "busybox:latest"

  # Autoscaler config
  [runners.autoscaler]
    plugin = "fleeting-plugin-aws"

    capacity_per_instance = 1
    max_use_count = 1
    max_instances = 10

    [runners.autoscaler.plugin_config] # plugin specific configuration (see plugin documentation)
      name             = "my-docker-asg"               # AWS Autoscaling Group name
      profile          = "default"                     # optional, default is 'default'
      config_file      = "/home/user/.aws/config"      # optional, default is '~/.aws/config'
      credentials_file = "/home/user/.aws/credentials" # optional, default is '~/.aws/credentials'

    [runners.autoscaler.connector_config]
      username          = "ec2-user"
      use_external_addr = true

    [[runners.autoscaler.policy]]
      idle_count = 5
      idle_time = "20m0s"
```
