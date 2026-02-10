package reconciler

import (
	"context"
	"time"

	"github.com/kyma-project/keda-manager/api/v1alpha1"

	"github.com/kyma-project/manager-toolkit/installation/base/resource"
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnVerify(_ context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	var ready int
	var replicaFailure int
	var kedaVersion string

	for _, obj := range s.objs {
		if !isDeployment(obj) {
			continue
		}

		var deployment appsv1.Deployment
		if err := fromUnstructured(obj.Object, &deployment); err != nil {
			s.instance.UpdateStateFromErr(
				v1alpha1.ConditionTypeInstalled,
				v1alpha1.ConditionReasonVerificationErr,
				err,
			)
			return stopWithErrorAndNoRequeue(err)
		}

		if deployment.GetName() == "keda-operator" {
			kedaVersion = deployment.GetLabels()["app.kubernetes.io/version"]
		}

		if resource.IsDeploymentReady(deployment) {
			r.log.Debugf("successfully verified keda deployment %s/%s", obj.GetNamespace(), obj.GetName())
			ready++
		}

		if hasDeployReplicaFailure(deployment) {
			replicaFailure++
		}

		r.log.Info("%s deployment is current deployment", deployment.GetName())
		if hasBootstrapperAnnotations(deployment) {
			r.log.Info("%s deployment does not have expected annotations", deployment.GetName())
		}
	}

	if replicaFailure > 0 {
		r.log.Debugf("%d deployments have ReplicaFailure condition", replicaFailure)
		s.instance.UpdateStateReplicaFailure(
			v1alpha1.ConditionTypeDeploymentFailure,
			v1alpha1.ConditionReasonDeploymentReplicaFailure,
			"one or more deployment/s have ReplicaFailure condition",
		)
		return stopWithRequeueAfter(time.Second * 10)
	}

	if ready != 3 {
		r.log.Debugf("%d deployments in ready state found ( 3 are expected ) ", ready)
		s.instance.UpdateStateProcessing(
			v1alpha1.ConditionTypeInstalled,
			v1alpha1.ConditionReasonVerification,
			"verification in progress",
		)
		return stopWithRequeueAfter(time.Second * 10)
	}

	// remove possible previous DeploymentFailure condition
	s.instance.RemoveCondition(v1alpha1.ConditionTypeDeploymentFailure)

	if s.instance.Status.State == "Ready" {
		return stopWithNoRequeue()
	}

	s.instance.Status.KedaVersion = kedaVersion
	s.instance.UpdateStateReady(
		v1alpha1.ConditionTypeInstalled,
		v1alpha1.ConditionReasonVerified,
		"keda-operator and keda-operator-metrics-server ready",
	)
	return stopWithNoRequeue()
}

func hasDeployReplicaFailure(deployment appsv1.Deployment) bool {
	return resource.HasDeploymentConditionTrueStatus(deployment.Status.Conditions, appsv1.DeploymentReplicaFailure)
}

func hasBootstrapperAnnotations(deployment appsv1.Deployment) bool {
	annotations := deployment.GetAnnotations()
	if annotations == nil {
		return false
	}

	_, okPull := annotations[v1alpha1.KymaBootstraperAddImagePullSecretMutation]
	_, okReg := annotations[v1alpha1.KymaBootstraperRegistryUrlMutation]

	return okPull && okReg
}
