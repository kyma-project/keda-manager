package reconciler

import (
	"context"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/kyma-project/keda-manager/pkg/reconciler/networkpolicy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnUpdateKedaDeployment(_ context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	u, err := r.kedaOperatorDeployment()
	if err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeploymentUpdateErr,
			err,
		)
		return stopWithErrorAndNoRequeue(err)
	}
	next := buildSfnUpdateOperatorLogging(u)
	return switchState(next)
}

func loggingOperatorCfg(k *v1alpha1.Keda) *v1alpha1.LoggingOperatorCfg {
	if k != nil && k.Spec.Logging != nil {
		return k.Spec.Logging.Operator
	}
	return nil
}

func istioOperatorCfg(k *v1alpha1.Keda) *v1alpha1.IstioCfg {
	if k != nil && k.Spec.Istio != nil && k.Spec.Istio.Operator != nil {
		return k.Spec.Istio.Operator
	}

	return disabledIstioSidecar(k)
}

func podAnnotationsOperatorCfg(k *v1alpha1.Keda) *map[string]string {
	if k != nil && k.Spec.PodAnnotations != nil {
		return &k.Spec.PodAnnotations.Operator
	}
	return nil
}

func podAnnotationsMetricsServerCfg(k *v1alpha1.Keda) *map[string]string {
	if k != nil && k.Spec.PodAnnotations != nil {
		return &k.Spec.PodAnnotations.MetricsServer
	}
	return nil
}

func podAnnotationsAdmissionWebhookCfg(k *v1alpha1.Keda) *map[string]string {
	if k != nil && k.Spec.PodAnnotations != nil {
		return &k.Spec.PodAnnotations.AdmissionWebhook
	}
	return nil
}

// buildSfnUpdateOperatorLogging - builds state function to update operator's logging properties
func buildSfnUpdateOperatorLogging(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateOperatorLabels(u)
	return buildSfnUpdateObject(u, updateKedaOperatorContainer0Args, loggingOperatorCfg, next)
}

func buildSfnUpdateOperatorLabels(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateOperatorAnnotations(u)
	return buildSfnUpdateObject(u, updateDeploymentLabels, istioOperatorCfg, next)
}

func buildSfnUpdateOperatorAnnotations(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateOperatorPriorityClass(u)
	return buildSfnUpdateObject(u, updateDeploymentAnnotations, podAnnotationsOperatorCfg, next)
}

func buildSfnUpdateOperatorPriorityClass(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateOperatorResources(u)
	return buildSfnUpdateObject(u, updateDeploymentPriorityClass, priorityClassName, next)
}

func buildSfnUpdateOperatorResources(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateOperatorEnvs(u)
	return buildSfnUpdateObject(u, updateKedaContanier0Resources, operatorResources, next)
}

func buildSfnUpdateOperatorEnvs(u *unstructured.Unstructured) stateFn {
	next := buildSfnAppendOperatorNetworkPolicy(u)
	return buildSfnUpdateObject(u, updateKedaContanierEnvs, envVars, next)
}

func buildSfnAppendOperatorNetworkPolicy(u *unstructured.Unstructured) stateFn {
	return buildSfnAddNetworkPolicy(u, sFnUpdateMetricsServerDeployment)
}

func sFnUpdateMetricsServerDeployment(_ context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	u, err := r.kedaMetricsServerDeployment()
	if err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeploymentUpdateErr,
			err,
		)
		return stopWithErrorAndNoRequeue(err)
	}
	next := buildSfnUpdateMetricsSvrLogging(u)
	return switchState(next)
}

func buildSfnUpdateMetricsSvrLogging(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateMetricsSvrLabels(u)
	return buildSfnUpdateObject(u, updateKedaMetricsServerContainer0Args, loggingMetricsSrvCfg, next)
}

func buildSfnUpdateMetricsSvrLabels(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateMetricsSvrAnnotations(u)
	return buildSfnUpdateObject(u, updateDeploymentLabels, istioMetricServerCfg, next)
}

func buildSfnUpdateMetricsSvrAnnotations(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateMetricsSvrPriorityClass(u)
	return buildSfnUpdateObject(u, updateDeploymentAnnotations, podAnnotationsMetricsServerCfg, next)
}

func buildSfnUpdateMetricsSvrPriorityClass(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateMetricsSvrResources(u)
	return buildSfnUpdateObject(u, updateDeploymentPriorityClass, priorityClassName, next)
}

func buildSfnUpdateMetricsSvrResources(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateMetricsSvrEnvVars(u)
	return buildSfnUpdateObject(u, updateKedaContanier0Resources, metricsSvrResources, next)
}

func buildSfnUpdateMetricsSvrEnvVars(u *unstructured.Unstructured) stateFn {
	next := buildSfnAppendMetricsSvrNetworkPolicy(u)
	return buildSfnUpdateObject(u, updateKedaContanierEnvs, envVars, next)
}

func buildSfnAppendMetricsSvrNetworkPolicy(u *unstructured.Unstructured) stateFn {
	return buildSfnAddNetworkPolicy(u, sFnUpdateAdmissionWebhooksDeployment)
}

