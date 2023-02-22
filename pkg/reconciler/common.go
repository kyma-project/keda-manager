package reconciler

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

func stopWithErrorAndNoRequeue(err error) (stateFn, *ctrl.Result, error) {
	return sFnUpdateStatus(nil, err), nil, nil
}

func stopWithNoRequeue() (stateFn, *ctrl.Result, error) {
	return sFnUpdateStatus(nil, nil), nil, nil
}

func stopWithRequeue() (stateFn, *ctrl.Result, error) {
	return sFnUpdateStatus(&ctrl.Result{Requeue: true}, nil), nil, nil
}

func switchState(fn stateFn) (stateFn, *ctrl.Result, error) {
	return fn, nil, nil
}
