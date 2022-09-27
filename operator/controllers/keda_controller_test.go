package controllers

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	rtypes "github.com/kyma-project/module-manager/operator/pkg/types"

	"github.com/kyma-project/keda-manager/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Keda controller", func() {
	Context("When creating fresh instance", func() {
		const (
			namespaceName      = "keda"
			kedaName           = "test"
			operatorName       = "keda-manager"
			serviceAccountName = "keda-manager"
		)

		var (
			metricsDeploymentName = fmt.Sprintf("%s-metrics-apiserver", operatorName)
			kedaDeploymentName    = operatorName
		)

		It("The status should be Success", func() {
			ctx := context.Background()
			h := testHelper{
				ctx:           ctx,
				namespaceName: namespaceName,
			}

			//TODO: add enum on LogLevel values
			debug := v1alpha1.LogLevel("debug")
			kedaSpec := v1alpha1.KedaSpec{
				//TODO: test other properties
				Logging: &v1alpha1.LoggingCfg{
					Operator: &v1alpha1.LoggingOperatorCfg{
						Level: &debug,
					},
				},
				Env: &map[string]string{
					"blekota": "makota",
					"ala":     "mela",
				},
			}

			h.createNamespace()
			h.createKeda(kedaName, kedaSpec)

			// check if some object started
			var serviceAccount corev1.ServiceAccount
			Eventually(h.createGetKubernetesObjectFunc(serviceAccountName, &serviceAccount)).
				WithPolling(time.Second * 2).
				WithTimeout(time.Second * 10).
				Should(BeTrue())

			labelLen := len(serviceAccount.Labels)
			Expect(labelLen).Should(Equal(7))

			// we have to update deployment status manually
			h.updateDeploymentStatus(metricsDeploymentName)
			h.updateDeploymentStatus(kedaDeploymentName)

			// check if keda started
			Eventually(h.createGetKedaStateFunc(kedaName)).
				WithPolling(time.Second * 2).
				WithTimeout(time.Second * 20).
				Should(Equal(rtypes.StateReady))

			///////////////////////////////////////////////////////
			// check if keda crd specification are properly passed to kubernetes object (e.g. logging)
			var kedaDeployment appsv1.Deployment
			Eventually(h.createGetKubernetesObjectFunc(kedaDeploymentName, &kedaDeployment)).
				WithPolling(time.Second * 2).
				WithTimeout(time.Second * 10).
				Should(BeTrue())

			containerIndex := -1
			for i, container := range kedaDeployment.Spec.Template.Spec.Containers {
				if container.Name == operatorName {
					containerIndex = i
					break
				}
			}
			Expect(containerIndex).Should(Not(Equal(-1)))
			Expect(kedaDeployment.Spec.Template.Spec.Containers[containerIndex].Env).Should(BeEquivalentTo(kedaSpec.Env))
			// TODO: create dedicated matchers for keda

			////////////////////////////////////////
			// update
			//var keda v1alpha1.Keda
			//Eventually(h.createGetKubernetesObjectFunc(kedaName, &keda)).
			//	WithPolling(time.Second * 2).
			//	WithTimeout(time.Second * 10).
			//	Should(BeTrue())
			//
			//envs := map[string]string{}
			//if keda.Spec.Env != nil {
			//	envs = *keda.Spec.Env
			//}
			//const (
			//	envKey = "update-test"
			//	envVal = "makapaka"
			//)
			//envs[envKey] = envVal
			//keda.Spec.Env = &envs
			//
			//Expect(k8sClient.Update(h.ctx, &keda)).Should(Succeed())
			//
			//var deployment appsv1.Deployment
			//Eventually(h.createGetKubernetesObjectFunc(kedaDeploymentName, &deployment)).
			//	WithPolling(time.Second * 2).
			//	WithTimeout(time.Second * 10).
			//	Should(BeTrue())
			//
			//containerIndex := -1
			//for i, container := range deployment.Spec.Template.Spec.Containers {
			//	if container.Name == operatorName {
			//		containerIndex = i
			//		break
			//	}
			//}
			//Expect(containerIndex).Should(Not(Equal(-1)))
			//
			//envIndex := -1
			//for i, env := range deployment.Spec.Template.Spec.Containers[containerIndex].Env {
			//	if env.Name == envKey {
			//		envIndex = i
			//		break
			//	}
			//}
			//Expect(envIndex).Should(Not(Equal(-1)))
			//Expect(deployment.Spec.Template.Spec.Containers[containerIndex].Env[envIndex]).Should(Equal(envVal))

			////////////////////////////////////////////////////////
			// delete
			// TODO: check all kinds of kubernetes objects
			var nsListBefore appsv1.DeploymentList
			Expect(k8sClient.List(h.ctx, &nsListBefore)).Should(Succeed())
			Expect(nsListBefore.Items).Should(Not(BeEmpty()))

			var keda v1alpha1.Keda
			Eventually(h.createGetKubernetesObjectFunc(kedaName, &keda)).
				WithPolling(time.Second * 2).
				WithTimeout(time.Second * 10).
				Should(BeTrue())

			Expect(k8sClient.Delete(h.ctx, &keda)).Should(Succeed())

			// TODO: check all kinds of kubernetes objects
			Eventually(h.createKymaObjectListIsEmptyFunc()).
				WithPolling(time.Second * 2).
				WithTimeout(time.Second * 10).
				Should(BeTrue())
		})
	})
})

