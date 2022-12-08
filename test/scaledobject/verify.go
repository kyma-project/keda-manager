package scaledobject

import (
	"fmt"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kyma-project/keda-manager/test/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func VerifyDeletion(utils *utils.TestUtils) error {
	return client.IgnoreNotFound(Verify(utils))
}

func Verify(utils *utils.TestUtils) error {
	var keda v1alpha1.ScaledObject
	objectKey := client.ObjectKey{
		Name:      utils.ScaledObjectName,
		Namespace: utils.Namespace,
	}

	if err := utils.Client.Get(utils.Ctx, objectKey, &keda); err != nil {
		return err
	}

	return verify(utils, &keda)
}

func verify(utils *utils.TestUtils, keda *v1alpha1.ScaledObject) error {
	for i := range keda.Status.Conditions {
		condition := keda.Status.Conditions[i]
		if condition.Type == v1alpha1.ConditionReady && condition.Status == "True" {
			return nil
		}
	}

	return fmt.Errorf("scaledobject '%s' not ready", utils.ScaledObjectName)
}
