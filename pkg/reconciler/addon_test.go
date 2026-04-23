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

func TestOverrideNamespace(t *testing.T) {
	t.Run("sets namespace on namespaced resources", func(t *testing.T) {
		objs := []unstructured.Unstructured{{Object: map[string]interface{}{
			"apiVersion": "v1", "kind": "Service",
			"metadata": map[string]interface{}{"name": "svc", "namespace": "old-ns"},
		}}}
		overrideNamespace(objs, "new-ns")
		require.Equal(t, "new-ns", objs[0].GetNamespace())
	})
	t.Run("skips cluster-scoped resources", func(t *testing.T) {
		objs := []unstructured.Unstructured{{Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRole",
			"metadata": map[string]interface{}{"name": "cr"},
		}}}
		overrideNamespace(objs, "new-ns")
		require.Empty(t, objs[0].GetNamespace())
	})
	t.Run("patches ClusterRoleBinding subjects", func(t *testing.T) {
		objs := []unstructured.Unstructured{{Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRoleBinding",
			"metadata": map[string]interface{}{"name": "crb"},
			"subjects": []interface{}{
				map[string]interface{}{"kind": "ServiceAccount", "name": "sa", "namespace": "old-ns"},
			},
		}}}
		overrideNamespace(objs, "new-ns")
		subjects, _, _ := unstructured.NestedSlice(objs[0].Object, "subjects")
		require.Equal(t, "new-ns", subjects[0].(map[string]interface{})["namespace"])
	})
	t.Run("patches Deployment env vars and istio annotation", func(t *testing.T) {
		objs := []unstructured.Unstructured{{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep", "namespace": "old-ns"},
			"spec": map[string]interface{}{"template": map[string]interface{}{
				"metadata": map[string]interface{}{},
				"spec": map[string]interface{}{"containers": []interface{}{
					map[string]interface{}{"name": "c1", "env": []interface{}{
						map[string]interface{}{"name": "KEDA_HTTP_OPERATOR_NAMESPACE", "value": "old-ns"},
						map[string]interface{}{"name": "OTHER", "value": "keep"},
					}},
				}},
			}},
		}}}
		overrideNamespace(objs, "new-ns")

		containers, _, _ := unstructured.NestedSlice(objs[0].Object, "spec", "template", "spec", "containers")
		envList := containers[0].(map[string]interface{})["env"].([]interface{})
		require.Equal(t, "new-ns", envList[0].(map[string]interface{})["value"])
		require.Equal(t, "keep", envList[1].(map[string]interface{})["value"])

		ann, _, _ := unstructured.NestedStringMap(objs[0].Object, "spec", "template", "metadata", "annotations")
		require.Equal(t, "9090", ann[istioExcludeInboundPortsAnnotation])
	})
}

func TestPatchDeploymentIstioAnnotation(t *testing.T) {
	t.Run("adds annotation when missing", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec":     map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{}}},
		}}
		patchDeploymentIstioAnnotation(obj)
		ann, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
		require.Equal(t, istioExcludeInboundPortsValue, ann[istioExcludeInboundPortsAnnotation])
	})
	t.Run("no-op when already set", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec": map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{
				"annotations": map[string]interface{}{istioExcludeInboundPortsAnnotation: istioExcludeInboundPortsValue},
			}}},
		}}
		patchDeploymentIstioAnnotation(obj)
		ann, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
		require.Equal(t, istioExcludeInboundPortsValue, ann[istioExcludeInboundPortsAnnotation])
	})
}

func TestPatchSubjectsNamespace(t *testing.T) {
	t.Run("updates ServiceAccount namespace", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRoleBinding",
			"metadata": map[string]interface{}{"name": "crb"},
			"subjects": []interface{}{
				map[string]interface{}{"kind": "ServiceAccount", "name": "sa", "namespace": "old"},
				map[string]interface{}{"kind": "Group", "name": "grp", "namespace": "old"},
			},
		}}
		patchSubjectsNamespace(obj, "new")
		subjects, _, _ := unstructured.NestedSlice(obj.Object, "subjects")
		require.Equal(t, "new", subjects[0].(map[string]interface{})["namespace"])
		require.Equal(t, "old", subjects[1].(map[string]interface{})["namespace"])
	})
	t.Run("no-op when namespace already matches", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "RoleBinding",
			"metadata": map[string]interface{}{"name": "rb"},
			"subjects": []interface{}{
				map[string]interface{}{"kind": "ServiceAccount", "name": "sa", "namespace": "same"},
			},
		}}
		patchSubjectsNamespace(obj, "same")
		subjects, _, _ := unstructured.NestedSlice(obj.Object, "subjects")
		require.Equal(t, "same", subjects[0].(map[string]interface{})["namespace"])
	})
	t.Run("no-op when no subjects", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRoleBinding",
			"metadata": map[string]interface{}{"name": "crb"},
		}}
		patchSubjectsNamespace(obj, "ns")
	})
}

func TestPatchDeploymentEnvNamespace(t *testing.T) {
	t.Run("overrides matching env vars", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec": map[string]interface{}{"template": map[string]interface{}{"spec": map[string]interface{}{
				"containers": []interface{}{map[string]interface{}{
					"name": "c1", "env": []interface{}{
						map[string]interface{}{"name": "KEDA_HTTP_SCALER_TARGET_ADMIN_NAMESPACE", "value": "old"},
						map[string]interface{}{"name": "KEDA_HTTP_OPERATOR_NAMESPACE", "value": "old"},
						map[string]interface{}{"name": "UNRELATED", "value": "keep"},
					},
				}},
			}}},
		}}
		patchDeploymentEnvNamespace(obj, "new-ns")
		containers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
		envList := containers[0].(map[string]interface{})["env"].([]interface{})
		require.Equal(t, "new-ns", envList[0].(map[string]interface{})["value"])
		require.Equal(t, "new-ns", envList[1].(map[string]interface{})["value"])
		require.Equal(t, "keep", envList[2].(map[string]interface{})["value"])
	})
	t.Run("no-op when no containers", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec":     map[string]interface{}{"template": map[string]interface{}{"spec": map[string]interface{}{}}},
		}}
		patchDeploymentEnvNamespace(obj, "ns")
	})
}

func TestSFnHandleAddon(t *testing.T) {
	t.Run("disabled addon switches to delete", func(t *testing.T) {
		s := &systemState{instance: v1alpha1.Keda{ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{annotationAddonEnabled: "false"},
		}}}
		fn, result, err := sFnHandleAddon(context.TODO(), nil, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, fn)
	})
	t.Run("enabled without version switches to resolve", func(t *testing.T) {
		s := &systemState{instance: v1alpha1.Keda{ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{annotationAddonEnabled: "true"},
		}}}
		fn, result, err := sFnHandleAddon(context.TODO(), nil, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, fn)
	})
	t.Run("invalid version sets error condition", func(t *testing.T) {
		s := &systemState{instance: v1alpha1.Keda{ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotationAddonEnabled: "true",
				annotationAddonVersion: "not-a-semver",
			},
		}}}
		_, _, err := sFnHandleAddon(context.TODO(), nil, s)
		require.NoError(t, err)
		require.NotEmpty(t, s.instance.Status.Conditions)
	})
	t.Run("valid version switches to apply", func(t *testing.T) {
		s := &systemState{instance: v1alpha1.Keda{ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotationAddonEnabled: "true",
				annotationAddonVersion: "0.13.0",
			},
		}}}
		fn, result, err := sFnHandleAddon(context.TODO(), nil, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, fn)
		require.Equal(t, "0.13.0", s.instance.GetAnnotations()[annotationAddonVersion])
	})
}
