package metricsserver

import (
	"fmt"

	"github.com/kyma-project/keda-manager/test/utils"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Create(testutil *utils.TestUtils) error {
	cm := createMetricsServerConfigMap(testutil)
	deploy := createMetricsServerDeployment(testutil)
	svc := createMetricsServerService(testutil)

	if err := testutil.Client.Create(testutil.Ctx, cm); err != nil {
		return err
	}

	if err := testutil.Client.Create(testutil.Ctx, deploy); err != nil {
		return err
	}

	return testutil.Client.Create(testutil.Ctx, svc)
}

func createMetricsServerConfigMap(testutil *utils.TestUtils) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testutil.MetricsServerName,
			Namespace: testutil.Namespace,
		},
		Data: map[string]string{
			"nginx.conf": fmt.Sprintf(`
events {}
http {
  server {
    listen %d;
    root /usr/share/nginx/html;
    default_type application/json;
    location = %s {
      return 200 '{"value": %d}';
    }
  }
}`, testutil.MetricsServerPort, testutil.MetricsServerEndpoint, testutil.ScaleDeploymentTo),
		},
	}
}

func createMetricsServerDeployment(testutil *utils.TestUtils) *v1.Deployment {
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testutil.MetricsServerName,
			Namespace: testutil.Namespace,
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": testutil.MetricsServerName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": testutil.MetricsServerName,
					},
					Annotations: map[string]string{
						// besides health check port, excude also the server port to be accessible from keda outside the mesh
						"traffic.sidecar.istio.io/excludeInboundPorts": fmt.Sprintf("%d, 15020", testutil.MetricsServerPort),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "nginx:1.25-alpine",
							Name:  testutil.MetricsServerName,
							Command: []string{
								"tail", "-f", "/dev/null",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "nginx-conf",
									MountPath: "/etc/nginx/nginx.conf",
									SubPath:   "nginx.conf",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: testutil.MetricsServerPort,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "nginx-conf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: testutil.MetricsServerName,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func createMetricsServerService(testutil *utils.TestUtils) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testutil.MetricsServerName,
			Namespace: testutil.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": testutil.MetricsServerName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       testutil.MetricsServerPort,
					TargetPort: intstr.FromInt32(testutil.MetricsServerPort),
				},
			},
		},
	}
}
