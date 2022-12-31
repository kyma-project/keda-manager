package reconciler

import (
	"context"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ stateFn = sFnUpdate

func sFnUpdate(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	// update logging
	// update resources
	// update environmental variables
	return sFnUpdateKedaDeployment, nil, nil
}

func sFnUpdateKedaDeployment(_ context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if err := r.updateOperatorLogging2(*s.instance.Spec.Logging.Operator); err != nil {
		return stopWithError(err)
	}
	return switchState(sFnApply)
}
