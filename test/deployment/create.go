package deployment

import (
	"github.com/kyma-project/keda-manager/test/utils"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	DeploymentName = "test-deployment"
)

func Create(testutil *utils.TestUtils) error {
	deploy := createTestDeployment(testutil)

	return testutil.Client.Create(testutil.Ctx, deploy)
}

func createTestDeployment(testutil *utils.TestUtils) *v1.Deployment {
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testutil.DeploymentName,
			Namespace: testutil.Namespace,
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": testutil.DeploymentName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": testutil.DeploymentName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "alpine",
							Name:  testutil.DeploymentName,
							Command: []string{
								"tail", "-f", "/dev/null",
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: k8sresource.MustParse("100m"),
								},
							},
						},
					},
				},
			},
		},
	}
}
