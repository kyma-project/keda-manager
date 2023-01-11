package reconciler

import (
	"context"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnUpdateStatus(result *ctrl.Result, err error) stateFn {
	return func(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
		updateErr := m.Status().Update(ctx, &s.instance)
		if updateErr != nil {
			m.log.With("updateErr", updateErr).Warn("unable to update instance status")
			if err == nil {
				err = updateErr
			}
			return nil, nil, err
		}

		next := sFnEmmitEventfunc(nil, result, err)
		return next, nil, nil
	}
}
