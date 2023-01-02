package reconciler

import (
	"errors"
	"testing"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apirt "k8s.io/apimachinery/pkg/runtime"
)

var (
	testLevel    v1alpha1.OperatorLogLevel = "test"
	testFormat   v1alpha1.LogFormat        = "test"
	timeEncoding v1alpha1.LogTimeEncoding  = "test"
)

func newTestDeployment(args []string) *appsv1.Deployment {
	var result appsv1.Deployment

	container := corev1.Container{
		Args: args,
	}
	result.Spec.Template.Spec.Containers = []corev1.Container{container}
	return &result
}

func newUpdateOperatorArgsDeployment() *appsv1.Deployment {
	return newTestDeployment([]string{
		"--leader-elect",
		"--zap-log-level=info",
		"--zap-encoder=console",
		"--zap-time-encoding=rfc3339",
	})
}

//func Test_updateOperatorArgs(t *testing.T) {
//	type args struct {
//		cfg v1alpha1.LoggingOperatorCfg
//		d   *appsv1.Deployment
//	}
//	tests := []struct {
//		name     string
//		args     args
//		wantArgs []string
//	}{
//		{
//			name: "all override",
//			args: args{
//				cfg: v1alpha1.LoggingOperatorCfg{
//					Level:        &testLevel,
//					Format:       &testFormat,
//					TimeEncoding: &timeEncoding,
//				},
//				d: newUpdateOperatorArgsDeployment(),
//			},
//			wantArgs: []string{
//				"--leader-elect",
//				"--zap-log-level=test",
//				"--zap-encoder=test",
//				"--zap-time-encoding=test",
//			},
//		},
//		{
//			name: "override level",
//			args: args{
//				cfg: v1alpha1.LoggingOperatorCfg{
//					Level: &testLevel,
//				},
//				d: newUpdateOperatorArgsDeployment(),
//			},
//			wantArgs: []string{
//				"--leader-elect",
//				"--zap-log-level=test",
//				"--zap-encoder=console",
//				"--zap-time-encoding=rfc3339",
//			},
//		},
//		{
//			name: "override encoder",
//			args: args{
//				cfg: v1alpha1.LoggingOperatorCfg{
//					Format: &testFormat,
//				},
//				d: newUpdateOperatorArgsDeployment(),
//			},
//			wantArgs: []string{
//				"--leader-elect",
//				"--zap-log-level=info",
//				"--zap-encoder=test",
//				"--zap-time-encoding=rfc3339",
//			},
//		},
//		{
//			name: "override time encoding",
//			args: args{
//				cfg: v1alpha1.LoggingOperatorCfg{
//					TimeEncoding: &timeEncoding,
//				},
//				d: newUpdateOperatorArgsDeployment(),
//			},
//			wantArgs: []string{
//				"--leader-elect",
//				"--zap-log-level=info",
//				"--zap-encoder=console",
//				"--zap-time-encoding=test",
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			updateOperatorArgs(tt.args.cfg, tt.args.d)
//			NewWithT(t).
//				Expect(tt.args.d.Spec.Template.Spec.Containers[0].Args).
//				To(ContainElements(tt.wantArgs))
//		})
//	}
//}

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
