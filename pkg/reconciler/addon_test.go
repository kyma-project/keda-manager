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
		want        v1alpha1.AddonCfg
	}{
		{"nil annotations", nil, v1alpha1.AddonCfg{}},
		{"empty annotations", map[string]string{}, v1alpha1.AddonCfg{}},
		{
			"enabled with namespace",
			map[string]string{
				v1alpha1.AnnotationAddonEnabled:   "true",
				v1alpha1.AnnotationAddonNamespace: "custom-ns",
			},
			v1alpha1.AddonCfg{Enabled: true, Namespace: "custom-ns"},
		},
		{
			"enabled case insensitive",
			map[string]string{v1alpha1.AnnotationAddonEnabled: "True"},
			v1alpha1.AddonCfg{Enabled: true},
		},
		{
			"disabled explicitly",
			map[string]string{v1alpha1.AnnotationAddonEnabled: "false"},
			v1alpha1.AddonCfg{Enabled: false},
		},
		{
			"istio injection enabled",
			map[string]string{
				v1alpha1.AnnotationAddonEnabled:        "true",
				v1alpha1.AnnotationAddonIstioInjection: "true",
			},
			v1alpha1.AddonCfg{Enabled: true, IstioInjection: true},
		},
		{
			"istio injection disabled explicitly",
			map[string]string{
				v1alpha1.AnnotationAddonEnabled:        "true",
				v1alpha1.AnnotationAddonIstioInjection: "false",
			},
			v1alpha1.AddonCfg{Enabled: true, IstioInjection: false},
		},
		{
			"istio injection absent defaults to false",
			map[string]string{
				v1alpha1.AnnotationAddonEnabled: "true",
			},
			v1alpha1.AddonCfg{Enabled: true, IstioInjection: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &v1alpha1.Keda{ObjectMeta: metav1.ObjectMeta{Annotations: tt.annotations}}
			require.Equal(t, tt.want, v1alpha1.ReadAddonCfg(instance))
		})
	}
}

func TestEffectiveNamespace(t *testing.T) {
	t.Run("returns default when empty", func(t *testing.T) {
		require.Equal(t, v1alpha1.DefaultAddonNamespace, v1alpha1.AddonCfg{}.EffectiveNamespace())
	})
	t.Run("returns custom namespace", func(t *testing.T) {
		require.Equal(t, "my-ns", v1alpha1.AddonCfg{Namespace: "my-ns"}.EffectiveNamespace())
	})
}

