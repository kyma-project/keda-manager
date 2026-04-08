package addon

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// addonNamespace is the namespace where the http-add-on components are deployed.
const addonNamespace = "keda"

// NetworkPolicies returns the NetworkPolicy objects required for the http-add-on
// components (scaler, interceptor, operator) to function correctly.
// These policies allow egress to the Kubernetes API server and DNS, plus the
// required inter-component traffic within the keda namespace.
func NetworkPolicies() []unstructured.Unstructured {
	components := []struct {
		name      string
		component string
		selector  map[string]interface{}
	}{
		{
			name:      "keda-add-ons-http-scaler",
			component: "scaler",
			selector: map[string]interface{}{
				"app.kubernetes.io/component": "add-on",
				"app.kubernetes.io/instance":  "external-scaler",
				"app.kubernetes.io/name":      "http",
				"app.kubernetes.io/part-of":   "keda",
			},
		},
		{
			name:      "keda-add-ons-http-interceptor",
			component: "interceptor",
			selector: map[string]interface{}{
				"app.kubernetes.io/component": "add-on",
				"app.kubernetes.io/instance":  "interceptor",
				"app.kubernetes.io/name":      "http",
				"app.kubernetes.io/part-of":   "keda",
			},
		},
		{
			name:      "keda-add-ons-http-operator",
			component: "operator",
			selector: map[string]interface{}{
				"app.kubernetes.io/component": "add-on",
				"app.kubernetes.io/instance":  "operator",
				"app.kubernetes.io/name":      "http",
				"app.kubernetes.io/part-of":   "keda",
			},
		},
	}

	var policies []unstructured.Unstructured

	for _, c := range components {
		// Allow egress to the Kubernetes API server
		apiServerPolicy := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "networking.k8s.io/v1",
				"kind":       "NetworkPolicy",
				"metadata": map[string]interface{}{
					"name":      c.name + "-allow-to-apiserver",
					"namespace": addonNamespace,
					"labels": map[string]interface{}{
						"app.kubernetes.io/component": c.component,
						"app.kubernetes.io/name":      "http-add-on",
						"app.kubernetes.io/part-of":   "keda",
					},
				},
				"spec": map[string]interface{}{
					"podSelector": map[string]interface{}{
						"matchLabels": c.selector,
					},
					"policyTypes": []interface{}{"Egress"},
					"egress": []interface{}{
						map[string]interface{}{
							"ports": []interface{}{
								map[string]interface{}{
									"port":     int64(443),
									"protocol": "TCP",
								},
								map[string]interface{}{
									"port":     int64(6443),
									"protocol": "TCP",
								},
							},
						},
					},
				},
			},
		}

		// Allow egress to DNS
		dnsPolicy := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "networking.k8s.io/v1",
				"kind":       "NetworkPolicy",
				"metadata": map[string]interface{}{
					"name":      c.name + "-allow-to-dns",
					"namespace": addonNamespace,
					"labels": map[string]interface{}{
						"app.kubernetes.io/component": c.component,
						"app.kubernetes.io/name":      "http-add-on",
						"app.kubernetes.io/part-of":   "keda",
					},
				},
				"spec": map[string]interface{}{
					"podSelector": map[string]interface{}{
						"matchLabels": c.selector,
					},
					"policyTypes": []interface{}{"Egress"},
					"egress": []interface{}{
						map[string]interface{}{
							"ports": []interface{}{
								map[string]interface{}{
									"port":     int64(53),
									"protocol": "UDP",
								},
								map[string]interface{}{
									"port":     int64(53),
									"protocol": "TCP",
								},
							},
							"to": []interface{}{
								map[string]interface{}{
									"ipBlock": map[string]interface{}{
										"cidr": "0.0.0.0/0",
									},
								},
							},
						},
					},
				},
			},
		}

		policies = append(policies, apiServerPolicy, dnsPolicy)
	}

	// Allow scaler → interceptor-admin (gRPC) traffic
	scalerToInterceptorPolicy := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "NetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      "keda-add-ons-http-scaler-allow-to-interceptor",
				"namespace": addonNamespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/component": "scaler",
					"app.kubernetes.io/name":      "http-add-on",
					"app.kubernetes.io/part-of":   "keda",
				},
			},
			"spec": map[string]interface{}{
				"podSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app.kubernetes.io/component": "add-on",
						"app.kubernetes.io/instance":  "external-scaler",
						"app.kubernetes.io/name":      "http",
						"app.kubernetes.io/part-of":   "keda",
					},
				},
				"policyTypes": []interface{}{"Egress"},
				"egress": []interface{}{
					map[string]interface{}{
						"ports": []interface{}{
							map[string]interface{}{
								"port":     int64(9090),
								"protocol": "TCP",
							},
						},
						"to": []interface{}{
							map[string]interface{}{
								"podSelector": map[string]interface{}{
									"matchLabels": map[string]interface{}{
										"app.kubernetes.io/component": "add-on",
										"app.kubernetes.io/instance":  "interceptor",
										"app.kubernetes.io/name":      "http",
										"app.kubernetes.io/part-of":   "keda",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Allow KEDA operator → scaler (external scaler gRPC) traffic
	operatorToScalerPolicy := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "NetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      "keda-add-ons-http-operator-allow-to-scaler",
				"namespace": addonNamespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/component": "operator",
					"app.kubernetes.io/name":      "http-add-on",
					"app.kubernetes.io/part-of":   "keda",
				},
			},
			"spec": map[string]interface{}{
				"podSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app.kubernetes.io/component": "add-on",
						"app.kubernetes.io/instance":  "operator",
						"app.kubernetes.io/name":      "http",
						"app.kubernetes.io/part-of":   "keda",
					},
				},
				"policyTypes": []interface{}{"Egress"},
				"egress": []interface{}{
					map[string]interface{}{
						"ports": []interface{}{
							map[string]interface{}{
								"port":     int64(9090),
								"protocol": "TCP",
							},
						},
						"to": []interface{}{
							map[string]interface{}{
								"podSelector": map[string]interface{}{
									"matchLabels": map[string]interface{}{
										"app.kubernetes.io/component": "add-on",
										"app.kubernetes.io/instance":  "external-scaler",
										"app.kubernetes.io/name":      "http",
										"app.kubernetes.io/part-of":   "keda",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Allow interceptor to proxy traffic to workload services (all egress)
	interceptorEgressPolicy := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "NetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      "keda-add-ons-http-interceptor-allow-egress",
				"namespace": addonNamespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/component": "interceptor",
					"app.kubernetes.io/name":      "http-add-on",
					"app.kubernetes.io/part-of":   "keda",
				},
			},
			"spec": map[string]interface{}{
				"podSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app.kubernetes.io/component": "add-on",
						"app.kubernetes.io/instance":  "interceptor",
						"app.kubernetes.io/name":      "http",
						"app.kubernetes.io/part-of":   "keda",
					},
				},
				"policyTypes": []interface{}{"Egress"},
				"egress":      []interface{}{map[string]interface{}{}},
			},
		},
	}

	policies = append(policies,
		scalerToInterceptorPolicy,
		operatorToScalerPolicy,
		interceptorEgressPolicy,
	)

	return policies
}

