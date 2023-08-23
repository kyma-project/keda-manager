package label

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetKedaNamespaceLabels(ctx context.Context, c client.Client) {
	namespace := corev1.Namespace{}
	match := client.ObjectKey{
		Namespace: "kyma-system",
	}


	err := c.Get(ctx, match, namespace)
	if err != nil
}
