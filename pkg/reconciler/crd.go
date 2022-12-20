package reconciler

import (
	"context"
	apixtv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InstallCRDs(ctx context.Context, c client.Client, crds []unstructured.Unstructured) (bool, error) {
	var installed bool
	for _, obj := range crds {
		var crd apixtv1.CustomResourceDefinition
		keyObj := client.ObjectKeyFromObject(&obj)
		err := c.Get(ctx, keyObj, &crd)

		if client.IgnoreNotFound(err) != nil {
			return false, err
		}
		// crd exists - continue with crds installation
		if err == nil {
			continue
		}
		// crd does not exit - create it
		if err = c.Create(ctx, &obj); err != nil {
			return false, err
		}

		installed = true
	}
	return installed, nil
}
