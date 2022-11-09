package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rtypes "github.com/kyma-project/module-manager/operator/pkg/types"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Keda controller", func() {
	Context("When creating fresh instance", func() {
		const (
			namespaceName = "kyma-system"
			kedaName      = "test"
			operatorName  = "keda-manager"
		)

		var (
			metricsDeploymentName           = fmt.Sprintf("%s-metrics-apiserver", operatorName)
			kedaDeploymentName              = operatorName
			notDefaultOperatorLogLevel      = v1alpha1.OperatorLogLevelDebug
			notDefaultLogFormat             = v1alpha1.LogFormatJSON
			notDefaultLogTimeEncoding       = v1alpha1.TimeEncodingEpoch
			notDefaultMetricsServerLogLevel = v1alpha1.MetricsServerLogLevelDebug
			kedaSpec                        = v1alpha1.KedaSpec{
				Logging: &v1alpha1.LoggingCfg{
					Operator: &v1alpha1.LoggingOperatorCfg{
						Level:        &notDefaultOperatorLogLevel,
						Format:       &notDefaultLogFormat,
						TimeEncoding: &notDefaultLogTimeEncoding,
					},
					MetricsServer: &v1alpha1.LoggingMetricsSrvCfg{
						Level: &notDefaultMetricsServerLogLevel,
					},
				},
				Resources: &v1alpha1.Resources{
					Operator: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("171m"),
							corev1.ResourceMemory: resource.MustParse("172Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("173m"),
							corev1.ResourceMemory: resource.MustParse("174Mi"),
						},
					},
					MetricsServer: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("175m"),
							corev1.ResourceMemory: resource.MustParse("176Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("177m"),
							corev1.ResourceMemory: resource.MustParse("178Mi"),
						},
					},
				},
				Env: []v1alpha1.NameValue{
					{
						Name:  "some-env-name",
						Value: "some-env-value",
					},
					{
						Name:  "other-env-name",
						Value: "other-env-value",
					},
				},
			}
		)

		It("The status should be Success", func() {
			h := testHelper{
				ctx:           context.Background(),
				namespaceName: namespaceName,
			}
			h.createNamespace()

			// operations like C(R)UD can be tested in separated tests,
			// but we have time-consuming flow and decided do it in one test
			shouldCreateKeda(h, kedaName, kedaDeploymentName, metricsDeploymentName, kedaSpec)

			shouldPropagateKedaCrdSpecProperties(h, kedaDeploymentName, metricsDeploymentName, kedaSpec)

			//TODO: disabled because of bug in operator (https://github.com/kyma-project/module-manager/issues/94)
			//shouldUpdateKeda(h, kedaName, kedaDeploymentName)

			shouldDeleteKeda(h, kedaName)
		})
	})
})

func shouldCreateKeda(h testHelper, kedaName, kedaDeploymentName, metricsDeploymentName string, kedaSpec v1alpha1.KedaSpec) {
	// act
	h.createKeda(kedaName, kedaSpec)

	// we have to update deployment status manually
	h.updateDeploymentStatus(metricsDeploymentName)
	h.updateDeploymentStatus(kedaDeploymentName)

	// assert
	Eventually(h.createGetKedaStateFunc(kedaName)).
		WithPolling(time.Second * 2).
		WithTimeout(time.Second * 20).
		Should(Equal(rtypes.StateReady))
}

func shouldDeleteKeda(h testHelper, kedaName string) {
	// initial assert
	Expect(h.getKedaCount()).To(Equal(1))
	kedaState, err := h.getKedaState(kedaName)
	Expect(err).To(BeNil())
	Expect(kedaState).To(Equal(rtypes.StateReady))

	// act
	var keda v1alpha1.Keda
	Eventually(h.createGetKubernetesObjectFunc(kedaName, &keda)).
		WithPolling(time.Second * 2).
		WithTimeout(time.Second * 10).
		Should(BeTrue())
	Expect(k8sClient.Delete(h.ctx, &keda)).To(Succeed())

	// assert
	Eventually(h.getKedaCount).
		WithPolling(time.Second * 2).
		WithTimeout(time.Second * 10).
		Should(Equal(0))

}

func shouldUpdateKeda(h testHelper, kedaName string, kedaDeploymentName string) {
	// arrange
	var keda v1alpha1.Keda
	Eventually(h.createGetKubernetesObjectFunc(kedaName, &keda)).
		WithPolling(time.Second * 2).
		WithTimeout(time.Second * 10).
		Should(BeTrue())

	newTestEnv := v1alpha1.NameValue{
		Name:  "update-test-env-key",
		Value: "update-test-env-value",
	}
	keda.Spec.Env = append(keda.Spec.Env, newTestEnv)

	// act
	Expect(k8sClient.Update(h.ctx, &keda)).To(Succeed())

	// assert
	var deployment appsv1.Deployment
	Eventually(h.createGetKubernetesObjectFunc(kedaDeploymentName, &deployment)).
		WithPolling(time.Second * 2).
		WithTimeout(time.Second * 10).
		Should(BeTrue())

	expectedEnv := ToEnvVar(newTestEnv)
	Expect(deployment.Spec.Template.Spec.Containers[0].Env).To(ContainElement(expectedEnv))
}

