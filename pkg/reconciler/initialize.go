package reconciler

import (
	"context"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func sFnInitialize(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	instanceIsBeingDeleted := !s.instance.GetDeletionTimestamp().IsZero()
	instanceHasFinalizer := controllerutil.ContainsFinalizer(&s.instance, r.Finalizer)

	// in case instance does not have finalizer - add it and update instance
	if !instanceIsBeingDeleted && !instanceHasFinalizer {
		r.log.Debug("adding finalizer")
		controllerutil.AddFinalizer(&s.instance, r.Finalizer)

		err := r.Update(ctx, &s.instance)
		if err != nil {
			return stopWithErrorAndNoRequeue(err)
		}

		s.instance.UpdateStateProcessing(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonInitialized,
			"initialized",
		)
		return stopWithRequeue()
	}
	// in case instance has no finalizer and instance is being deleted - end reconciliation
	if instanceIsBeingDeleted && !controllerutil.ContainsFinalizer(&s.instance, r.Finalizer) {
		r.log.Debug("instance is being deleted")
		// stop state machine
		return nil, nil, nil
	}
	// in case instance is being deleted and has finalizer - delete all resources
	if instanceIsBeingDeleted {
		return switchState(sFnDeleteResources)
	}
	return switchState(sFnUpdateKedaDeployment)
}
