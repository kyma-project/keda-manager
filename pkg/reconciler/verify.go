package reconciler

import (
	"context"
	"time"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnVerify(_ context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	var count int
	var kedaVersion string
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

		if deployment.GetName() == "keda-operator" {
			kedaVersion = deployment.GetLabels()["app.kubernetes.io/version"]
		}

		for _, cond := range deployment.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable && cond.Status == v1.ConditionTrue {
				r.log.Debugf("successfully verified keda deployment %s/%s", obj.GetNamespace(), obj.GetName())
				count++
			}
		}
	}

	if count != 3 {
		r.log.Debugf("%d deployments in ready state found ( 3 are expected ) ", count)
		s.instance.UpdateStateProcessing(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonVerification,
			"verification in progress",
		)
		return stopWithRequeueAfter(time.Second * 10)
	}

	if s.instance.Status.State == "Ready" {
		return nil, nil, nil
	}

	s.instance.Status.KedaVersion = kedaVersion
	s.instance.UpdateStateReady(
		v1alpha1.ConditionTypeInstalled,
		v1alpha1.ConditionReasonVerified,
		"keda-operator and keda-operator-metrics-server ready",
	)
	return stopWithNoRequeue()
}
