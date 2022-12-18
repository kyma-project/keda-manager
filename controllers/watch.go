package controllers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kyma-project/keda-manager/pkg/crypto/sha256"
)

func registerWatchDistinct(objs []unstructured.Unstructured, registerWatch func(unstructured.Unstructured)) error {
	visited := map[string]struct{}{}
	for _, obj := range objs {
		shaStr, err := sha256.DefaultCalculator.CalculateSum(obj)
		if err != nil {
			return err
		}

		if _, found := visited[shaStr]; found {
			continue
		}

		registerWatch(obj)
		visited[shaStr] = struct{}{}
	}
	return nil
}
