package reconciler

import (
	"context"
	"errors"
	"reflect"
	"time"

	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func sFnApply(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	var isError bool
	for _, obj := range r.Objs {
		r.log.With("gvk", obj.GetObjectKind().GroupVersionKind()).
			With("name", obj.GetName()).
			Debug("applying")

		err := r.Patch(ctx, &obj, client.Apply, &client.PatchOptions{
			Force:        pointer.Bool(true),
			FieldManager: "keda-manager",
		})

		if err != nil {
			r.log.With("err", err).Debug("apply result")
			isError = true
		}

		s.objs = append(s.objs, obj)
	}
	// no errors
	if !isError {
		return switchState(sFnVerify)
	}

	err := errors.New("installation error")
	instance := s.instance.DeepCopy()
	//newCondition := cHelper.Installed().False(v1alpha1.ConditionReasonCrdError, err.Error())
	//meta.SetStatusCondition(&s.instance.Status.Conditions, newCondition)

	if reflect.DeepEqual(s.instance.Status, instance.Status) {
		return nil, nil, nil
	}

	if err := r.Status().Update(ctx, instance); err != nil {
		r.log.Error(err)
	}

	return nil, &ctrl.Result{RequeueAfter: time.Second}, err
}
