package reconciler

import (
	"context"
	"fmt"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func sFnServedFilter(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if s.instance.IsServedEmpty() || s.instance.Status.Served == v1alpha1.ServedFalse {
		// keda CRs check
		servedKeda, err := findServedKeda(ctx, r.Client)
		if err != nil {
			return stopWithErrorAndNoRequeue(err)
		}

		s.instance.UpdateServed(v1alpha1.ServedTrue)
		if servedKeda != nil {
			s.instance.UpdateServed(v1alpha1.ServedFalse)
			s.instance.UpdateStateFromErr(v1alpha1.ConditionTypeInstalled, v1alpha1.ConditionReasonKedaDuplicated,
				fmt.Errorf("only one instance of Keda is allowed (current served instance: %s/%s)",
					servedKeda.GetNamespace(), servedKeda.GetName()))
		}

		return stopWithRequeue()
	}

	return switchState(sFnTakeSnapshot)
}

func findServedKeda(ctx context.Context, c client.Client) (*v1alpha1.Keda, error) {
	var kedaList v1alpha1.KedaList

	err := c.List(ctx, &kedaList)

	if err != nil {
		return nil, err
	}

	for _, item := range kedaList.Items {
		if !item.IsServedEmpty() && item.Status.Served == v1alpha1.ServedTrue {
			return &item, nil
		}
	}

	return nil, nil
}