func shouldPropagateKedaCrdSpecProperties(h testHelper, kedaDeploymentName string, metricsDeploymentName string, kedaSpec v1alpha1.KedaSpec) {
	checkKedaCrdSpecPropertyPropagationToKedaDeployment(h, kedaDeploymentName, kedaSpec)
	checkKedaCrdSpecPropertyPropagationToMetricsDeployment(h, metricsDeploymentName, kedaSpec)
}

func checkKedaCrdSpecPropertyPropagationToKedaDeployment(h testHelper, kedaDeploymentName string, kedaSpec v1alpha1.KedaSpec) {
	// act
	var kedaDeployment appsv1.Deployment
	Eventually(h.createGetKubernetesObjectFunc(kedaDeploymentName, &kedaDeployment)).
		WithPolling(time.Second * 2).
		WithTimeout(time.Second * 10).
		Should(BeTrue())

	expectedEnvs := ToEnvVars(kedaSpec.Env)

	// assert
	firstContainer := kedaDeployment.Spec.Template.Spec.Containers[0]
	Expect(firstContainer.Args).
		To(ContainElement(fmt.Sprintf("--zap-log-level=%s", *kedaSpec.Logging.Operator.Level)))
	Expect(firstContainer.Args).
		To(ContainElement(fmt.Sprintf("--zap-encoder=%s", *kedaSpec.Logging.Operator.Format)))
	Expect(firstContainer.Args).
		To(ContainElement(fmt.Sprintf("--zap-time-encoding=%s", *kedaSpec.Logging.Operator.TimeEncoding)))

	Expect(firstContainer.Resources).To(Equal(*kedaSpec.Resources.Operator))

	Expect(firstContainer.Env).To(ContainElements(expectedEnvs))
}

func checkKedaCrdSpecPropertyPropagationToMetricsDeployment(h testHelper, metricsDeploymentName string, kedaSpec v1alpha1.KedaSpec) {
	// act
	var metricsDeployment appsv1.Deployment
	Eventually(h.createGetKubernetesObjectFunc(metricsDeploymentName, &metricsDeployment)).
		WithPolling(time.Second * 2).
		WithTimeout(time.Second * 10).
		Should(BeTrue())

	expectedEnvs := ToEnvVars(kedaSpec.Env)

	// assert
	firstContainer := metricsDeployment.Spec.Template.Spec.Containers[0]
	Expect(firstContainer.Args).
		To(ContainElement(fmt.Sprintf("--v=%s", *kedaSpec.Logging.MetricsServer.Level)))

	Expect(firstContainer.Resources).To(Equal(*kedaSpec.Resources.MetricsServer))

	Expect(firstContainer.Env).To(ContainElements(expectedEnvs))
}

func ToEnvVar(nv v1alpha1.NameValue) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  nv.Name,
		Value: nv.Value,
	}
}

func ToEnvVars(nvs []v1alpha1.NameValue) []corev1.EnvVar {
	var result []corev1.EnvVar
	for _, nv := range nvs {
		result = append(result, ToEnvVar(nv))
	}
	return result
}

type testHelper struct {
	ctx           context.Context
	namespaceName string
}

func (h *testHelper) getKedaCount() int {
	var objectList v1alpha1.KedaList
	Expect(k8sClient.List(h.ctx, &objectList)).To(Succeed())
	return len(objectList.Items)
}

func (h *testHelper) createGetKedaStateFunc(kedaName string) func() (rtypes.State, error) {
	return func() (rtypes.State, error) {
		return h.getKedaState(kedaName)
	}
}

func (h *testHelper) getKedaState(kedaName string) (rtypes.State, error) {
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
	Expect(k8sClient.Status().Update(h.ctx, &deployment)).To(Succeed())

	replicaSetName := h.createReplicaSetForDeployment(deployment)

	var replicaSet appsv1.ReplicaSet
	Eventually(h.createGetKubernetesObjectFunc(replicaSetName, &replicaSet)).
		WithPolling(time.Second * 2).
		WithTimeout(time.Second * 10).
		Should(BeTrue())

	replicaSet.Status.ReadyReplicas = 1
	replicaSet.Status.Replicas = 1
	Expect(k8sClient.Status().Update(h.ctx, &replicaSet)).To(Succeed())

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
	Expect(k8sClient.Create(h.ctx, &replicaSet)).To(Succeed())
	By(fmt.Sprintf("Replica set (for deployment) created: %s", replicaSetName))
	return replicaSetName
}

func (h *testHelper) createKeda(kedaName string, spec v1alpha1.KedaSpec) {
	By(fmt.Sprintf("Creating crd: %s", kedaName))
	keda := v1alpha1.Keda{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kedaName,
			Namespace: h.namespaceName,
			Labels: map[string]string{
				"operator.kyma-project.io/kyma-name": "test",
			},
		},
		Spec: spec,
	}
	Expect(k8sClient.Create(h.ctx, &keda)).To(Succeed())
	By(fmt.Sprintf("Crd created: %s", kedaName))
}

func (h *testHelper) createNamespace() {
	By(fmt.Sprintf("Creating namespace: %s", h.namespaceName))
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.namespaceName,
		},
	}
	Expect(k8sClient.Create(h.ctx, &namespace)).To(Succeed())
	By(fmt.Sprintf("Namespace created: %s", h.namespaceName))
}
