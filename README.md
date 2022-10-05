> **NOTE:** It is a general template that can be used for a project README.md, example README.md, or any other README.md type in all Kyma repositories in the Kyma organization. Not all the sections are mandatory. Use only those that suit your use case but keep the proposed section order.

# Keda Manager

## Overview (mandatory)

Keda Manager is a module compatible with `lifecycle-manager` that allows to add KEDA Event Driven Autoscaler to Kyma ecosystem.

See also:
- [lifecycle-manager documetation](https://github.com/kyma-project/lifecycle-manager)
- [KEDA documentation](https://keda.sh/docs/2.7/concepts/)

## Prerequisites

> List the requirements to run the project or example.

## Installation

1. Building project

```bash
make build
```

2. Build image

```bash
make docker-build IMG=<image-name>:<image-tag>
```

3. Push image to registry 

- If using globaly available docker registry

```bash
make docker-push IMG=<image-name>:<image-tag>
```

- If using k3d

```bash
k3d image import <image-name>:>image-tag> -c <k3d_context>
```

> Explain the steps to install your project. Create an ordered list for each installation task.
>
> If it is an example README.md, describe how to build, run locally, and deploy the example. Format the example as code blocks and specify the language, highlighting where possible. Explain how you can validate that the example ran successfully. For example, define the expected output or commands to run which check a successful deployment.
>
> Add subsections (H3) for better readability.

