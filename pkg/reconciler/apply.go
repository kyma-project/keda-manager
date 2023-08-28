package reconciler

import (
	"context"
	"errors"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/kyma-project/keda-manager/pkg/annotation"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	InstallationErr = errors.New("installation error")
)

func sFnApply(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	var isError bool
	for _, obj := range r.Objs {
		r.log.
			With("gvk", obj.GetObjectKind().GroupVersionKind()).
			With("name", obj.GetName()).
			With("ns", obj.GetNamespace()).
			Debug("applying")

		obj = annotation.AddDoNotEditDisclaimer(obj)
		err := r.Patch(ctx, &obj, client.Apply, &client.PatchOptions{
			Force:        pointer.Bool(true),
			FieldManager: "keda-manager",
		})

		if err != nil {
			r.log.With("err", err).Error("apply error")
			isError = true
		}

		s.objs = append(s.objs, obj)
	}
	// no errors
	if !isError {
		return switchState(sFnVerify)
	}

	s.instance.UpdateStateFromErr(
		v1alpha1.ConditionTypeInstalled,
		v1alpha1.ConditionReasonApplyObjError,
		InstallationErr,
	)
	return stopWithNoRequeue()
}
