package yaml

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadData(t *testing.T) {
	t.Run("parses single document", func(t *testing.T) {
		input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: default
`
		objs, err := LoadData(strings.NewReader(input))
		require.NoError(t, err)
		require.Len(t, objs, 1)
		require.Equal(t, "ConfigMap", objs[0].GetKind())
		require.Equal(t, "test-cm", objs[0].GetName())
	})

	t.Run("parses multi-document YAML", func(t *testing.T) {
		input := `apiVersion: v1
kind: Service
metadata:
  name: svc1
---
apiVersion: v1
kind: Service
metadata:
  name: svc2
`
		objs, err := LoadData(strings.NewReader(input))
		require.NoError(t, err)
		require.Len(t, objs, 2)
		require.Equal(t, "svc1", objs[0].GetName())
		require.Equal(t, "svc2", objs[1].GetName())
	})

	t.Run("CRDs are prepended", func(t *testing.T) {
		input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: dep
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: foos.example.com
`
		objs, err := LoadData(strings.NewReader(input))
		require.NoError(t, err)
		require.Len(t, objs, 2)
		require.Equal(t, "CustomResourceDefinition", objs[0].GetKind())
		require.Equal(t, "Deployment", objs[1].GetKind())
	})

	t.Run("skips empty documents", func(t *testing.T) {
		input := `---
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
---
`
		objs, err := LoadData(strings.NewReader(input))
		require.NoError(t, err)
		require.Len(t, objs, 1)
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		objs, err := LoadData(strings.NewReader(""))
		require.NoError(t, err)
		require.Empty(t, objs)
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		_, err := LoadData(strings.NewReader("{{invalid"))
		require.Error(t, err)
	})

	t.Run("normalizes integers to float64", func(t *testing.T) {
		input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
data:
  replicas: 3
`
		objs, err := LoadData(strings.NewReader(input))
		require.NoError(t, err)
		require.Len(t, objs, 1)
		data, _ := objs[0].Object["data"].(map[string]interface{})
		if data != nil {
			if v, ok := data["replicas"]; ok {
				_, isFloat := v.(float64)
				require.True(t, isFloat, "expected float64 after normalization, got %T", v)
			}
		}
	})
}

