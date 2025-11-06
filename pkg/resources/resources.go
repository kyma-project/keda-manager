package resources

import (
	"fmt"
	"os"

	"github.com/kyma-project/keda-manager/pkg/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func LoadFromPaths(path ...string) ([]unstructured.Unstructured, error) {
	var objs []unstructured.Unstructured
	for _, p := range path {
		file, err := os.Open(p)
		if err != nil {
			return nil, fmt.Errorf("unable to open file %s: %w", p, err)
		}
		defer file.Close()

		data, err := yaml.LoadData(file)
		if err != nil {
			return nil, fmt.Errorf("unable to parse file %s: %w", p, err)
		}

		objs = append(objs, data...)
	}

	return objs, nil
}
