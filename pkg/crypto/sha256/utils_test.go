package sha256

import (
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var errTest = errors.New("test error")

func Test_calculateSHA256(t *testing.T) {
	type args struct {
		obj unstructured.Unstructured
	}
	tests := []struct {
		name        string
		args        args
		hashBuilder WriterSumerBuilder
		want        string
		wantErr     bool
	}{
		{
			name:        "empty",
			hashBuilder: DefaultWriterSumerBuilder,
			args: args{
				obj: unstructured.Unstructured{},
			},
			want: "cVRoVdYnnvcNIJCbKSxCwtywLNBr3gFIXaUtE-ME6_Q=",
		},
		{
			name:        "no-empty",
			hashBuilder: DefaultWriterSumerBuilder,
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
			got, err := tt.hashBuilder.CalculateSHA256(tt.args.obj)
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
