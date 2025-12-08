package scaledobject

import (
	"fmt"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kyma-project/keda-manager/test/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
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
			MinReplicaCount: ptr.To[int32](1),
			MaxReplicaCount: ptr.To[int32](5),
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: utils.DeploymentName,
			},
			Triggers: []v1alpha1.ScaleTriggers{
				{
					MetricType:       "Value",
					Name:             "activeTenantsMax",
					Type:             "metrics-api",
					UseCachedMetrics: true,
					Metadata: map[string]string{
						"format":        "json",
						"targetValue":   "1",
						"url":           fmt.Sprintf("http://%s.%s.svc.cluster.local:%d%s", utils.MetricsServerName, utils.Namespace, utils.MetricsServerPort, utils.MetricsServerEndpoint),
						"valueLocation": "value",
					},
				},
			},
		},
	}
}
