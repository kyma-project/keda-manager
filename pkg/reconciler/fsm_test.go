package reconciler

import (
	"errors"
	"testing"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apirt "k8s.io/apimachinery/pkg/runtime"
)

func Test_updateObj_convert_errors(t *testing.T) {
	var errTest = errors.New("test error")

	type args struct {
		toUnstructed   func(interface{}) (map[string]interface{}, error)
		fromUnstructed func(map[string]interface{}, interface{}) error
	}

	u := unstructured.Unstructured{}
	u.SetName(operatorName)
	u.SetAPIVersion("apps/v1")
	u.SetKind("Deployment")

	tests := []struct {
		name          string
		args          args
		expectedError error
	}{
		{
			name: "from unstructed fail",
			args: args{
				fromUnstructed: func(u map[string]interface{}, obj interface{}) error {
					return errTest
				},
				toUnstructed: apirt.DefaultUnstructuredConverter.ToUnstructured,
			},
			expectedError: errTest,
		},
		{
			name: "to unstructed fail",
			args: args{
				toUnstructed: func(obj interface{}) (map[string]interface{}, error) {
					return nil, errTest
				},
				fromUnstructed: apirt.DefaultUnstructuredConverter.FromUnstructured,
			},
			expectedError: errTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toUnstructed = tt.args.toUnstructed
			fromUnstructured = tt.args.fromUnstructed

			err := updateObj(&u, nil, func(*appsv1.Deployment, interface{}) error {
				t.Log("deployment updated")
				return nil
			})

			g := NewWithT(t)

			g.Expect(err).Should(HaveOccurred())
			g.Expect(err).Should(Equal(tt.expectedError))
		})
	}

}

func Test_UpdateupdateDeploymentLabels(t *testing.T) {
	t.Run("enable istio sidecar injection", func(t *testing.T) {
		t.Setenv("KEDA_MODULE_VERSION", "test")

		deployment := appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"test":                         "test",
							"app.kubernetes.io/managed-by": "upstream",
						},
					},
				},
			},
		}
		config := v1alpha1.IstioCfg{
			EnabledSidecarInjection: true,
		}

		expectedLabels := map[string]string{
			"test":                         "test",
			"sidecar.istio.io/inject":      "true",
			"app.kubernetes.io/managed-by": "keda-manager",
			"kyma-project.io/module":       "keda",
			"app.kubernetes.io/part-of":    "keda-manager",
			"app.kubernetes.io/version":    "test",
		}

		err := updateDeploymentLabels(&deployment, config)
		require.NoError(t, err)
		require.EqualValues(t, expectedLabels, deployment.Spec.Template.ObjectMeta.Labels)
	})

	t.Run("disable istio sidecar injection", func(t *testing.T) {
		t.Setenv("KEDA_MODULE_VERSION", "test")

		deployment := appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"test":                         "test",
							"app.kubernetes.io/managed-by": "upstream",
							"sidecar.istio.io/inject":      "true",
						},
					},
				},
			},
		}
		config := v1alpha1.IstioCfg{
			EnabledSidecarInjection: false,
		}

		expectedLabels := map[string]string{
			"test":                         "test",
			"sidecar.istio.io/inject":      "false",
			"app.kubernetes.io/managed-by": "keda-manager",
			"kyma-project.io/module":       "keda",
			"app.kubernetes.io/part-of":    "keda-manager",
			"app.kubernetes.io/version":    "test",
		}

		err := updateDeploymentLabels(&deployment, config)
		require.NoError(t, err)
		require.EqualValues(t, expectedLabels, deployment.Spec.Template.ObjectMeta.Labels)
	})
}