func TestSetAnnotation(t *testing.T) {
	t.Run("set on nil annotations", func(t *testing.T) {
		instance := &v1alpha1.Keda{}
		v1alpha1.SetAnnotation(instance, "key", "value")
		require.Equal(t, "value", instance.GetAnnotations()["key"])
	})
	t.Run("update existing", func(t *testing.T) {
		instance := &v1alpha1.Keda{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"key": "old"}}}
		v1alpha1.SetAnnotation(instance, "key", "new")
		require.Equal(t, "new", instance.GetAnnotations()["key"])
	})
	t.Run("delete when empty value", func(t *testing.T) {
		instance := &v1alpha1.Keda{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"key": "val"}}}
		v1alpha1.SetAnnotation(instance, "key", "")
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
		overrideNamespace(objs, "new-ns", false)
		require.Equal(t, "new-ns", objs[0].GetNamespace())
		// Top-level Service metadata gets the standard Kyma module labels.
		labels := objs[0].GetLabels()
		require.Equal(t, kymaModuleLabelValue, labels[kymaModuleLabel])
		require.Equal(t, "keda-manager", labels["app.kubernetes.io/part-of"])
		require.Equal(t, "keda-manager", labels["app.kubernetes.io/managed-by"])
	})
	t.Run("skips cluster-scoped resources", func(t *testing.T) {
		objs := []unstructured.Unstructured{{Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRole",
			"metadata": map[string]interface{}{"name": "cr"},
		}}}
		overrideNamespace(objs, "new-ns", false)
		require.Empty(t, objs[0].GetNamespace())
		// Cluster-scoped resources still get the standard module labels.
		require.Equal(t, kymaModuleLabelValue, objs[0].GetLabels()[kymaModuleLabel])
	})
	t.Run("patches ClusterRoleBinding subjects", func(t *testing.T) {
		objs := []unstructured.Unstructured{{Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRoleBinding",
			"metadata": map[string]interface{}{"name": "crb"},
			"subjects": []interface{}{
				map[string]interface{}{"kind": "ServiceAccount", "name": "sa", "namespace": "old-ns"},
			},
		}}}
		overrideNamespace(objs, "new-ns", false)
		subjects, _, _ := unstructured.NestedSlice(objs[0].Object, "subjects")
		require.Equal(t, "new-ns", subjects[0].(map[string]interface{})["namespace"])
	})
	t.Run("patches Deployment env vars and istio when enabled", func(t *testing.T) {
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
		overrideNamespace(objs, "new-ns", true)

		containers, _, _ := unstructured.NestedSlice(objs[0].Object, "spec", "template", "spec", "containers")
		envList := containers[0].(map[string]interface{})["env"].([]interface{})
		require.Equal(t, "new-ns", envList[0].(map[string]interface{})["value"])
		require.Equal(t, "keep", envList[1].(map[string]interface{})["value"])

		ann, _, _ := unstructured.NestedStringMap(objs[0].Object, "spec", "template", "metadata", "annotations")
		require.Equal(t, "9090", ann[istioExcludeInboundPortsAnnotation])
		require.Equal(t, "true", ann[istioSidecarInjectAnnotation])

		// Top-level Deployment metadata gets the standard module labels.
		topLabels := objs[0].GetLabels()
		require.Equal(t, kymaModuleLabelValue, topLabels[kymaModuleLabel])
		require.Equal(t, "keda-manager", topLabels["app.kubernetes.io/part-of"])
		require.Equal(t, "keda-manager", topLabels["app.kubernetes.io/managed-by"])

		// Pod template gets only the Kyma module label — we must not overwrite
		// upstream selector keys like app.kubernetes.io/part-of on the template.
		podLabels, _, _ := unstructured.NestedStringMap(objs[0].Object, "spec", "template", "metadata", "labels")
		require.Equal(t, kymaModuleLabelValue, podLabels[kymaModuleLabel])
		require.NotContains(t, podLabels, "app.kubernetes.io/part-of")
		require.NotContains(t, podLabels, "app.kubernetes.io/managed-by")
	})
	t.Run("sets sidecar inject false annotation when istio disabled", func(t *testing.T) {
		objs := []unstructured.Unstructured{{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep", "namespace": "old-ns"},
			"spec": map[string]interface{}{"template": map[string]interface{}{
				"metadata": map[string]interface{}{},
				"spec": map[string]interface{}{"containers": []interface{}{
					map[string]interface{}{"name": "c1", "env": []interface{}{
						map[string]interface{}{"name": "KEDA_HTTP_OPERATOR_NAMESPACE", "value": "old-ns"},
					}},
				}},
			}},
		}}}
		overrideNamespace(objs, "new-ns", false)

		containers, _, _ := unstructured.NestedSlice(objs[0].Object, "spec", "template", "spec", "containers")
		envList := containers[0].(map[string]interface{})["env"].([]interface{})
		require.Equal(t, "new-ns", envList[0].(map[string]interface{})["value"])

		ann, _, _ := unstructured.NestedStringMap(objs[0].Object, "spec", "template", "metadata", "annotations")
		require.Empty(t, ann[istioExcludeInboundPortsAnnotation])
		require.Equal(t, "false", ann[istioSidecarInjectAnnotation])

		// Module label lands on top-level (full set) and on pod template (only kyma-project.io/module).
		require.Equal(t, kymaModuleLabelValue, objs[0].GetLabels()[kymaModuleLabel])
		podLabels, _, _ := unstructured.NestedStringMap(objs[0].Object, "spec", "template", "metadata", "labels")
		require.Equal(t, kymaModuleLabelValue, podLabels[kymaModuleLabel])
		require.NotContains(t, podLabels, "app.kubernetes.io/part-of")
		require.NotContains(t, podLabels, "app.kubernetes.io/managed-by")
	})
}

func TestPatchDeploymentIstioExcludePortsAnnotation(t *testing.T) {
	t.Run("adds annotation when missing", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec":     map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{}}},
		}}
		patchDeploymentIstioExcludePortsAnnotation(obj)
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
		patchDeploymentIstioExcludePortsAnnotation(obj)
		ann, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
		require.Equal(t, istioExcludeInboundPortsValue, ann[istioExcludeInboundPortsAnnotation])
	})
}

func TestPatchDeploymentIstioSidecarAnnotation(t *testing.T) {
	t.Run("adds true annotation when missing", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec":     map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{}}},
		}}
		patchDeploymentIstioSidecarAnnotation(obj, "true")
		ann, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
		require.Equal(t, "true", ann[istioSidecarInjectAnnotation])
	})
	t.Run("adds false annotation when missing", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec":     map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{}}},
		}}
		patchDeploymentIstioSidecarAnnotation(obj, "false")
		ann, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
		require.Equal(t, "false", ann[istioSidecarInjectAnnotation])
	})
	t.Run("no-op when already set to same value", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec": map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{
				"annotations": map[string]interface{}{istioSidecarInjectAnnotation: "true"},
			}}},
		}}
		patchDeploymentIstioSidecarAnnotation(obj, "true")
		ann, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
		require.Equal(t, "true", ann[istioSidecarInjectAnnotation])
	})
	t.Run("overwrites true with false", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec": map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{
				"annotations": map[string]interface{}{istioSidecarInjectAnnotation: "true"},
			}}},
		}}
		patchDeploymentIstioSidecarAnnotation(obj, "false")
		ann, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
		require.Equal(t, "false", ann[istioSidecarInjectAnnotation])
	})
	t.Run("preserves existing annotations", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec": map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{
				"annotations": map[string]interface{}{"existing": "value"},
			}}},
		}}
		patchDeploymentIstioSidecarAnnotation(obj, "false")
		ann, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
		require.Equal(t, "false", ann[istioSidecarInjectAnnotation])
		require.Equal(t, "value", ann["existing"])
	})
}

