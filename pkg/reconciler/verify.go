package reconciler

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnVerify(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	var count int
	for _, obj := range s.objs {
		if obj.GetKind() != "Deployment" {
			continue
		}

		var deployment appsv1.Deployment
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deployment); err != nil {
			return stopWithError(err)
		}

		for _, cond := range deployment.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable && cond.Status == v1.ConditionTrue {
				count++
			}
		}
	}

	result := ctrl.Result{}

	instance := s.instance.DeepCopy()
	if count == 2 {
		if instance.Status.State == "Ready" {
			return nil, nil, nil
		}

		instance.Status.State = "Ready"
		r.log.Debug("deployment rdy")
	} else {
		r.log.With("status", instance.Status).Debug("current status")
		instance.Status.State = "Processing"
		//condition := cHelper.Installed().True(v1alpha1.ConditionReasonVerification, "verification started")
		//meta.SetStatusCondition(&instance.Status.Conditions, condition)
	}

	r.log.With("rscVersion", instance.ResourceVersion).
		With("generation", instance.Generation).
		Debug("changing status")

	err := r.Status().Update(ctx, instance)
	if err != nil {
		result = ctrl.Result{
			RequeueAfter: time.Second,
		}
	}
	return nil, &result, err
}
