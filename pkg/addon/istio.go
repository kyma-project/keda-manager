package addon

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

const (
	peerAuthenticationName = "keda-add-ons-http-envoy-metrics"
	envoyStatsPort         = "15090"
)

// PeerAuthentication returns a port-level PERMISSIVE PeerAuthentication for
// the Istio sidecar stats port on HTTP add-on Pods. Under mesh-wide STRICT
// mTLS, telemetry-metric-agent scrapes :15090/stats/prometheus over plain
// HTTP; without this exemption the sidecar rejects the scrape.
func PeerAuthentication(namespace string) unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "security.istio.io/v1",
		"kind":       "PeerAuthentication",
		"metadata": map[string]interface{}{
			"name":      peerAuthenticationName,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"kyma-project.io/module":       "keda",
				"app.kubernetes.io/name":       "http-add-on",
				"app.kubernetes.io/part-of":    "keda-manager",
				"app.kubernetes.io/managed-by": "keda-manager",
				"app.kubernetes.io/component":  "istio",
			},
		},
		"spec": map[string]interface{}{
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app.kubernetes.io/component": "add-on",
					"app.kubernetes.io/name":      "http",
					"app.kubernetes.io/part-of":   "keda",
				},
			},
			"portLevelMtls": map[string]interface{}{
				envoyStatsPort: map[string]interface{}{
					"mode": "PERMISSIVE",
				},
			},
		},
	}}
}
