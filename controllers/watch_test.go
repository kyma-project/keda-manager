package controllers

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Test_registerWatchDistinct(t *testing.T) {
	var count int

	type args struct {
		objs          []unstructured.Unstructured
		registerWatch func(unstructured.Unstructured)
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "OK",
			args: args{
				registerWatch: func() func(unstructured.Unstructured) {
					return func(_ unstructured.Unstructured) {
						count++
					}
				}(),
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"kind":       "test",
							"apiVersion": "operator.kyma-project.io/v1alpha1",
						},
					},
					{
						Object: map[string]interface{}{
							"kind":       "test2",
							"apiVersion": "operator.kyma-project.io/v1alpha1",
						},
					},

					{
						Object: map[string]interface{}{
							"kind":       "test",
							"apiVersion": "operator.kyma-project.io/v1alpha1",
						},
					},
					{
						Object: map[string]interface{}{
							"kind":       "test2",
							"apiVersion": "operator.kyma-project.io/v1alpha1",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := registerWatchDistinct(tt.args.objs, tt.args.registerWatch); (err != nil) != tt.wantErr {
				t.Errorf("registerWatchDistinct() error = %v, wantErr %v", err, tt.wantErr)
			}
			wantCount := 2
			if count != wantCount {
				t.Errorf("registerWatchDistinct() count = %d, wantCount = %d", count, wantCount)
			}
		})
	}
}