func TestPatchDeploymentPodTemplateLabels(t *testing.T) {
	t.Run("adds kyma module label when missing", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec":     map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{}}},
		}}
		patchDeploymentPodTemplateLabels(obj)
		labels, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "labels")
		require.Equal(t, kymaModuleLabelValue, labels[kymaModuleLabel])
		// Pod template MUST NOT receive the part-of/managed-by labels — those
		// would overwrite immutable selector keys on upstream Deployments.
		require.NotContains(t, labels, "app.kubernetes.io/part-of")
		require.NotContains(t, labels, "app.kubernetes.io/managed-by")
	})
	t.Run("no-op when already set", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec": map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{
				"labels": map[string]interface{}{
					kymaModuleLabel: kymaModuleLabelValue,
				},
			}}},
		}}
		patchDeploymentPodTemplateLabels(obj)
		labels, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "labels")
		require.Equal(t, kymaModuleLabelValue, labels[kymaModuleLabel])
	})
	t.Run("preserves existing labels and does not touch part-of", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "dep"},
			"spec": map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{
				"labels": map[string]interface{}{
					"app":                       "interceptor",
					"app.kubernetes.io/part-of": "keda",
				},
			}}},
		}}
		patchDeploymentPodTemplateLabels(obj)
		labels, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "labels")
		require.Equal(t, kymaModuleLabelValue, labels[kymaModuleLabel])
		require.Equal(t, "interceptor", labels["app"])
		// Critically, part-of stays at the upstream value, NOT "keda-manager".
		require.Equal(t, "keda", labels["app.kubernetes.io/part-of"])
	})
	t.Run("template.labels stays consistent with upstream selector.matchLabels", func(t *testing.T) {
		// Regression for the http-add-on Deployment: spec.selector.matchLabels
		// is immutable and includes app.kubernetes.io/part-of: keda. The
		// API server rejects the apply if our patch changes template.labels
		// in a way that breaks selector == template invariant.
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]interface{}{"name": "keda-add-ons-http-operator"},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app.kubernetes.io/component": "add-on",
						"app.kubernetes.io/instance":  "operator",
						"app.kubernetes.io/name":      "http",
						"app.kubernetes.io/part-of":   "keda",
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app.kubernetes.io/component": "add-on",
							"app.kubernetes.io/instance":  "operator",
							"app.kubernetes.io/name":      "http",
							"app.kubernetes.io/part-of":   "keda",
						},
					},
				},
			},
		}}
		patchDeploymentPodTemplateLabels(obj)

		selector, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "selector", "matchLabels")
		template, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "labels")
		for k, v := range selector {
			require.Equal(t, v, template[k],
				"template.labels[%q] must equal selector.matchLabels[%q]; the API server otherwise rejects the Deployment", k, k)
		}
		// And we did add our marker.
		require.Equal(t, kymaModuleLabelValue, template[kymaModuleLabel])
	})
}

func TestApplyCommonMetadataLabels(t *testing.T) {
	t.Run("adds labels when metadata has none", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1", "kind": "Service",
			"metadata": map[string]interface{}{"name": "svc"},
		}}
		applyCommonMetadataLabels(obj)
		labels := obj.GetLabels()
		require.Equal(t, kymaModuleLabelValue, labels[kymaModuleLabel])
		require.Equal(t, "keda-manager", labels["app.kubernetes.io/part-of"])
		require.Equal(t, "keda-manager", labels["app.kubernetes.io/managed-by"])
	})
	t.Run("merges with existing labels", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1", "kind": "Service",
			"metadata": map[string]interface{}{
				"name":   "svc",
				"labels": map[string]interface{}{"app": "interceptor"},
			},
		}}
		applyCommonMetadataLabels(obj)
		labels := obj.GetLabels()
		require.Equal(t, kymaModuleLabelValue, labels[kymaModuleLabel])
		require.Equal(t, "interceptor", labels["app"])
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
			Annotations: map[string]string{v1alpha1.AnnotationAddonEnabled: "false"},
		}}}
		fn, result, err := sFnHandleAddon(context.TODO(), nil, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, fn)
	})
	t.Run("enabled addon switches to apply", func(t *testing.T) {
		s := &systemState{instance: v1alpha1.Keda{ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{v1alpha1.AnnotationAddonEnabled: "true"},
		}}}
		fn, result, err := sFnHandleAddon(context.TODO(), nil, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, fn)
	})
}
