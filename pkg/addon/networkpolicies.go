package addon

import (
	"fmt"
	"os"

	yamlutil "github.com/kyma-project/keda-manager/pkg/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const addonNetworkPoliciesFile = "keda-addon-networkpolicies.yaml"

// NetworkPolicies returns the NetworkPolicy objects required for the http-add-on
// components (scaler, interceptor, operator) to function correctly.
// The policies are loaded from the embedded YAML file and their namespace is
// overridden to the given value.
func NetworkPolicies(namespace string) []unstructured.Unstructured {
	f, err := os.Open(addonNetworkPoliciesFile)
	if err != nil {
		panic(fmt.Sprintf("failed to open %s: %v", addonNetworkPoliciesFile, err))
	}
	defer f.Close()

	objs, err := yamlutil.LoadData(f)
	if err != nil {
		panic(fmt.Sprintf("failed to parse %s: %v", addonNetworkPoliciesFile, err))
	}

	for i := range objs {
		objs[i].SetNamespace(namespace)
	}

	return objs
}
