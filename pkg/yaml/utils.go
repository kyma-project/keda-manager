package yaml

import (
	"encoding/json"
	"io"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// normalizeToJSON round-trips a value through JSON encoding/decoding so that
// all numeric types become float64 (JSON-compatible), avoiding panics in
// k8s.io/apimachinery's DeepCopyJSONValue which does not handle Go int.
func normalizeToJSON(v map[string]interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

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

		// Skip empty documents (e.g. blank --- separators in multi-doc YAML).
		if len(obj) == 0 {
			continue
		}

		// Normalize numeric types to JSON-compatible float64 to avoid panics
		// in k8s.io/apimachinery's DeepCopyJSONValue (which doesn't handle int).
		obj, err = normalizeToJSON(obj)
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
