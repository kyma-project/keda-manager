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

type stateFn func(context.Context, *reconciler, *systemState, *out) stateFn

type cfg struct {
	finalizer string
	objs      []unstructured.Unstructured
}

type systemState struct {
	instance v1alpha1.Keda
	objs []unstructured.Unstructured
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
			m.log.Info(fmt.Sprintf("switching state: %s", m.stateFnName()))
			m.fn = m.fn(ctx, m, &state, &out)
		}
	}

	m.log.
		With("result", out.result).
		With("error", out.err).
		Info("reconciliation done")

	return out.result, out.err
}
