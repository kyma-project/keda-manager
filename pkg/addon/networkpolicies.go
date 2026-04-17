package addon

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// NetworkPolicies returns the NetworkPolicy objects required for the http-add-on
// components (scaler, interceptor, operator) to function correctly.
// These policies allow egress to the Kubernetes API server and DNS, plus the
// required inter-component traffic within the given namespace.
func NetworkPolicies(namespace string) []unstructured.Unstructured {
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
					"namespace": namespace,
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
					"namespace": namespace,
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
				"namespace": namespace,
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
				"namespace": namespace,
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
				"namespace": namespace,
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

	// ── Ingress policies ──────────────────────────────────────────────────
	// These are required when a default-deny NetworkPolicy is present in the
	// namespace; without them inter-component traffic is blocked.

	// Allow ingress to interceptor admin port (9090) from scaler pods.
	// The scaler pings the interceptor-admin endpoint to fetch request queue
	// counts — this is the traffic that causes the "there isn't any valid
	// interceptor endpoint" error when blocked.
	interceptorIngressFromScalerPolicy := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "NetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      "keda-add-ons-http-interceptor-allow-from-scaler",
				"namespace": namespace,
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
				"policyTypes": []interface{}{"Ingress"},
				"ingress": []interface{}{
					map[string]interface{}{
						"ports": []interface{}{
							map[string]interface{}{
								"port":     int64(9090),
								"protocol": "TCP",
							},
						},
						"from": []interface{}{
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

	// Allow ingress to interceptor proxy port (8080) from any pod.
	// The interceptor is a reverse proxy that receives HTTP requests from
	// workloads/ingress and forwards them to the target service.
	interceptorIngressProxyPolicy := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "NetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      "keda-add-ons-http-interceptor-allow-ingress-proxy",
				"namespace": namespace,
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
				"policyTypes": []interface{}{"Ingress"},
				"ingress": []interface{}{
					map[string]interface{}{
						"ports": []interface{}{
							map[string]interface{}{
								"port":     int64(8080),
								"protocol": "TCP",
							},
						},
					},
				},
			},
		},
	}

	// Allow ingress to scaler gRPC port (9090) from any pod.
	// The KEDA operator (which may run in a different namespace such as
	// keda-system) connects to the external-scaler gRPC endpoint to query
	// scaling metrics.
	scalerIngressPolicy := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "NetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      "keda-add-ons-http-scaler-allow-ingress",
				"namespace": namespace,
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
				"policyTypes": []interface{}{"Ingress"},
				"ingress": []interface{}{
					map[string]interface{}{
						"ports": []interface{}{
							map[string]interface{}{
								"port":     int64(9090),
								"protocol": "TCP",
							},
						},
					},
				},
			},
		},
	}

	policies = append(policies,
		scalerToInterceptorPolicy,
		operatorToScalerPolicy,
		interceptorEgressPolicy,
		interceptorIngressFromScalerPolicy,
		interceptorIngressProxyPolicy,
		scalerIngressPolicy,
	)

	return policies
}
