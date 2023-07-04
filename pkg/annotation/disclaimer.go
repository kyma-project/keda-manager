package annotation

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

const (
	annotation = "serverless-manager.kyma-project.io/managed-by-serverless-manager-disclaimer"
	message    = "DO NOT EDIT - This resource is managed by Serverless-Manager.\nAny modifications are discarded and the resource is reverted to the original state."
)

func AddDoNotEditDisclaimer(obj unstructured.Unstructured) unstructured.Unstructured {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations[annotation] = message
	obj.SetAnnotations(annotations)

	return obj
}
