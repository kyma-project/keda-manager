package reconciler

import (
	"context"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func buildSfnDeleteOrphanResources(next stateFn) stateFn {
	return func(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
		err := deleteResources(ctx, r, s.orphanedObjs, nil)
		if err != nil {
			s.instance.UpdateStateFromErr(
				v1alpha1.ConditionTypeInstalled,
				v1alpha1.ConditionReasonOrphanDeletionErr,
				err,
			)
			return stopWithErrorAndNoRequeue(err)
		}
		return switchState(next)
	}
}
