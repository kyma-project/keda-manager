package reconciler

import (
	"context"

	"github.com/go-errors/errors"
	"github.com/kyma-project/keda-manager/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	safeDeleteStrategy = true
)

var (
	DeletionErr = errors.New("deletion error")
)

func sFnDeleteResources(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if safeDeleteStrategy {
		return switchState(sFnSafeDeleteStrategy)
	}
	return switchState(sFnCascadeDeleteStrategy)
}

func sFnCascadeDeleteStrategy(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	return deleteResourcesWithFilter(ctx, r, s, alwaysTrueFilter)
}

func sFnSafeDeleteStrategy(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	return deleteResourcesWithFilter(ctx, r, s, withoutCRDFilter)
}

func withoutCRDFilter(u unstructured.Unstructured) bool {
	if u.GroupVersionKind().GroupKind() == apiextensionsv1.Kind("CustomResourceDefinition") {
		return false
	}

	return true
}

func alwaysTrueFilter(unstructured.Unstructured) bool {
	return true
}

type filterFunc func(unstructured.Unstructured) bool

func deleteResourcesWithFilter(ctx context.Context, r *fsm, s *systemState, filterFunc filterFunc) (stateFn, *ctrl.Result, error) {
	var err error
	for _, obj := range r.Objs {
		if !filterFunc(obj) {
			r.log.
				With("objName", obj.GetName()).
				With("gvk", obj.GroupVersionKind()).
				Debug("skipped")
			continue
		}

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
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeletionErr,
			DeletionErr,
		)
		// stop state machine with an error and requeue reconciliation in 1min
		return stopWithErrorAnNoRequeue(err)
	}
	return switchState(sFnRemoveFinalizer)
}
