package reconciler

import (
	"context"
	"testing"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
)

func httpAddOnObj() unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": "keda-http-add-on",
				"labels": map[string]interface{}{
					httpAddOnComponentLabel: httpAddOnComponentValue,
				},
			},
		},
	}
}

func Test_httpAddOnEnabled(t *testing.T) {
	t.Run("returns true when annotation is present and enabled", func(t *testing.T) {
		instance := &v1alpha1.Keda{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					httpAddOnAnnotationKey: httpAddOnAnnotationEnabled,
				},
			},
		}
		require.True(t, httpAddOnEnabled(instance))
	})

	t.Run("returns false when annotation is absent", func(t *testing.T) {
		instance := &v1alpha1.Keda{}
		require.False(t, httpAddOnEnabled(instance))
	})

	t.Run("returns false when annotation has a different value", func(t *testing.T) {
		instance := &v1alpha1.Keda{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					httpAddOnAnnotationKey: "disabled",
				},
			},
		}
		require.False(t, httpAddOnEnabled(instance))
	})

	t.Run("returns false when annotations map is nil", func(t *testing.T) {
		instance := &v1alpha1.Keda{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: nil,
			},
		}
		require.False(t, httpAddOnEnabled(instance))
	})
}

func Test_httpAddOnObjs(t *testing.T) {
	t.Run("returns only objects with the http-add-on component label", func(t *testing.T) {
		addOn := httpAddOnObj()
		other := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name": "keda-operator",
				},
			},
		}
		result := httpAddOnObjs([]unstructured.Unstructured{addOn, other})
		require.Len(t, result, 1)
		require.Equal(t, "keda-http-add-on", result[0].GetName())
	})

	t.Run("returns empty slice when no objects match", func(t *testing.T) {
		result := httpAddOnObjs([]unstructured.Unstructured{})
		require.Empty(t, result)
	})
}

func Test_sFnHttpAddOnDecision(t *testing.T) {
	t.Run("switches to sFnApplyHttpAddOn when annotation is enabled", func(t *testing.T) {
		s := &systemState{
			instance: v1alpha1.Keda{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						httpAddOnAnnotationKey: httpAddOnAnnotationEnabled,
					},
				},
			},
		}
		next, result, err := sFnHttpAddOnDecision(context.Background(), &fsm{}, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, next)
	})

	t.Run("switches to sFnDeleteHttpAddOn when annotation is absent", func(t *testing.T) {
		s := &systemState{instance: v1alpha1.Keda{}}
		next, result, err := sFnHttpAddOnDecision(context.Background(), &fsm{}, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, next)
	})
}

func Test_sFnApplyHttpAddOn_emptyObjs(t *testing.T) {
	t.Run("sets unknown condition when no http-add-on objects in Objs", func(t *testing.T) {
		s := &systemState{
			instance: v1alpha1.Keda{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						httpAddOnAnnotationKey: httpAddOnAnnotationEnabled,
					},
				},
			},
		}
		m := &fsm{Cfg: Cfg{Objs: nil}}
		next, result, err := sFnApplyHttpAddOn(context.Background(), m, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, next)

		found := false
		for _, c := range s.instance.Status.Conditions {
			if c.Type == string(v1alpha1.ConditionTypeHttpAddOnInstalled) {
				found = true
				require.Equal(t, metav1.ConditionUnknown, c.Status)
			}
		}
		require.True(t, found, "expected HttpAddOnInstalled condition to be set")
	})
}

func Test_sFnDeleteHttpAddOn_noCondition(t *testing.T) {
	t.Run("stops with no requeue when HttpAddOnInstalled condition is not present", func(t *testing.T) {
		s := &systemState{instance: v1alpha1.Keda{}}
		m := &fsm{Cfg: Cfg{Objs: []unstructured.Unstructured{httpAddOnObj()}}}
		next, result, err := sFnDeleteHttpAddOn(context.Background(), m, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, next)
	})
}

func Test_sFnDeleteHttpAddOn_emptyObjs(t *testing.T) {
	t.Run("stops with no requeue when no http-add-on objects in Objs", func(t *testing.T) {
		s := &systemState{instance: v1alpha1.Keda{}}
		m := &fsm{Cfg: Cfg{Objs: nil}}
		next, result, err := sFnDeleteHttpAddOn(context.Background(), m, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, next)
	})
}

func Test_sFnHttpAddOnDecision_branches(t *testing.T) {
	enabledState := &systemState{
		instance: v1alpha1.Keda{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					httpAddOnAnnotationKey: httpAddOnAnnotationEnabled,
				},
			},
		},
	}
	disabledState := &systemState{instance: v1alpha1.Keda{}}
	m := &fsm{Cfg: Cfg{Objs: nil}}

	applyNext, _, _ := sFnHttpAddOnDecision(context.Background(), m, enabledState)
	deleteNext, _, _ := sFnHttpAddOnDecision(context.Background(), m, disabledState)
	require.NotNil(t, applyNext)
	require.NotNil(t, deleteNext)

	s2 := &systemState{instance: v1alpha1.Keda{}}
	applyResult, applyCtrlResult, applyErr := applyNext(context.Background(), m, s2)
	require.NoError(t, applyErr)
	require.Nil(t, applyCtrlResult)
	require.NotNil(t, applyResult)

	s3 := &systemState{instance: v1alpha1.Keda{}}
	deleteResult, deleteCtrlResult, deleteErr := deleteNext(context.Background(), m, s3)
	require.NoError(t, deleteErr)
	require.Nil(t, deleteCtrlResult)
	require.NotNil(t, deleteResult)

	_ = ctrl.Result{}
}
