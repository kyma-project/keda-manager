package reconciler

import (
	"context"
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	testResource1 = unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "Deployment",
			"apiVersion": "apps/v1",
			"metadata": map[string]interface{}{
				"name":      "test-resource-1",
				"namespace": "test",
			},
		},
	}

	testResource2 = unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "CustomResourceDefinition",
			"apiVersion": "apiextensions.k8s.io/v1",
			"metadata": map[string]interface{}{
				"name":      "test-resource-2",
				"namespace": "test",
			},
		},
	}

	testResource3 = unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "Service",
			"apiVersion": "v1",
			"metadata": map[string]interface{}{
				"name":      "test-resource-3",
				"namespace": "test",
			},
		},
	}
)

func Test_sFnDeleteResources(t *testing.T) {
	t.Run("cascade delete strategy", func(t *testing.T) {
		clientBuilder := fake.NewClientBuilder().
			WithObjects(&testResource1, &testResource2, &testResource3)
		client := clientBuilder.Build()
		ctx := context.Background()
		objs := []unstructured.Unstructured{
			testResource1, testResource2, testResource3,
		}
		r := &fsm{
			log: zap.NewNop().Sugar(),
			K8s: K8s{Client: client},
			Cfg: Cfg{Objs: objs},
		}

		fn, resp, err := sFnCascadeDeleteStrategy(ctx, r, &systemState{})
		require.Nil(t, resp)
		require.NoError(t, err)
		require.Equal(t, fnName(fn), fnName(sFnRemoveFinalizer))

		require.Error(t, canGetFakeResource(client, testResource1))
		require.Error(t, canGetFakeResource(client, testResource2))
		require.Error(t, canGetFakeResource(client, testResource3))
	})

	t.Run("safe delete strategy", func(t *testing.T) {
		clientBuilder := fake.NewClientBuilder().
			WithObjects(&testResource1, &testResource2, &testResource3)
		client := clientBuilder.Build()
		ctx := context.Background()
		objs := []unstructured.Unstructured{
			testResource1, testResource2, testResource3,
		}
		r := &fsm{
			log: zap.NewNop().Sugar(),
			K8s: K8s{Client: client},
			Cfg: Cfg{Objs: objs},
		}

		fn, resp, err := sFnSafeDeleteStrategy(ctx, r, &systemState{})
		require.Nil(t, resp)
		require.NoError(t, err)
		require.Equal(t, fnName(fn), fnName(sFnRemoveFinalizer))

		require.Error(t, canGetFakeResource(client, testResource1))
		require.NoError(t, canGetFakeResource(client, testResource2))
		require.Error(t, canGetFakeResource(client, testResource3))
	})
}

func fnName(fn interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func canGetFakeResource(c client.Client, u unstructured.Unstructured) error {
	return c.Get(context.Background(),
		types.NamespacedName{
			Name:      testResource2.GetName(),
			Namespace: testResource2.GetNamespace(),
		},
		&u)
}
