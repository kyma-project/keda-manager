package sha256_test

import (
	"errors"
	"testing"

	"github.com/kyma-project/keda-manager/pkg/crypto/sha256"
	sha256mock "github.com/kyma-project/keda-manager/pkg/crypto/sha256/automock"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	errTest = errors.New("test error")
)

func Test_calculateSHA256(t *testing.T) {
	type args struct {
		obj unstructured.Unstructured
	}
	tests := []struct {
		name        string
		args        args
		hashBuilder sha256.Calculator
		want        string
		wantErr     bool
	}{
		{
			name: "write error",
			hashBuilder: func() sha256.Calculator {
				ws := sha256mock.NewWriterSumer(t)
				ws.On("Write", mock.AnythingOfType("[]uint8")).Return(0, errTest).Once()
				return func() sha256.WriterSumer {
					return ws
				}
			}(),
			wantErr: true,
		},
		{
			name:        "empty",
			hashBuilder: sha256.DefaultCalculator,
			args: args{
				obj: unstructured.Unstructured{},
			},
			want: "cVRoVdYnnvcNIJCbKSxCwtywLNBr3gFIXaUtE-ME6_Q=",
		},
		{
			name:        "no-empty",
			hashBuilder: sha256.DefaultCalculator,
			args: args{
				obj: func() unstructured.Unstructured {
					var u unstructured.Unstructured
					u.SetGroupVersionKind(schema.GroupVersionKind{
						Kind:    "CustomResourceDefinition",
						Group:   "apiextensions.k8s.io",
						Version: "v1",
					})

					return u
				}(),
			},
			want: "9NtR-1kpz4ub0a8jS4YySJEGZKmPfvC5FLh5GNW5UlA=",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.hashBuilder.CalculateSum(tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("calculateSHA256() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("calculateSHA256() = %v, want %v", got, tt.want)
			}
		})
	}
}
