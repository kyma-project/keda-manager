package reconciler

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
)

func Test_hasRestrictedAnnotations(t *testing.T) {
	type args struct {
		dep v1alpha1.Keda
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "no restricted annotations",
			args: args{dep: v1alpha1.Keda{Spec: v1alpha1.KedaSpec{PodAnnotations: &v1alpha1.PodAnnotations{
				Operator:         map[string]string{},
				MetricsServer:    map[string]string{},
				AdmissionWebhook: map[string]string{},
			}}}},
			want: false,
		},
		{
			name: "restricted annotations in Operator",
			args: args{dep: v1alpha1.Keda{Spec: v1alpha1.KedaSpec{PodAnnotations: &v1alpha1.PodAnnotations{
				Operator:         map[string]string{v1alpha1.KymaBootstraperRegistryUrlMutation: "maka", v1alpha1.KymaBootstrapperSetFipsMode: "unknown"},
				MetricsServer:    map[string]string{},
				AdmissionWebhook: map[string]string{},
			}}}},
			want: true,
		},
		{
			name: "restricted annotations in MetricsServer",
			args: args{dep: v1alpha1.Keda{Spec: v1alpha1.KedaSpec{PodAnnotations: &v1alpha1.PodAnnotations{
				Operator:         map[string]string{},
				MetricsServer:    map[string]string{v1alpha1.KymaBootstraperAddImagePullSecretMutation: "true", v1alpha1.KymaBootstraperRegistryUrlMutation: "chleb"},
				AdmissionWebhook: map[string]string{},
			}}}},
			want: true,
		},
		{
			name: "restricted annotations in AdmissionWebhook",
			args: args{dep: v1alpha1.Keda{Spec: v1alpha1.KedaSpec{PodAnnotations: &v1alpha1.PodAnnotations{
				Operator:         map[string]string{},
				MetricsServer:    map[string]string{},
				AdmissionWebhook: map[string]string{v1alpha1.KymaBootstraperAddImagePullSecretMutation: "true", v1alpha1.KymaBootstraperRegistryUrlMutation: "paka"},
			}}}},
			want: true,
		},
		{
			name: "restricted annotations in all Keda deployments",
			args: args{dep: v1alpha1.Keda{Spec: v1alpha1.KedaSpec{PodAnnotations: &v1alpha1.PodAnnotations{
				Operator:         map[string]string{v1alpha1.KymaBootstraperAddImagePullSecretMutation: "true", v1alpha1.KymaBootstraperRegistryUrlMutation: "chleb"},
				MetricsServer:    map[string]string{v1alpha1.KymaBootstraperRegistryUrlMutation: "chleb", v1alpha1.KymaBootstraperAddImagePullSecretMutation: "true"},
				AdmissionWebhook: map[string]string{v1alpha1.KymaBootstraperRegistryUrlMutation: "paka", v1alpha1.KymaBootstraperAddImagePullSecretMutation: "true"},
			}}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasRestrictedAnnotations(tt.args.dep)
			require.Equal(t, tt.want, got, "hasRestrictedAnnotations() mismatch")
		})
	}
}

func Test_sFnBootstrappeValidation(t *testing.T) {
	t.Run("no restricted annotations move to the next state", func(t *testing.T) {
		instance := v1alpha1.Keda{
			Spec: v1alpha1.KedaSpec{
				PodAnnotations: &v1alpha1.PodAnnotations{
					Operator:         map[string]string{},
					MetricsServer:    map[string]string{},
					AdmissionWebhook: map[string]string{},
				},
			},
		}
		s := &systemState{instance: instance}

		gotFn, gotResult, err := sFnBootstrappeValidation(context.Background(), nil, s)

		require.NoError(t, err)
		require.Nil(t, gotResult, "result should be nil")
		requireEqualFunc(t,
			gotFn,
			sFnUpdateKedaDeployment,
		)
	})

	t.Run("restricted annotations stop with error and update status", func(t *testing.T) {
		instance := v1alpha1.Keda{
			Spec: v1alpha1.KedaSpec{
				PodAnnotations: &v1alpha1.PodAnnotations{
					Operator:         map[string]string{v1alpha1.KymaBootstraperRegistryUrlMutation: "chleb"},
					MetricsServer:    map[string]string{v1alpha1.KymaBootstraperAddImagePullSecretMutation: "true"},
					AdmissionWebhook: map[string]string{v1alpha1.KymaBootstraperAddImagePullSecretMutation: "true"},
				},
			},
		}
		s := &systemState{instance: instance}

		gotFn, gotResult, err := sFnBootstrappeValidation(context.Background(), nil, s)

		require.NoError(t, err)
		require.Nil(t, gotResult, "result should be nil")
		requireEqualFunc(t, gotFn, sFnUpdateStatus(nil, err))
		require.Equal(t, v1alpha1.StateError, s.instance.Status.State,
			"expected instance state to be error")
	})
}
