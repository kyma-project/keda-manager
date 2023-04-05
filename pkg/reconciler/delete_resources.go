package reconciler

import (
	"context"
	"fmt"

	"github.com/go-errors/errors"
	"github.com/kyma-project/keda-manager/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultDeletionStrategy = safeDeletionStrategy
	kedaOperatorLeaseName   = "operator.keda.sh"
	kedaManagerLeaseName    = "4123c01c.operator.kyma-project.io"
)

var (
	DeletionErr       = errors.New("deletion error")
	KedaOperatorLease = fixLeaseObject(kedaOperatorLeaseName)
	KedaManagerLease  = fixLeaseObject(kedaManagerLeaseName)
)

func fixLeaseObject(leaseName string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "Lease",
			"apiVersion": "coordination.k8s.io/v1",
			"metadata": map[string]interface{}{
				"name":      leaseName,
				"namespace": "kyma-system",
			},
		},
	}
}

func sFnDeleteResources(_ context.Context, _ *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if !isKedaDeleting(s) {
		s.instance.UpdateStateDeletion(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeletion,
			"deletion in progress",
		)

		return stopWithRequeue()
	}

	// TODO: thinkg about deletion configuration
	return switchState(deletionStrategyBuilder(defaultDeletionStrategy))
}

type deletionStrategy string

const (
	cascadeDeletionStrategy  deletionStrategy = "cascadeDeletionStrategy"
	safeDeletionStrategy     deletionStrategy = "safeDeletionStrategy"
	upstreamDeletionStrategy deletionStrategy = "upstreamDeletionStrategy"
)

func deletionStrategyBuilder(strategy deletionStrategy) stateFn {
	switch strategy {
	case cascadeDeletionStrategy:
		return sFnCascadeDeletionState
	case upstreamDeletionStrategy:
		return sFnUpstreamDeletionState
	case safeDeletionStrategy:
		return sFnSafeDeletionState
	default:
		return deletionStrategyBuilder(safeDeletionStrategy)
	}
}

func sFnCascadeDeletionState(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	return deleteResourcesWithFilter(ctx, r, s)
}

func sFnUpstreamDeletionState(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	return deleteResourcesWithFilter(ctx, r, s, withoutCRDFilter)
}

func sFnSafeDeletionState(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if err := checkCRDOrphanResources(ctx, r); err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeletionErr,
			err,
		)

		// stop state machine with an error and requeue reconciliation in 1min
		return stopWithErrorAndNoRequeue(err)
	}

	return deleteResourcesWithFilter(ctx, r, s)
}

func withoutCRDFilter(u unstructured.Unstructured) bool {
	return !isCRD(u)
}

type filterFunc func(unstructured.Unstructured) bool

func deleteResourcesWithFilter(ctx context.Context, r *fsm, s *systemState, filterFunc ...filterFunc) (stateFn, *ctrl.Result, error) {
	var err error

	//ensure lease object will be removed as well
	r.Objs = append(r.Objs, KedaManagerLease, KedaOperatorLease)

	for _, obj := range r.Objs {
		if !fitToFilters(obj, filterFunc...) {
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
		return stopWithErrorAndNoRequeue(err)
	}
	return switchState(sFnRemoveFinalizer)
}

func fitToFilters(u unstructured.Unstructured, filterFunc ...filterFunc) bool {
	for _, fn := range filterFunc {
		if !fn(u) {
			return false
		}
	}

	return true
}

func checkCRDOrphanResources(ctx context.Context, r *fsm) error {
	for _, obj := range r.Objs {
		if !isCRD(obj) {
			continue
		}

		crdList, err := buildResourceListFromCRD(obj)
		if err != nil {
			return err
		}

		err = r.List(ctx, &crdList)
		if err != nil {
			return err
		}

		if len(crdList.Items) > 0 {
			return fmt.Errorf("found %d items with VersionKind %s", len(crdList.Items), crdList.GetAPIVersion())
		}
	}

	return nil
}

func isCRD(u unstructured.Unstructured) bool {
	return u.GroupVersionKind().GroupKind() == apiextensionsv1.Kind("CustomResourceDefinition")
}

func buildResourceListFromCRD(u unstructured.Unstructured) (unstructured.UnstructuredList, error) {
	crd := apiextensionsv1.CustomResourceDefinition{}
	crdList := unstructured.UnstructuredList{}

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &crd)
	if err != nil {
		return crdList, err
	}

	crdList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   crd.Spec.Group,
		Version: getCRDStoredVersion(crd),
		Kind:    crd.Spec.Names.Kind,
	})

	return crdList, nil
}

func getCRDStoredVersion(crd apiextensionsv1.CustomResourceDefinition) string {
	for _, version := range crd.Spec.Versions {
		if version.Storage {
			return version.Name
		}
	}

	return ""
}

func isKedaDeleting(s *systemState) bool {
	condition := meta.FindStatusCondition(s.instance.Status.Conditions, string(v1alpha1.ConditionTypeInstalled))
	if condition == nil {
		return false
	}

	if condition.Reason != string(v1alpha1.ConditionReasonDeletion) &&
		condition.Reason != string(v1alpha1.ConditionReasonDeletionErr) {
		return false
	}

	return true
}
