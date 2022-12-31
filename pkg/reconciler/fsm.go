package reconciler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apirt "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type stateFn func(context.Context, *fsm, *systemState) (stateFn, *ctrl.Result, error)

// module specific configuuration
type Cfg struct {
	// the Finalizer identifies the module and is is used to delete
	// the module resources
	Finalizer string
	// the objects are module component parts; objects are applied
	// on the cluster one by one with given order
	Objs []unstructured.Unstructured
}

var (
	fromUnstructured = apirt.DefaultUnstructuredConverter.FromUnstructured
	toUnstructed     = apirt.DefaultUnstructuredConverter.ToUnstructured
)

func updateObj(u *unstructured.Unstructured, updateDeployment func(*appsv1.Deployment)) error {
	var result appsv1.Deployment
	err := fromUnstructured(u.Object, &result)
	if err != nil {
		return err
	}

	updateDeployment(&result)
	u.Object, err = toUnstructed(&result)
	return err
}

type argumentPredicate func(*string) (string, bool)

var (
	isLevel = func(s *string) (string, bool) {
		return zapLogLevel, s != nil && strings.HasPrefix(*s, zapLogLevel)
	}
	isFormat = func(s *string) (string, bool) {
		return zapEncoder, s != nil && strings.HasPrefix(*s, zapEncoder)
	}
	isTimeEncoding = func(s *string) (string, bool) {
		return zapTimeEncoding, s != nil && strings.HasPrefix(*s, zapTimeEncoding)
	}
)

func updateLogArgument(cfg v1alpha1.LoggingOperatorCfg, arg *string) {
	for _, pkv := range []struct {
		p argumentPredicate
		v string
		d string
	}{
		{
			p: isLevel,
			v: cfg.Level.String(),
			d: "info",
		},
		{
			p: isFormat,
			v: cfg.Format.String(),
			d: "console",
		},
		{
			p: isTimeEncoding,
			v: cfg.TimeEncoding.String(),
			d: "rfc3339",
		},
	} {
		key, ok := pkv.p(arg)
		if !ok {
			continue
		}
		value := pkv.v
		if value == "" {
			value = pkv.d
		}
		result := fmt.Sprintf("%s=%s", key, value)
		*arg = result
		return
	}
}

func (c *Cfg) operatorDeployment() *unstructured.Unstructured {
	for i := range c.Objs {
		if !isOperator(c.Objs[i]) {
			continue
		}

		return &c.Objs[i]
	}
	return nil
}

func (cfg *Cfg) updateOperatorLogging2(logCfg v1alpha1.LoggingOperatorCfg) error {
	u := cfg.operatorDeployment()
	if u == nil {
		return fmt.Errorf("%w: operator object", ErrNotFound)
	}

	var deployment appsv1.Deployment
	err := fromUnstructured(u.Object, &deployment)
	if err != nil {
		return err
	}

	for i := range deployment.Spec.Template.Spec.Containers[0].Args {
		updateLogArgument(logCfg, &deployment.Spec.Template.Spec.Containers[0].Args[i])
	}

	u.Object, err = toUnstructed(&deployment)
	return err
}

// the state of controlled system (k8s cluster)
type systemState struct {
	instance v1alpha1.Keda
	// the state of module component parts on cluster used detect
	// module readiness
	objs []unstructured.Unstructured
}

const (
	operatorName    = "keda-manager"
	zapLogLevel     = "--zap-log-level"
	zapEncoder      = "--zap-encoder"
	zapTimeEncoding = "--zap-time-encoding"
)

type predicate func(unstructured.Unstructured) bool

var (
	ErrContextDone = errors.New("context done")
	ErrNotFound    = errors.New("not found")

	hasOperatorName predicate = func(u unstructured.Unstructured) bool {
		return u.GetName() == operatorName
	}
	isDeployment predicate = func(u unstructured.Unstructured) bool {
		return u.GetKind() == "Deployment" &&
			u.GetAPIVersion() == "apps/v1"
	}
	isOperator predicate = func(u unstructured.Unstructured) bool {
		return hasOperatorName(u) && isDeployment(u)
	}
)

func updateOperatorArgs(cfg v1alpha1.LoggingOperatorCfg, d *appsv1.Deployment) {
	for i, arg := range d.Spec.Template.Spec.Containers[0].Args {

		if cfg.Level != nil && strings.HasPrefix(arg, zapLogLevel) {
			d.Spec.Template.Spec.Containers[0].Args[i] =
				fmt.Sprintf("%s=%s", zapLogLevel, *cfg.Level)
		}

		if cfg.Format != nil && strings.HasPrefix(arg, zapEncoder) {
			d.Spec.Template.Spec.Containers[0].Args[i] =
				fmt.Sprintf("%s=%s", zapEncoder, *cfg.Format)
		}

		if cfg.TimeEncoding != nil && strings.HasPrefix(arg, zapTimeEncoding) {
			d.Spec.Template.Spec.Containers[0].Args[i] =
				fmt.Sprintf("%s=%s", zapTimeEncoding, *cfg.TimeEncoding)
		}
	}
}

type K8s struct {
	client.Client
}

type Fsm interface {
	Run(ctx context.Context, v v1alpha1.Keda) (ctrl.Result, error)
}

type fsm struct {
	fn  stateFn
	log *zap.SugaredLogger
	K8s
	Cfg
}

func (m *fsm) stateFnName() string {
	fullName := runtime.FuncForPC(reflect.ValueOf(m.fn).Pointer()).Name()
	splitFullName := strings.Split(fullName, ".")

	if len(splitFullName) < 3 {
		return fullName
	}

	shortName := splitFullName[2]
	return shortName
}

func (m *fsm) Run(ctx context.Context, v v1alpha1.Keda) (ctrl.Result, error) {
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

	m.log.With("error", err).
		With("result", result).
		Info("reconciliation done")

	if result != nil {
		return *result, err
	}

	return ctrl.Result{
		Requeue: false,
	}, err
}

func NewFsm(ctx context.Context, log *zap.SugaredLogger, cfg Cfg, k8s K8s) Fsm {
	return &fsm{
		fn:  sFnInitialize,
		Cfg: cfg,
		log: log,
		K8s: k8s,
	}
}
