package addon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	yamlutil "github.com/kyma-project/keda-manager/pkg/yaml"
)

func TestPeerAuthentication_PortLevelPermissiveOnEnvoyStats(t *testing.T) {
	obj := PeerAuthentication("custom-ns")

	require.Equal(t, "PeerAuthentication", obj.GetKind())
	require.Equal(t, "security.istio.io/v1", obj.GetAPIVersion())
	require.Equal(t, "keda-add-ons-http-envoy-metrics", obj.GetName())
	require.Equal(t, "custom-ns", obj.GetNamespace())

	selector, found, err := unstructured.NestedStringMap(obj.Object, "spec", "selector", "matchLabels")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, map[string]string{
		"app.kubernetes.io/component": "add-on",
		"app.kubernetes.io/name":      "http",
		"app.kubernetes.io/part-of":   "keda",
	}, selector)

	mode, found, err := unstructured.NestedString(obj.Object, "spec", "portLevelMtls", "15090", "mode")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "PERMISSIVE", mode)

	_, found, err = unstructured.NestedFieldNoCopy(obj.Object, "spec", "mtls")
	require.NoError(t, err)
	require.False(t, found, "must not weaken mTLS on application ports with workload-wide PERMISSIVE")
}

func TestAddonNetworkPoliciesAllowEnvoyStatsScrape(t *testing.T) {
	objs := loadAddonNetworkPolicies(t)

	wantInstances := map[string]bool{
		"interceptor":     false,
		"external-scaler": false,
		"operator":        false,
	}
	for i := range objs {
		if !networkPolicyAllowsEnvoyStatsFromMetricAgent(&objs[i]) {
			continue
		}
		instance, _, _ := unstructured.NestedString(objs[i].Object, "spec", "podSelector", "matchLabels", "app.kubernetes.io/instance")
		if _, ok := wantInstances[instance]; ok {
			wantInstances[instance] = true
		}
	}
	for instance, found := range wantInstances {
		require.True(t, found, "missing 15090 scrape NetworkPolicy for HTTP add-on instance %s", instance)
	}
}

func loadAddonNetworkPolicies(t *testing.T) []unstructured.Unstructured {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	var root string
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			root = dir
			break
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "go.mod not found walking up from %s", dir)
		dir = parent
	}
	f, err := os.Open(filepath.Join(root, addonNetworkPoliciesFile))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	objs, err := yamlutil.LoadData(f)
	require.NoError(t, err)
	return objs
}

func networkPolicyAllowsEnvoyStatsFromMetricAgent(obj *unstructured.Unstructured) bool {
	if obj.GetKind() != "NetworkPolicy" {
		return false
	}
	ingress, found, err := unstructured.NestedSlice(obj.Object, "spec", "ingress")
	if err != nil || !found {
		return false
	}
	for _, raw := range ingress {
		rule, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if !ingressRuleHasPort(rule, 15090) || !ingressRuleFromMetricsScraping(rule) {
			continue
		}
		return true
	}
	return false
}

func ingressRuleHasPort(rule map[string]interface{}, port int64) bool {
	ports, _ := rule["ports"].([]interface{})
	for _, raw := range ports {
		p, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		switch v := p["port"].(type) {
		case float64:
			if int64(v) == port {
				return true
			}
		case int64:
			if v == port {
				return true
			}
		}
	}
	return false
}

func ingressRuleFromMetricsScraping(rule map[string]interface{}) bool {
	from, _ := rule["from"].([]interface{})
	for _, raw := range from {
		peer, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		pod, _ := peer["podSelector"].(map[string]interface{})
		labels, _ := pod["matchLabels"].(map[string]interface{})
		if labels["networking.kyma-project.io/metrics-scraping"] != "allowed" {
			continue
		}
		ns, _ := peer["namespaceSelector"].(map[string]interface{})
		nsLabels, _ := ns["matchLabels"].(map[string]interface{})
		if nsLabels["kubernetes.io/metadata.name"] == "kyma-system" {
			return true
		}
	}
	return false
}
