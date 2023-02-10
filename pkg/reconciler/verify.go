package reconciler

import (
	"context"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnVerify(_ context.Context, _ *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	var count int
	for _, obj := range s.objs {
		if !isDeployment(obj) {
			continue
		}

		var deployment appsv1.Deployment
		if err := fromUnstructured(obj.Object, &deployment); err != nil {
			s.instance.UpdateStateFromErr(
				v1alpha1.ConditionTypeInstalled,
				v1alpha1.ConditionReasonVerificationErr,
				err,
			)
			return stopWithErrorAndNoRequeue(err)
		}

		for _, cond := range deployment.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable && cond.Status == v1.ConditionTrue {
				count++
			}
		}
	}

	if count != 2 {
		s.instance.UpdateStateProcessing(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonVerification,
			"verification in progress",
		)
		return stopWithNoRequeue()
	}

	if s.instance.Status.State == "Ready" {
		return nil, nil, nil
	}

	s.instance.UpdateStateReady(
		v1alpha1.ConditionTypeInstalled,
		v1alpha1.ConditionReasonVerified,
		"keda-manager and keda-manager-metrics-server ready",
	)
	return stopWithNoRequeue()
}
