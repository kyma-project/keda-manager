package hpa

import (
	"fmt"

	"github.com/kyma-project/keda-manager/test/utils"
	v2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	hpaLabel = "scaledobject.keda.sh/name"
)

func VerifyDeletion(utils *utils.TestUtils) error {
	return client.IgnoreNotFound(Verify(utils))
}

func Verify(utils *utils.TestUtils) error {
	labelSelector, err := labels.NewRequirement(hpaLabel, selection.DoubleEquals, []string{utils.ScaledObjectName})
	if err != nil {
		return err
	}

	var hpa v2.HorizontalPodAutoscalerList
	err = utils.Client.List(utils.Ctx, &hpa, &client.ListOptions{
		Namespace:     utils.Namespace,
		LabelSelector: labels.NewSelector().Add(*labelSelector),
	})
	if err != nil {
		return err
	}
	if len(hpa.Items) != 1 {
		return fmt.Errorf("found '%d' hpas, expected '1'", len(hpa.Items))
	}

	return verify(utils, &hpa.Items[0])
}

func verify(utils *utils.TestUtils, hpa *v2.HorizontalPodAutoscaler) error {
	if *hpa.Spec.MinReplicas != 1 {
		return fmt.Errorf("hpa '%s' has minReplicas == '%d', expected 1", hpa.Name, *hpa.Spec.MinReplicas)
	}

	if hpa.Spec.MaxReplicas != 5 {
		return fmt.Errorf("hpa '%s' has maxReplicas == '%d', expected 5", hpa.Name, hpa.Spec.MaxReplicas)
	}

	if hpa.Status.CurrentReplicas != utils.ScaleDeploymentTo {
		return fmt.Errorf("hpa '%s' has currentReplicas == '%d', expected '%d'", hpa.Name, hpa.Status.CurrentReplicas, utils.ScaleDeploymentTo)
	}

	return verifyHpaCondition(hpa)
}

func verifyHpaCondition(hpa *v2.HorizontalPodAutoscaler) error {
	for _, condition := range hpa.Status.Conditions {
		if condition.Type == v2.AbleToScale && condition.Status == corev1.ConditionTrue {
			return nil
		}
	}

	return fmt.Errorf("hpa '%s' is not ready", hpa.Name)
}
