package reconciler

import (
	"context"
	"fmt"
	"github.com/kyma-project/keda-manager/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnBootstrapperValidation(_ context.Context, _ *fsm, s *systemState) (stateFn, *ctrl.Result, error) {

	if hasRestrictedAnnotations(s.instance) {
		err := fmt.Errorf("used restricted annotations in Keda CR %s", s.instance.GetName())
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonValidationErr,
			err,
		)
		return stopWithErrorAndNoRequeue(err)
	}

	return switchState(sFnUpdateKedaDeployment)
}

func hasRestrictedAnnotations(dep v1alpha1.Keda) bool {
	// PodAnnotations is a pointer in the spec; guard against nil to avoid panic
	if dep.Spec.PodAnnotations == nil {
		return false
	}

	anns := dep.Spec.PodAnnotations

	restricted := []string{
		v1alpha1.KymaBootstraperAddImagePullSecretMutation,
		v1alpha1.KymaBootstraperRegistryUrlMutation,
		v1alpha1.KymaBootstrapperSetFipsMode,
	}

	for _, an := range restricted {
		if anns.AdmissionWebhook != nil {
			if _, ok := anns.AdmissionWebhook[an]; ok {
				return true
			}
		}
		if anns.Operator != nil {
			if _, ok := anns.Operator[an]; ok {
				return true
			}
		}
		if anns.MetricsServer != nil {
			if _, ok := anns.MetricsServer[an]; ok {
				return true
			}
		}
	}

	return false
}
