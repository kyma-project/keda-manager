package scaledobject

import (
	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kyma-project/keda-manager/test/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var (
	MinReplicaCount = ptr.To[int32](2)
	MaxReplicaCount = ptr.To[int32](2)
)

func Create(utils *utils.TestUtils) error {
	scaledObject := fixScaledObject(utils)

	return utils.Client.Create(utils.Ctx, scaledObject)
}

func fixScaledObject(utils *utils.TestUtils) *v1alpha1.ScaledObject {
	return &v1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.ScaledObjectName,
			Namespace: utils.Namespace,
		},
		Spec: v1alpha1.ScaledObjectSpec{
			MinReplicaCount: MinReplicaCount,
			MaxReplicaCount: MaxReplicaCount,
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: utils.DeploymentName,
			},
			Triggers: []v1alpha1.ScaleTriggers{
				{
					Type:       "cpu",
					MetricType: "Utilization",
					Metadata: map[string]string{
						"value": "60",
					},
				},
			},
		},
	}
}
