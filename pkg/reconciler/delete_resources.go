package reconciler

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func sFnDeleteResources(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	// TODO add CR cleanup before objs cleanup
	// TODO reconcile - event base (on edit) - watch, label
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
		s.instance.Status.State = "Error"
		if err := r.Status().Update(ctx, &s.instance); err != nil {
			r.log.Error(err)
		}
		return nil, &ctrl.Result{RequeueAfter: time.Second}, err
	}

	return switchState(sFnRemoveFinalizer)
}
