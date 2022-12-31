package reconciler

import (
	"context"
	"time"

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
			return stopWithError(err)
		}

		instance := s.instance.DeepCopy()
		instance.Status.State = "Initialized"
		err = r.K8s.Status().Update(ctx, instance)

		// stop state machine with potential error
		return nil, &ctrl.Result{RequeueAfter: time.Second}, err
	}
	// in case instance has no finalizer and instance is being deleted - end reconciliation
	if instanceIsBeingDeleted && !controllerutil.ContainsFinalizer(&s.instance, r.Finalizer) {
		r.log.Debug("instance is being deleted")
		// stop state machine
		return stop()
	}
	// in case instance is being deleted and has finalizer - delete all resources
	if instanceIsBeingDeleted {
		return switchState(sFnDeleteResources)
	}

	return switchState(sFnUpdate)
}