func sFnUpdateAdmissionWebhooksDeployment(_ context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	u, err := r.kedaAdmissionWebhooksDeployment()
	if err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeploymentUpdateErr,
			err,
		)
		return stopWithErrorAndNoRequeue(err)
	}
	next := buildSfnUpdateAdmissionWebhooksLabels(u)
	return switchState(next)
}

func buildSfnUpdateAdmissionWebhooksLabels(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateAdmissionWebhooksAnnotations(u)
	return buildSfnUpdateObject(u, updateDeploymentLabels, disabledIstioSidecar, next)
}

func buildSfnUpdateAdmissionWebhooksAnnotations(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateAdmissionWebhooksResources(u)
	return buildSfnUpdateObject(u, updateDeploymentAnnotations, podAnnotationsAdmissionWebhookCfg, next)
}

func buildSfnUpdateAdmissionWebhooksResources(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateAdmissionWebhooksPriorityClass(u)
	return buildSfnUpdateObject(u, updateKedaContanier0Resources, admissionWebhookResources, next)
}

func buildSfnUpdateAdmissionWebhooksPriorityClass(u *unstructured.Unstructured) stateFn {
	next := buildSfnAppendAdmissionWebhooksNetworkPolicy(u)
	return buildSfnUpdateObject(u, updateDeploymentPriorityClass, priorityClassName, next)
}

func buildSfnAppendAdmissionWebhooksNetworkPolicy(u *unstructured.Unstructured) stateFn {
	return buildSfnAddNetworkPolicy(u, sFnApply)
}

func buildSfnUpdateObject[T any, R any](u *unstructured.Unstructured, update func(T, R) error, getData func(*v1alpha1.Keda) *R, next stateFn) stateFn {
	return func(_ context.Context, _ *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
		data := getData(&s.instance)
		if data == nil {
			return switchState(next)
		}
		if err := updateObj(u, *data, update); err != nil {
			s.instance.UpdateStateFromErr(
				v1alpha1.ConditionTypeInstalled,
				v1alpha1.ConditionReasonDeploymentUpdateErr,
				err,
			)
			return stopWithErrorAndNoRequeue(err)
		}
		return switchState(next)
	}
}

func buildSfnAddNetworkPolicy(u *unstructured.Unstructured, next stateFn) stateFn {
	return func(_ context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
		if s.instance.Spec.NetworkPoliciesEnabled == nil || !*s.instance.Spec.NetworkPoliciesEnabled {
			// skip if network policies are disabled
			return switchState(next)
		}

		var deploy appsv1.Deployment
		if err := fromUnstructured(u.Object, &deploy); err != nil {
			s.instance.UpdateStateFromErr(
				v1alpha1.ConditionTypeInstalled,
				v1alpha1.ConditionReasonNetworkPolicyComposeErr,
				err,
			)
			return stopWithErrorAndNoRequeue(err)
		}

		networkPolicy := networkpolicy.New(deploy.GetName(), deploy.GetNamespace(), deploy.Spec.Selector.MatchLabels)

		networkPolicyObj, err := toUnstructed(networkPolicy)
		if err != nil {
			s.instance.UpdateStateFromErr(
				v1alpha1.ConditionTypeInstalled,
				v1alpha1.ConditionReasonNetworkPolicyComposeErr,
				err,
			)
			return stopWithErrorAndNoRequeue(err)
		}

		s.objs = append(s.objs, unstructured.Unstructured{Object: networkPolicyObj})
		return switchState(next)
	}
}

func loggingMetricsSrvCfg(k *v1alpha1.Keda) *v1alpha1.LoggingMetricsSrvCfg {
	if k != nil && k.Spec.Logging != nil {
		return k.Spec.Logging.MetricsServer
	}
	return nil
}

func istioMetricServerCfg(k *v1alpha1.Keda) *v1alpha1.IstioCfg {
	if k != nil && k.Spec.Istio != nil && k.Spec.Istio.MetricServer != nil {
		return k.Spec.Istio.MetricServer
	}

	return disabledIstioSidecar(k)

}

func operatorResources(k *v1alpha1.Keda) *corev1.ResourceRequirements {
	if k != nil && k.Spec.Resources != nil {
		return k.Spec.Resources.Operator
	}
	return nil
}

func metricsSvrResources(k *v1alpha1.Keda) *corev1.ResourceRequirements {
	if k != nil && k.Spec.Resources != nil {
		return k.Spec.Resources.MetricsServer
	}
	return nil
}

func envVars(k *v1alpha1.Keda) *v1alpha1.EnvVars {
	if k != nil {
		return &k.Spec.Env
	}
	return nil
}

func disabledIstioSidecar(_ *v1alpha1.Keda) *v1alpha1.IstioCfg {
	return &v1alpha1.IstioCfg{
		EnabledSidecarInjection: false,
	}
}

func priorityClassName(_ *v1alpha1.Keda) *string {
	priorityClassName := "keda-priority-class"
	return &priorityClassName
}

func admissionWebhookResources(k *v1alpha1.Keda) *corev1.ResourceRequirements {
	if k != nil && k.Spec.Resources != nil {
		return k.Spec.Resources.AdmissionWebhook
	}
	return nil
}
