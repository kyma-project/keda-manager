# Install Keda Manager 

- [Install Keda Manager](#install-keda-manager)
  - [Prerequisites](#prerequisites)
  - [Install Keda Manager from the local sources](#install-keda-manager-from-the-local-sources)
  - [Make targets to run Keda module locally k3d](#make-targets-to-run-keda-module-locally-k3d)
    - [Run Keda module with Lifecycle Manager](#run-keda-module-with-lifecycle-manager)
    - [Run Keda module on bare k3d](#run-keda-module-on-bare-k3d)
  - [Install Keda module on remote Kyma runtime](#install-keda-module-on-remote-kyma-runtime)


Learn how to install Keda Manager locally (on k3d) or on your remote cluster.

## Prerequisites

- Access to a Kubernetes (v1.24 or higher) cluster or [k3d](https://k3d.io/v5.4.6/)
- [Go](https://go.dev/)
- [Docker](https://www.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/)

## Install Keda Manager from the local sources 

You can build and run the Keda Manager in the Kubernetes cluster without Kyma.
For the day-to-day development on your machine, you don't always need to have it controlled by Kyma's Lifecycle Manager.

Run the following commands to deploy Keda Manager on a target Kubernetes cluster (for example, on k3d):

1. Clone the project.

   ```bash
   git clone https://github.com/kyma-project/keda-manager.git && cd keda-manager/
   ```

2. Set the Keda Manager image name.

   > NOTE: You can use local k3d registry or your Docker Hub account to push intermediate images.  
   ```bash
   export IMG=<DOCKER_USERNAME>/custom-keda-manager:0.0.2
   ```

3. Verify the compatibility.

   ```bash
   make test
   ```
4. Build and push the image to the registry.

   ```bash
   make module-image
   ```
5. Create a target Namespace.

   ```bash
   kubectl create ns kyma-system
   ```

6. Deploy Keda Manager.

   ```bash
   make deploy
   ```

7. Verify if Keda Manager is deployed.

   ```bash
   kubectl get deployments -n kyma-system
   ```

   You should get a result similar to this example:

   ```
   NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
   keda-manager            1/1     1            1           1m
   ```

## Make targets to run Keda module locally k3d

### Run Keda module with Lifecycle Manager

Use the dedicated `make` target (in the `hack` folder).

   ```bash
   make -C hack/local run-with-lifecycle-manager
   ```
   
### Run Keda module on bare k3d

When using a local k3d cluster, you can also use the local OCI image registry that comes with it.
Thanks to that, you don't need to push the Keda module images to a remote registry and you can test the changes in the Kyma installation set up entirely on your machine.

1. Clone the project.

   ```bash
   git clone https://github.com/kyma-project/keda-manager.git && cd keda-manager/
   ```
2. Build the manager locally and run it on the k3d cluster.

   ```bash
   make -C hack/local run-without-lifecycle-manager
   ```
3. If you want to clean up the k3d cluster, use the `make -C hack/local stop` make target.

## Install Keda module on remote Kyma runtime

Prerequisite: Lifecycle Manager must be installed on the cluster (locally), or the cluster itself must be managed remotely by the central control-plane.

In this section, you will learn how to install a pull request (PR) version of the Keda module with Lifecycle Manager on a remote cluster.
You need OCI images for the Keda module version to be built and pushed into a public registry. You also need ModuleTemplate matching the version, to apply it on the remote cluster.
CI jobs running on PRs and on main branch help you to achieve that.

1. Create a PR or use an existing one in the [`keda-manager`](https://github.com/kyma-project/keda-manager) repository; on the PR page, scroll down to the Prow jobs status list. 

   ![Prow job status](/docs/assets/prow_job_status.png)

2. After the job has finished with success, click **Details** next to the `pull-keda-module-build` job.

   ![Pull Keda module build](/docs/assets/pull_keda_module_build.png)

The ModuleTemplate will be printed in the MODULE TEMPLATE section, between the tags.

> `~~~~~~~~~~~~BEGINING OF MODULE TEMPLATE~~~~~~~~~~~~~~`

   ```yaml
   apiVersion: operator.kyma-project.io/v1alpha1
   kind: ModuleTemplate
   metadata:
   name: moduletemplate-keda
   ...
   ```

> `~~~~~~~~~~~~~~~END OF MODULE TEMPLATE~~~~~~~~~~~~~~~~`

<details>
<summary><b>Example of full job build result</b></summary>

   ```text
   make: Entering directory '/home/prow/go/src/github.com/kyma-project/keda-manager/hack/ci'
   make[1]: Entering directory '/home/prow/go/src/github.com/kyma-project/keda-manager'
   mkdir -p /home/prow/go/src/github.com/kyma-project/keda-manager/bin
   ## Detect if operating system 
   test -f /home/prow/go/src/github.com/kyma-project/keda-manager/bin/kyma-unstable || curl -s -Lo /home/prow/go/src/github.com/kyma-project/keda-manager/bin/kyma-unstable https://storage.googleapis.com/kyma-cli-unstable/kyma-linux
   chmod 0100 /home/prow/go/src/github.com/kyma-project/keda-manager/bin/kyma-unstable
   test -s /home/prow/go/src/github.com/kyma-project/keda-manager/bin/kustomize || { curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash -s -- 4.5.6 /home/prow/go/src/github.com/kyma-project/keda-manager/bin; }
   {Version:kustomize/v4.5.6 GitCommit:29ca6935bde25565795e1b4e13ca211c4aa56417 BuildDate:2022-07-29T20:42:23Z GoOs:linux GoArch:amd64}
   kustomize installed to /home/prow/go/src/github.com/kyma-project/keda-manager/bin/kustomize
   cd config/manager && /home/prow/go/src/github.com/kyma-project/keda-manager/bin/kustomize edit set image controller=europe-docker.pkg.dev/kyma-project/dev/keda-manager:PR-101
   [0;33;1mWARNING: This command is experimental and might change in its final version. Use at your own risk.
   [0m- Kustomize ready
   - Module built
   - Default CR validation succeeded
   - Creating module archive at "./mod"
   - Image created
   - Pushing image to "europe-docker.pkg.dev/kyma-project/dev/unsigned"
   - Generating module template
   make[1]: Leaving directory '/home/prow/go/src/github.com/kyma-project/keda-manager'

   ~~~~~~~~~~~~BEGINING OF MODULE TEMPLATE~~~~~~~~~~~~~~
   apiVersion: operator.kyma-project.io/v1alpha1
   kind: ModuleTemplate
   metadata:
   name: moduletemplate-keda
   namespace: kcp-system
   labels:
	   "operator.kyma-project.io/managed-by": "lifecycle-manager"
	   "operator.kyma-project.io/controller-name": "manifest"
	   "operator.kyma-project.io/module-name": "keda"
   annotations:
	   "operator.kyma-project.io/module-version": "0.0.2-PR-101"
	   "operator.kyma-project.io/module-provider": "internal"
	   "operator.kyma-project.io/descriptor-schema-version": "v2"
   spec:
   target: remote
   channel: fast
   data:
	   apiVersion: operator.kyma-project.io/v1alpha1
	   kind: Keda
	   metadata:
		   name: default
	   spec:
		   logging:
		   operator:
			   level: "debug"
		   resources:
		   operator:
			   limits:
				   cpu: "1"
				   memory: "200Mi"
			   requests:
				   cpu: "0.5"
				   memory: "150Mi"
		   metricServer:
			   limits:
				   cpu: "1"
				   memory: "1000Mi"
			   requests:
				   cpu: "300m"
				   memory: "500Mi"
   descriptor:
	   component:
		   componentReferences: []
		   name: kyma-project.io/module/keda
		   provider: internal
		   repositoryContexts:
		   - baseUrl: europe-docker.pkg.dev/kyma-project/dev/unsigned
		   componentNameMapping: urlPath
		   type: ociRegistry
		   resources:
		   - access:
			   digest: sha256:3bf7c3bc2d666165ae2ae6cbcad2e3fcaa3a66ca3afebda8c9d008ab93413453
			   type: localOciBlob
		   name: keda
		   relation: local
		   type: helm-chart
		   version: 0.0.2-PR-101
		   - access:
			   digest: sha256:f4a599c4310b0fe9133b67b72d9b15ee96b52a1872132528c83978239b5effef
			   type: localOciBlob
		   name: config
		   relation: local
		   type: yaml
		   version: 0.0.2-PR-101
		   sources:
		   - access:
			   commit: f3b1b7ed6c175e89a7d29202b8a4cc4fc74cf998
			   ref: refs/heads/main
			   repoUrl: github.com/kyma-project/keda-manager
			   type: github
		   name: keda-manager
		   type: git
		   version: 0.0.2-PR-101
		   version: 0.0.2-PR-101
	   meta:
		   schemaVersion: v2

   ~~~~~~~~~~~~~~~END OF MODULE TEMPLATE~~~~~~~~~~~~~~~~
   make: Leaving directory '/home/prow/go/src/github.com/kyma-project/keda-manager/hack/ci'
   ```
</details>

3. Save the section's content in the local file.

4. Apply ModuleTemplate on your remote cluster:

   ```bash
   kubectl apply -f <saved_module_template_path>
   ```

5. Enable the Keda Manager module by patching the Kyma CRD.

   ```bash
   make -C hack/common module
   ```