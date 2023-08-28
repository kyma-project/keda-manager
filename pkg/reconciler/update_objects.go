package reconciler

import (
	"context"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
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

// buildSfnUpdateOperatorLogging - builds state function to update operator's logging properties
func buildSfnUpdateOperatorLogging(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateOperatorLabels(u)
	return buildSfnUpdateObject(u, updateKedaOperatorContainer0Args, loggingOperatorCfg, next)
}

func buildSfnUpdateOperatorLabels(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateOperatorResources(u)
	return buildSfnUpdateObject(u, updateKedaOperatorSidecarInjection, sidecarInjectionConfig, next)
}

func buildSfnUpdateOperatorResources(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateOperatorEnvs(u)
	return buildSfnUpdateObject(u, updateKedaContanier0Resources, operatorResources, next)
}

func buildSfnUpdateOperatorEnvs(u *unstructured.Unstructured) stateFn {
	return buildSfnUpdateObject(u, updateKedaContanierEnvs, envVars, sFnUpdateMetricsServerDeployment)
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
	next := buildSfnUpdateMetricsSvrResources(u)
	return buildSfnUpdateObject(u, updateKedaMetricsServerSidecarInjection, sidecarInjectionConfig, next)
}

func buildSfnUpdateMetricsSvrResources(u *unstructured.Unstructured) stateFn {
	next := buildSfnUpdateMetricsSvrEnvVars(u)
	return buildSfnUpdateObject(u, updateKedaContanier0Resources, metricsSvrResources, next)
}

func buildSfnUpdateMetricsSvrEnvVars(u *unstructured.Unstructured) stateFn {
	return buildSfnUpdateObject(u, updateKedaContanierEnvs, envVars, sFnUpdateAdmissionWebhooksDeployment)
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
	return buildSfnUpdateObject(u, updateKedaWebhookSidecarInjection, sidecarInjectionConfig, sFnApply)
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

func loggingMetricsSrvCfg(k *v1alpha1.Keda) *v1alpha1.LoggingMetricsSrvCfg {
	if k != nil && k.Spec.Logging != nil {
		return k.Spec.Logging.MetricsServer
	}
	return nil
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

type sidecarConfig struct {
	inject bool
}

func sidecarInjectionConfig(_ *v1alpha1.Keda) *sidecarConfig {
	return &sidecarConfig{false}
}
