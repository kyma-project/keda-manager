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
	if s.instance.Spec.Logging.Operator == nil {
		return switchState(sFnUpdateMetricsServerDeployment)
	}
	if err := r.updateOperatorLogging(*s.instance.Spec.Logging.Operator); err != nil {
		return stopWithError(err)
	}
	return switchState(sFnUpdateMetricsServerDeployment)
}

func sFnUpdateMetricsServerDeployment(_ context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if s.instance.Spec.Logging.MetricsServer == nil {
		return switchState(sFnApply)
	}
	if err := r.updateMatricsServerLogging(*s.instance.Spec.Logging.MetricsServer); err != nil {
		return stopWithError(err)
	}
	return switchState(sFnApply)
}
