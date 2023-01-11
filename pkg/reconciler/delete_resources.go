package reconciler

import (
	"context"

	"github.com/go-errors/errors"
	"github.com/kyma-project/keda-manager/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DeletionErr = errors.New("deletion error")
)

func sFnDeleteResources(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	var err error
	for _, obj := range r.Objs {
		r.log.
			With("objName", obj.GetName()).
			With("gvk", obj.GroupVersionKind()).
			Debug("deleting")

		err = r.Delete(ctx, &obj)
		err = client.IgnoreNotFound(err)

		if err != nil {
			r.log.With("deleting resource").Error(err)
		}
	}

	if err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeletionErr,
			DeletionErr,
		)
		// stop state machine with an error and requeue reconciliation in 1min
		return stopWithErrorAnNoRequeue(err)
	}
	return switchState(sFnRemoveFinalizer)
}
