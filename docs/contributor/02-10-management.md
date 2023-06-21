# Use Keda Manager to manage KEDA

Keda Manager reconciles KEDA deployment based on the watched Keda custom resources (CRs):

- Apply Keda CR (sample) to have KEDA installed.

   ```bash
   kubectl apply -f config/samples/operator_v1alpha1_keda_k3d.yaml
   ```

   After a while, you have KEDA installed, and you see its workloads:

   ```bash
   NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
   keda-manager                     1/1     1            1           3m
   keda-operator                    1/1     1            1           3m
   keda-operator-metrics-apiserver  1/1     1            1           3m
   ```

   Now you can use KEDA to scale workloads on the Kubernetes cluster. Check the [demo application](/docs/user/04-20-demo-application.md).

- Remove Keda CR to have KEDA uninstalled.

   ```bash
   kubectl delete -f config/samples/operator_v1alpha1_keda_k3d.yaml
   ```
   This uninstalls all KEDA workloads but leaves Keda Manager.

   > **NOTE:** Keda Manager uses finalizers to uninstall the Keda module from the cluster. It means that Keda Manager blocks the uninstallation process of KEDA until there are user-created CRs (for example, ScaledObjects).

- Update the specification of Keda CR to change the Keda installation

   The [configuration example](/docs/user/01-20-configuration.md) shows how to modify the Keda properties using the `keda.operator.kyma-project.io` CR.


   ```bash
   cat <<EOF | kubectl apply -f -
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
   EOF
   ```