type testHelper struct {
	ctx           context.Context
	namespaceName string
}

func (h *testHelper) createKymaObjectListIsEmptyFunc() func() (bool, error) {
	return func() (bool, error) {
		var nsList appsv1.DeploymentList
		err := k8sClient.List(h.ctx, &nsList)
		if err != nil {
			return false, err
		}
		return len(nsList.Items) == 0, nil
	}
}

func (h *testHelper) createGetKedaStateFunc(kedaName string) func() (rtypes.State, error) {
	return func() (rtypes.State, error) {
		var emptyState = rtypes.State("")
		var keda v1alpha1.Keda
		key := types.NamespacedName{
			Name:      kedaName,
			Namespace: h.namespaceName,
		}
		err := k8sClient.Get(h.ctx, key, &keda)
		if err != nil {
			return emptyState, err
		}
		return keda.Status.State, nil
	}
}

func (h *testHelper) createGetKubernetesObjectFunc(serviceAccountName string, obj client.Object) func() (bool, error) {
	return func() (bool, error) {
		key := types.NamespacedName{
			Name:      serviceAccountName,
			Namespace: h.namespaceName,
		}
		err := k8sClient.Get(h.ctx, key, obj)
		if err != nil {
			return false, err
		}
		return true, err
	}
}

func (h *testHelper) updateDeploymentStatus(deploymentName string) {
	By(fmt.Sprintf("Updating deployment status: %s", deploymentName))
	var deployment appsv1.Deployment
	Eventually(h.createGetKubernetesObjectFunc(deploymentName, &deployment)).
		WithPolling(time.Second * 2).
		WithTimeout(time.Second * 10).
		Should(BeTrue())

	deployment.Status.Conditions = append(deployment.Status.Conditions, appsv1.DeploymentCondition{
		Type:    appsv1.DeploymentAvailable,
		Status:  corev1.ConditionTrue,
		Reason:  "test-reason",
		Message: "test-message",
	})
	deployment.Status.Replicas = 1
	Expect(k8sClient.Status().Update(h.ctx, &deployment)).Should(Succeed())

	replicaSetName := h.createReplicaSetForDeployment(deployment)

	var replicaSet appsv1.ReplicaSet
	Eventually(h.createGetKubernetesObjectFunc(replicaSetName, &replicaSet)).
		WithPolling(time.Second * 2).
		WithTimeout(time.Second * 10).
		Should(BeTrue())

	replicaSet.Status.ReadyReplicas = 1
	replicaSet.Status.Replicas = 1
	Expect(k8sClient.Status().Update(h.ctx, &replicaSet)).Should(Succeed())

	By(fmt.Sprintf("Deployment status updated: %s", deploymentName))
}

func (h *testHelper) createReplicaSetForDeployment(deployment appsv1.Deployment) string {
	replicaSetName := fmt.Sprintf("%s-replica-set", deployment.Name)
	By(fmt.Sprintf("Creating replica set (for deployment): %s", replicaSetName))
	var (
		trueValue = true
		one       = int32(1)
	)
	replicaSet := appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      replicaSetName,
			Namespace: h.namespaceName,
			Labels: map[string]string{
				"app": deployment.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       deployment.Name,
					UID:        deployment.GetUID(),
					Controller: &trueValue,
				},
			},
		},
		// dummy values
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &one,
			Selector: deployment.Spec.Selector,
			Template: deployment.Spec.Template,
		},
	}
	Expect(k8sClient.Create(h.ctx, &replicaSet)).Should(Succeed())
	By(fmt.Sprintf("Replica set (for deployment) created: %s", replicaSetName))
	return replicaSetName
}

func (h *testHelper) createKeda(kedaName string, spec v1alpha1.KedaSpec) {
	By(fmt.Sprintf("Creating crd: %s", kedaName))
	keda := v1alpha1.Keda{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kedaName,
			Namespace: h.namespaceName,
		},
		Spec: spec,
	}
	Expect(k8sClient.Create(h.ctx, &keda)).Should(Succeed())
	By(fmt.Sprintf("Crd created: %s", kedaName))
}

func (h *testHelper) createNamespace() {
	By(fmt.Sprintf("Creating namespace: %s", h.namespaceName))
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.namespaceName,
		},
	}
	Expect(k8sClient.Create(h.ctx, &namespace)).Should(Succeed())
	By(fmt.Sprintf("Namespace created: %s", h.namespaceName))
}
