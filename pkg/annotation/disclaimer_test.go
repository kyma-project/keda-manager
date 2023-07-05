package annotation

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestAddDoNotEditDisclaimer(t *testing.T) {
	t.Run("add disclaimer", func(t *testing.T) {
		obj := unstructured.Unstructured{}
		obj = AddDoNotEditDisclaimer(obj)

		require.Equal(t, message, obj.GetAnnotations()[annotation])
	})
}
