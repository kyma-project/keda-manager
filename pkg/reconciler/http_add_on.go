package reconciler

import (
	"context"
	"errors"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/kyma-project/keda-manager/pkg/annotation"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	httpAddOnAnnotationKey     = "keda.kyma-project.io/http-add-on"
	httpAddOnAnnotationEnabled = "enabled"
)

func httpAddOnEnabled(instance *v1alpha1.Keda) bool {
	annotations := instance.GetAnnotations()
	if annotations == nil {
		return false
	}
	return annotations[httpAddOnAnnotationKey] == httpAddOnAnnotationEnabled
}

func sFnHttpAddOnDecision(_ context.Context, _ *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if httpAddOnEnabled(&s.instance) {
		return switchState(sFnApplyHttpAddOn)
	}
	return switchState(sFnDeleteHttpAddOn)
}

func sFnApplyHttpAddOn(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if len(r.HttpAddOnObjs) == 0 {
		s.instance.Status.Conditions = append(s.instance.Status.Conditions, metav1.Condition{
			Type:               string(v1alpha1.ConditionTypeHttpAddOnInstalled),
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: metav1.Now(),
			Reason:             string(v1alpha1.ConditionReasonHttpAddOnInstallErr),
			Message:            "no http-add-on manifests available",
		})
		return stopWithNoRequeue()
	}

	var installErr error
	for _, obj := range r.HttpAddOnObjs {
		obj = annotation.AddDoNotEditDisclaimer(obj)
		obj.SetLabels(setCommonLabels(obj.GetLabels()))

		err := r.Patch(ctx, &obj, client.Apply, &client.PatchOptions{
			Force:        ptr.To[bool](true),
			FieldManager: "keda-manager",
		})
		if err != nil {
			r.log.With("err", err).Error("http-add-on apply error")
			installErr = errors.Join(installErr, err)
		}
	}

	if installErr != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeHttpAddOnInstalled,
			v1alpha1.ConditionReasonHttpAddOnInstallErr,
			installErr,
		)
		return stopWithErrorAndNoRequeue(installErr)
	}

	s.instance.UpdateStateReady(
		v1alpha1.ConditionTypeHttpAddOnInstalled,
		v1alpha1.ConditionReasonHttpAddOnInstalled,
		"http-add-on installed",
	)
	return stopWithNoRequeue()
}

func sFnDeleteHttpAddOn(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if len(r.HttpAddOnObjs) == 0 {
		return stopWithNoRequeue()
	}

	condition := meta.FindStatusCondition(s.instance.Status.Conditions, string(v1alpha1.ConditionTypeHttpAddOnInstalled))
	if condition == nil || condition.Status != metav1.ConditionTrue {
		return stopWithNoRequeue()
	}

	var deletionErr error
	for _, obj := range r.HttpAddOnObjs {
		err := r.Delete(ctx, &obj)
		if client.IgnoreNotFound(err) != nil {
			r.log.With("err", err).Error("http-add-on delete error")
			deletionErr = errors.Join(deletionErr, err)
		}
	}

	if deletionErr != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeHttpAddOnInstalled,
			v1alpha1.ConditionReasonHttpAddOnInstallErr,
			deletionErr,
		)
		return stopWithErrorAndNoRequeue(deletionErr)
	}

	condition.Status = metav1.ConditionFalse
	condition.Reason = string(v1alpha1.ConditionReasonHttpAddOnNotInstalled)
	condition.Message = "http-add-on not installed"
	condition.LastTransitionTime = metav1.Now()
	meta.SetStatusCondition(&s.instance.Status.Conditions, *condition)

	return stopWithNoRequeue()
}
