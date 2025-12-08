package deployment

import (
	"fmt"

	"github.com/kyma-project/keda-manager/test/utils"
	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Verify(testutil *utils.TestUtils) error {
	return verifyDeploymentReplicas(testutil)
}

func verifyDeploymentReplicas(testutil *utils.TestUtils) error {
	deploy := &v1.Deployment{}
	err := testutil.Client.Get(testutil.Ctx, client.ObjectKey{
		Namespace: testutil.Namespace,
		Name:      testutil.DeploymentName,
	}, deploy)
	if err != nil {
		return err
	}

	if deploy.Status.AvailableReplicas != testutil.ScaleDeploymentTo {
		return fmt.Errorf("deployment '%s' has availableReplicas == '%d', expected '%d'", testutil.DeploymentName, deploy.Status.AvailableReplicas, testutil.ScaleDeploymentTo)
	}

	if deploy.Status.Replicas != testutil.ScaleDeploymentTo {
		return fmt.Errorf("deployment '%s' has replicas == '%d', expected '%d'", testutil.DeploymentName, deploy.Status.Replicas, testutil.ScaleDeploymentTo)
	}
	
	return nil
}
