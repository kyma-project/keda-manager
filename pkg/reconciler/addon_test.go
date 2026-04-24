package reconciler

import (
	"context"
	"testing"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestReadAddonCfg(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		want        addonCfg
	}{
		{"nil annotations", nil, addonCfg{}},
		{"empty annotations", map[string]string{}, addonCfg{}},
		{
			"enabled with version and namespace",
			map[string]string{
				annotationAddonEnabled:   "true",
				annotationAddonVersion:   "0.13.0",
				annotationAddonNamespace: "custom-ns",
			},
			addonCfg{enabled: true, version: "0.13.0", namespace: "custom-ns"},
		},
		{
			"enabled case insensitive",
			map[string]string{annotationAddonEnabled: "True"},
			addonCfg{enabled: true},
		},
		{
			"disabled explicitly",
			map[string]string{annotationAddonEnabled: "false"},
			addonCfg{enabled: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &v1alpha1.Keda{ObjectMeta: metav1.ObjectMeta{Annotations: tt.annotations}}
			require.Equal(t, tt.want, readAddonCfg(instance))
		})
	}
}

func TestEffectiveNamespace(t *testing.T) {
	t.Run("returns default when empty", func(t *testing.T) {
		require.Equal(t, defaultAddonNamespace, addonCfg{}.effectiveNamespace())
	})
	t.Run("returns custom namespace", func(t *testing.T) {
		require.Equal(t, "my-ns", addonCfg{namespace: "my-ns"}.effectiveNamespace())
	})
}

func TestSetAnnotation(t *testing.T) {
	t.Run("set on nil annotations", func(t *testing.T) {
		instance := &v1alpha1.Keda{}
		setAnnotation(instance, "key", "value")
		require.Equal(t, "value", instance.GetAnnotations()["key"])
	})
	t.Run("update existing", func(t *testing.T) {
		instance := &v1alpha1.Keda{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"key": "old"}}}
		setAnnotation(instance, "key", "new")
		require.Equal(t, "new", instance.GetAnnotations()["key"])
	})
	t.Run("delete when empty value", func(t *testing.T) {
		instance := &v1alpha1.Keda{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"key": "val"}}}
		setAnnotation(instance, "key", "")
		_, exists := instance.GetAnnotations()["key"]
		require.False(t, exists)
	})
}
