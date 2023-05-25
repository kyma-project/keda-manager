package reconciler

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_sFnServedFilter(t *testing.T) {
	t.Run("skip processing when served is false", func(t *testing.T) {
		s := &systemState{
			instance: v1alpha1.Keda{
				Status: v1alpha1.Status{
					Served: v1alpha1.ServedFalse,
				},
			},
		}

		nextFn, result, err := sFnServedFilter(context.TODO(), nil, s)

		require.Nil(t, err)
		require.Nil(t, nextFn)
		require.Nil(t, result)
	})

	t.Run("do next step when served is true", func(t *testing.T) {
		s := &systemState{
			instance: v1alpha1.Keda{
				Status: v1alpha1.Status{
					Served: v1alpha1.ServedTrue,
				},
			},
		}

		nextFn, result, err := sFnServedFilter(context.TODO(), nil, s)

		require.Nil(t, err)
		requireEqualFunc(t, sFnTakeSnapshot, nextFn)
		require.Nil(t, result)
	})

	t.Run("set served value from nil to true when there is no served keda on cluster", func(t *testing.T) {
		s := &systemState{
			instance: v1alpha1.Keda{
				Status: v1alpha1.Status{},
			},
		}

		r := &fsm{
			K8s: K8s{
				Client: func() client.Client {
					scheme := apiruntime.NewScheme()
					require.NoError(t, v1alpha1.AddToScheme(scheme))

					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithRuntimeObjects(
							fixServedKeda("test-1", "default", ""),
							fixServedKeda("test-2", "keda-test", v1alpha1.ServedFalse),
							fixServedKeda("test-3", "keda-test-2", ""),
							fixServedKeda("test-4", "default", v1alpha1.ServedFalse),
						).Build()
				}(),
			},
		}

		nextFn, result, err := sFnServedFilter(context.TODO(), r, s)

		require.Nil(t, err)
		requireEqualFunc(t, sFnUpdateStatus(&ctrl.Result{Requeue: true}, nil), nextFn)
		require.Nil(t, result)

		require.Equal(t, v1alpha1.ServedTrue, s.instance.Status.Served)
	})

	t.Run("set served value from nil to false and set condition to error when there is at lease one served keda on cluster", func(t *testing.T) {
		s := &systemState{
			instance: v1alpha1.Keda{
				Status: v1alpha1.Status{},
			},
		}

		r := &fsm{
			K8s: K8s{
				Client: func() client.Client {
					scheme := apiruntime.NewScheme()
					require.NoError(t, v1alpha1.AddToScheme(scheme))

					return fake.NewClientBuilder().
						WithScheme(scheme).
						WithRuntimeObjects(
							fixServedKeda("test-1", "default", v1alpha1.ServedFalse),
							fixServedKeda("test-2", "keda-test", v1alpha1.ServedTrue),
							fixServedKeda("test-3", "keda-test-2", ""),
							fixServedKeda("test-4", "default", v1alpha1.ServedFalse),
						).Build()
				}(),
			},
		}

		nextFn, result, err := sFnServedFilter(context.TODO(), r, s)

		require.Nil(t, err)
		requireEqualFunc(t, sFnUpdateStatus(&ctrl.Result{Requeue: true}, nil), nextFn)
		require.Nil(t, result)

		require.Equal(t, v1alpha1.StateError, s.instance.Status.State)
		require.Equal(t, v1alpha1.ServedFalse, s.instance.Status.Served)

		expectedCondition := metav1.Condition{
			Type:    string(v1alpha1.ConditionTypeInstalled),
			Status:  "False",
			Reason:  string(v1alpha1.ConditionReasonKedaDuplicated),
			Message: "only one instance of Keda is allowed (current served instance: keda-test/test-2)",
		}
		opt := cmp.Comparer(func(x, y metav1.Condition) bool {
			return x.Type == y.Type && x.Status == y.Status && x.Reason == y.Reason && x.Message == y.Message
		})
		g := gomega.NewWithT(t)
		g.Expect(s.instance.Status.Conditions).Should(gomega.ContainElement(gomega.BeComparableTo(expectedCondition, opt)))
	})
}

func fixServedKeda(name, namespace string, served string) *v1alpha1.Keda {
	return &v1alpha1.Keda{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: v1alpha1.Status{
			Served: served,
		},
	}
}

func requireEqualFunc(t *testing.T, expected, actual stateFn) {
	expectedValueOf := reflect.ValueOf(expected)
	actualValueOf := reflect.ValueOf(actual)
	require.True(t, expectedValueOf.Pointer() == actualValueOf.Pointer(),
		fmt.Sprintf("expected '%s', got '%s", getFnName(expected), getFnName(actual)))
}

func getFnName(fn stateFn) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}
