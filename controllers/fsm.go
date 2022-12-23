package controllers

import (
	"context"
	"fmt"
	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"reflect"
	"runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type stateFn func(context.Context, *reconciler, *systemState) (stateFn, *ctrl.Result, error)

// module specific configuuration
type cfg struct {
	// the finalizer identifies the module and is is used to delete
	// the module resources
	finalizer string
	// the objects are module component parts; objects are applied
	// on the cluster one by one with given order
	objs []unstructured.Unstructured
}

// the state of controlled system (k8s cluster)
type systemState struct {
	instance v1alpha1.Keda
	// the state of module component parts on cluster used detect
	// module readiness
	objs []unstructured.Unstructured
}

type k8s struct {
	client client.Client
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
	var err error
	var result *ctrl.Result
loop:
	for m.fn != nil && err == nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break loop

		default:
			m.log.Info(fmt.Sprintf("switching state: %s", m.stateFnName()))
			m.fn, result, err = m.fn(ctx, m, &state)
		}
	}

	m.log.
		With("error", err).
		With("result", result).
		Info("reconciliation done")

	if result != nil {
		return *result, err
	}

	return ctrl.Result{}, err
}
