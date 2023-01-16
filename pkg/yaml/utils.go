package yaml

import (
	"io"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func LoadData(r io.Reader) ([]unstructured.Unstructured, error) {
	results := make([]unstructured.Unstructured, 0)
	decoder := yaml.NewDecoder(r)

	for {
		var obj map[string]interface{}
		err := decoder.Decode(&obj)

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		u := unstructured.Unstructured{Object: obj}
		if u.GetObjectKind().GroupVersionKind().Kind == "CustomResourceDefinition" {
			results = append([]unstructured.Unstructured{u}, results...)
			continue
		}
		results = append(results, u)
	}

	return results, nil
}
