package reconciler

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

func stopWithError(err error) (stateFn, *ctrl.Result, error) {
	return nil, nil, err
}

func stop() (stateFn, *ctrl.Result, error) {
	return nil, nil, nil
}

func stopWithResult(result ctrl.Result) (stateFn, *ctrl.Result, error) {
	return nil, &result, nil
}

func switchState(fn stateFn) (stateFn, *ctrl.Result, error) {
	return fn, nil, nil
}
