package reconciler

import (
	"context"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnUpdateKedaDeployment(_ context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	// proceed next state if no overrides
	if s.instance.Spec.Logging.Operator == nil {
		return switchState(sFnUpdateMetricsServerDeployment)
	}

	u, err := r.kedaManagerDeployment()
	if err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeploymentUpdateErr,
			err,
		)
		return stopWithErrorAnNoRequeue(err)
	}
	// update keda manager's log related arguments
	logCfg := *s.instance.Spec.Logging.Operator
	if err := updateObj(u, logCfg, updateKedaOperatorContainer0Args); err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeploymentUpdateErr,
			err,
		)
		return stopWithErrorAnNoRequeue(err)
	}

	if s.instance.Spec.Resources == nil || s.instance.Spec.Resources.Operator == nil {
		return switchState(sFnApply)
	}

	resources := *s.instance.Spec.Resources.Operator
	if err := updateObj(u, resources, updateKedaContanier0Resources); err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeploymentUpdateErr,
			err,
		)
		// this error is not recoverable, stop state machine and return reconciliation error
		return stopWithErrorAndRequeue(err)
	}

	if s.instance.Spec.Env == nil {
		return switchState(sFnApply)
	}
	if err := updateObj(u, s.instance.Spec.Env, updateKedaContanierEnvs); err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeploymentUpdateErr,
			err,
		)
		// this error is not recoverable, stop state machine and return reconciliation error
		return stopWithErrorAndRequeue(err)
	}

	return switchState(sFnUpdateMetricsServerDeployment)
}

func sFnUpdateMetricsServerDeployment(_ context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	// proceed next state if no overrides
	if s.instance.Spec.Logging.MetricsServer == nil {
		return switchState(sFnApply)
	}

	u, err := r.kedaMetricsServerDeployment()
	if err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeploymentUpdateErr,
			err,
		)
		return stopWithErrorAnNoRequeue(err)
	}
	// update keda metrics server's log related arguments
	logCfg := *s.instance.Spec.Logging.MetricsServer
	if err := updateObj(u, logCfg, updateKedaMetricsServerContainer0Args); err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeploymentUpdateErr,
			err,
		)
		return stopWithErrorAnNoRequeue(err)
	}

	if s.instance.Spec.Resources == nil || s.instance.Spec.Resources.MetricsServer == nil {
		return switchState(sFnApply)
	}

	resources := *s.instance.Spec.Resources.MetricsServer
	if err := updateObj(u, resources, updateKedaContanier0Resources); err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeploymentUpdateErr,
			err,
		)
		// this error is not recoverable, stop state machine and return reconciliation error
		return stopWithErrorAndRequeue(err)
	}

	if err := updateObj(u, v1alpha1.EnvVars(s.instance.Spec.Env), updateKedaContanierEnvs); err != nil {
		s.instance.UpdateStateFromErr(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonDeploymentUpdateErr,
			err,
		)
		// this error is not recoverable, stop state machine and return reconciliation error
		return stopWithErrorAndRequeue(err)
	}

	return switchState(sFnApply)
}
