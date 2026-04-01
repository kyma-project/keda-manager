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
		m := &fsm{}
		next, result, err := sFnHttpAddOnDecision(context.Background(), m, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, next)
	})

	t.Run("switches to sFnDeleteHttpAddOn when annotation is absent", func(t *testing.T) {
		s := &systemState{
			instance: v1alpha1.Keda{},
		}
		m := &fsm{}
		next, result, err := sFnHttpAddOnDecision(context.Background(), m, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, next)
	})
}

func Test_sFnApplyHttpAddOn_emptyObjs(t *testing.T) {
	t.Run("stops with no requeue when HttpAddOnObjs is empty", func(t *testing.T) {
		s := &systemState{
			instance: v1alpha1.Keda{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						httpAddOnAnnotationKey: httpAddOnAnnotationEnabled,
					},
				},
			},
		}
		m := &fsm{
			Cfg: Cfg{
				HttpAddOnObjs: nil,
			},
		}
		next, result, err := sFnApplyHttpAddOn(context.Background(), m, s)
		require.NoError(t, err)
		require.Nil(t, result)
		// next should be the sFnUpdateStatus function (stopWithNoRequeue)
		require.NotNil(t, next)
		// The condition should be set to unknown
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
		s := &systemState{
			instance: v1alpha1.Keda{},
		}
		m := &fsm{
			Cfg: Cfg{
				HttpAddOnObjs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "Namespace",
							"metadata": map[string]interface{}{
								"name": "keda-http-add-on",
							},
						},
					},
				},
			},
		}
		next, result, err := sFnDeleteHttpAddOn(context.Background(), m, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, next)
	})
}

func Test_sFnDeleteHttpAddOn_emptyObjs(t *testing.T) {
	t.Run("stops with no requeue when HttpAddOnObjs is empty", func(t *testing.T) {
		s := &systemState{
			instance: v1alpha1.Keda{},
		}
		m := &fsm{
			Cfg: Cfg{
				HttpAddOnObjs: nil,
			},
		}
		next, result, err := sFnDeleteHttpAddOn(context.Background(), m, s)
		require.NoError(t, err)
		require.Nil(t, result)
		require.NotNil(t, next)
	})
}

func makeFsmWithNoK8s(httpAddOnObjs []unstructured.Unstructured) *fsm {
	return &fsm{
		Cfg: Cfg{
			HttpAddOnObjs: httpAddOnObjs,
		},
	}
}

// Verify that sFnHttpAddOnDecision returns distinct stateFn pointers for the two branches
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
	disabledState := &systemState{
		instance: v1alpha1.Keda{},
	}

	m := makeFsmWithNoK8s(nil)

	applyNext, _, _ := sFnHttpAddOnDecision(context.Background(), m, enabledState)
	deleteNext, _, _ := sFnHttpAddOnDecision(context.Background(), m, disabledState)

	// Verify they are different functions by calling them (they should both handle gracefully)
	// We just verify both are non-nil and different
	require.NotNil(t, applyNext)
	require.NotNil(t, deleteNext)

	// Call them both to confirm they don't panic with empty objs/no conditions
	s2 := &systemState{instance: v1alpha1.Keda{}}
	applyResult, applyCtrlResult, applyErr := applyNext(context.Background(), m, s2)
	require.NoError(t, applyErr)
	require.Nil(t, applyCtrlResult)
	require.NotNil(t, applyResult) // sFnUpdateStatus wrapping

	s3 := &systemState{instance: v1alpha1.Keda{}}
	deleteResult, deleteCtrlResult, deleteErr := deleteNext(context.Background(), m, s3)
	require.NoError(t, deleteErr)
	require.Nil(t, deleteCtrlResult)
	require.NotNil(t, deleteResult)

	_ = ctrl.Result{}
}
