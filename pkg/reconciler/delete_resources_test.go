package reconciler

import (
	"context"
	"reflect"
	"runtime"
	"testing"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	testDeployment = unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "Deployment",
			"apiVersion": "apps/v1",
			"metadata": map[string]interface{}{
				"name":      "test-deployment",
				"namespace": "test",
			},
		},
	}

	testEmptyCRD = unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "CustomResourceDefinition",
			"apiVersion": "apiextensions.k8s.io/v1",
			"metadata": map[string]interface{}{
				"name":      "test-empty-crd",
				"namespace": "test",
			},
		},
	}

	testService = unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "Service",
			"apiVersion": "v1",
			"metadata": map[string]interface{}{
				"name":      "test-service",
				"namespace": "test",
			},
		},
	}

	testCRD = unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "CustomResourceDefinition",
			"apiVersion": "apiextensions.k8s.io/v1",
			"metadata": map[string]interface{}{
				"name":      "test-crd",
				"namespace": "test",
			},
			"spec": map[string]interface{}{
				"group": "testgroup.io",
				"names": map[string]interface{}{
					"kind": "TestResource",
				},
				"versions": []interface{}{
					map[string]interface{}{
						"name":    "v1alphav1",
						"storage": false,
					},
					map[string]interface{}{
						"name":    "v1",
						"storage": true,
					},
				},
			},
		},
	}

	testCR = unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "TestResource",
			"apiVersion": "testgroup.io/v1",
			"metadata": map[string]interface{}{
				"name":      "test-crd",
				"namespace": "test",
			},
		},
	}
)

func Test_sFnDeleteResources(t *testing.T) {
	t.Run("update Keda CR condition first", func(t *testing.T) {
		system := systemState{
			instance: v1alpha1.Keda{
				Status: v1alpha1.Status{
					Conditions: []metav1.Condition{
						{
							Type:   string(v1alpha1.ConditionTypeInstalled),
							Reason: string(v1alpha1.ConditionReasonVerified),
						},
					},
				},
			},
		}

		stateFn, result, err := sFnDeleteResources(context.Background(), &fsm{}, &system)
		require.NoError(t, err)
		require.Nil(t, result)
		require.Equal(t, fnName(stateFn), fnName(sFnUpdateStatus(nil, nil)))
	})

	t.Run("choose right deletion strategy", func(t *testing.T) {
		system := systemState{
			instance: v1alpha1.Keda{
				Status: v1alpha1.Status{
					Conditions: []metav1.Condition{
						{
							Type:   string(v1alpha1.ConditionTypeInstalled),
							Reason: string(v1alpha1.ConditionReasonDeletion),
						},
					},
				},
			},
		}

		stateFn, result, err := sFnDeleteResources(context.Background(), &fsm{}, &system)
		require.NoError(t, err)
		require.Nil(t, result)
		require.Equal(t, fnName(stateFn), fnName(deletionStrategyBuilder(defaultDeletionStrategy)))
	})
}

func Test_sFnDeleteStrategy(t *testing.T) {
	t.Run("cascade delete strategy", func(t *testing.T) {
		clientBuilder := fake.NewClientBuilder().
			WithObjects(&testDeployment, &testEmptyCRD, &testService)
		client := clientBuilder.Build()
		ctx := context.Background()
		objs := []unstructured.Unstructured{
			testDeployment, testEmptyCRD, testService,
		}
		r := &fsm{
			log: zap.NewNop().Sugar(),
			K8s: K8s{Client: client},
			Cfg: Cfg{Objs: objs},
		}

		strategyFn := deletionStrategyBuilder(cascadeDeletionStrategy)

		fn, resp, err := strategyFn(ctx, r, &systemState{})
		require.Nil(t, resp)
		require.NoError(t, err)
		require.Equal(t, fnName(fn), fnName(sFnRemoveFinalizer))

		// check deletion progress
		require.False(t, canGetFakeResource(client, testDeployment))
		require.False(t, canGetFakeResource(client, testEmptyCRD))
		require.False(t, canGetFakeResource(client, testService))
	})

	t.Run("upstream delete strategy", func(t *testing.T) {
		clientBuilder := fake.NewClientBuilder().
			WithObjects(&testDeployment, &testEmptyCRD, &testService)
		client := clientBuilder.Build()
		ctx := context.Background()
		objs := []unstructured.Unstructured{
			testDeployment, testEmptyCRD, testService,
		}
		r := &fsm{
			log: zap.NewNop().Sugar(),
			K8s: K8s{Client: client},
			Cfg: Cfg{Objs: objs},
		}

		strategyFn := deletionStrategyBuilder(upstreamDeletionStrategy)

		fn, resp, err := strategyFn(ctx, r, &systemState{})
		require.Nil(t, resp)
		require.NoError(t, err)
		require.Equal(t, fnName(fn), fnName(sFnRemoveFinalizer))

		// check deletion progress
		require.False(t, canGetFakeResource(client, testDeployment))
		require.True(t, canGetFakeResource(client, testEmptyCRD))
		require.False(t, canGetFakeResource(client, testService))
	})

	t.Run("safe delete strategy", func(t *testing.T) {
		clientBuilder := fake.NewClientBuilder().
			WithObjects(&testDeployment, &testService, &testCRD)
		client := clientBuilder.Build()
		ctx := context.Background()
		objs := []unstructured.Unstructured{
			testDeployment, testService, testCRD,
		}
		r := &fsm{
			log: zap.NewNop().Sugar(),
			K8s: K8s{Client: client},
			Cfg: Cfg{Objs: objs},
		}

		strategyFn := deletionStrategyBuilder(safeDeletionStrategy)

		fn, resp, err := strategyFn(ctx, r, &systemState{})
		require.Nil(t, resp)
		require.NoError(t, err)
		require.Equal(t, fnName(fn), fnName(sFnRemoveFinalizer))

		// check deletion progress
		require.False(t, canGetFakeResource(client, testDeployment))
		require.False(t, canGetFakeResource(client, testService))
		require.False(t, canGetFakeResource(client, testCRD))
	})

	t.Run("safe delete with orphan resources error", func(t *testing.T) {
		clientBuilder := fake.NewClientBuilder().
			WithObjects(&testDeployment, &testEmptyCRD, &testService, &testCRD, &testCR)
		client := clientBuilder.Build()
		ctx := context.Background()
		objs := []unstructured.Unstructured{
			testDeployment, testEmptyCRD, testService,
		}
		r := &fsm{
			log: zap.NewNop().Sugar(),
			K8s: K8s{Client: client},
			Cfg: Cfg{Objs: objs},
		}

		strategy := "" // empty string should be resolved as safeDeletionStrategy
		strategyFn := deletionStrategyBuilder(deletionStrategy(strategy))

		fn, resp, err := strategyFn(ctx, r, &systemState{})
		require.Nil(t, resp)
		require.NoError(t, err)
		require.Equal(t, fnName(fn), fnName(sFnUpdateStatus(nil, nil)))

		// check deletion progress
		require.True(t, canGetFakeResource(client, testDeployment))
		require.True(t, canGetFakeResource(client, testEmptyCRD))
		require.True(t, canGetFakeResource(client, testService))
		require.True(t, canGetFakeResource(client, testCRD))
		require.True(t, canGetFakeResource(client, testCR))
	})
}

func fnName(fn interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func canGetFakeResource(c client.Client, u unstructured.Unstructured) bool {
	err := c.Get(context.Background(),
		types.NamespacedName{
			Name:      u.GetName(),
			Namespace: u.GetNamespace(),
		}, &u)
	return err == nil
}
