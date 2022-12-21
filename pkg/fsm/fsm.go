package fsm

import (
	"context"
	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"go.uber.org/zap"
	apixtv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"reflect"
	"runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type stateFn func(context.Context, *reconciler, *systemState, *out) stateFn

type cfg struct {
	crds []unstructured.Unstructured
}
type systemState struct {
	instance v1alpha1.Keda
}
func (s *systemState) setConditionInstalledFalse(reason v1alpha1.InstalledConditionReason, msg string) {
	instanceCopy := s.instance.DeepCopy()
	meta.SetStatusCondition(&instanceCopy.Status.Conditions, metav1.Condition{
		Type:               string(v1alpha1.ConditionTypeInstalled),
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            msg,
	})
}
type k8s struct {
	client client.Client
}

type out struct {
	result ctrl.Result
	err    error
}

type reconciler struct {
	fn  stateFn
	log *zap.SugaredLogger
	k8s
	cfg
}

func (m *reconciler) stateFnName() string {
	fullName := runtime.FuncForPC(reflect.ValueOf(m.fn).Pointer()).Name()
	splitFullName := strings.Split(fullName, ".")

	if len(splitFullName) < 3 {
		return fullName
	}

	shortName := splitFullName[2]
	return shortName
}

func (m *reconciler) reconcile(ctx context.Context, v v1alpha1.Keda) (ctrl.Result, error) {
	state := systemState{instance: v}
	out := out{}
loop:
	for m.fn != nil {
		select {
		case <-ctx.Done():
			out.err = ctx.Err()
			break loop

		default:
			m.log.With("stateFn", m.stateFnName()).Info("next state")
			m.fn = m.fn(ctx, m, &state, &out)
		}
	}

	m.log.
		With("requeueAfter", out.result.RequeueAfter).
		With("requeue", out.result.Requeue).
		With("error", out.err).
		Info("reconciliation result")

	return out.result, out.err
}

func sFnApplyObj(ctx context.Context, r *reconciler, s *systemState, out *out) stateFn {
	panic("not implemented yet")
}

func sFnApplyCRDs(ctx context.Context, r *reconciler, s *systemState, out *out) stateFn {
	var applied bool
	applied, out.err = applyCRDs(ctx, r.client, r.cfg.crds)

	if out.err != nil {
		s.setConditionInstalledFalse(v1alpha1.ConditionReasonCrdError, out.err.Error())
		if err := r.client.Status().Update(ctx, &s.instance); err != nil {
			r.log.Warn("unable to change state")
		}
		return nil
	}
	// all CRDs already exist - goto applyObj
	if !applied {
		return sFnApplyObj
	}
	// all CRDs applied
	return nil
}

func applyCRDs(ctx context.Context, c client.Client, crds []unstructured.Unstructured) (bool, error) {
	var installed bool

	for _, obj := range crds {
		var crd apixtv1.CustomResourceDefinition
		keyObj := client.ObjectKeyFromObject(&obj)

		err := c.Get(ctx, keyObj, &crd)

		// error while getting crd
		if client.IgnoreNotFound(err) != nil {
			return false, err
		}

		// crd exists - continue with crds installation
		if err == nil {
			continue
		}

		// crd does not exit - create it
		if err = c.Create(ctx, &obj); err != nil {
			return false, err
		}

		installed = true
	}

	return installed, nil
}
