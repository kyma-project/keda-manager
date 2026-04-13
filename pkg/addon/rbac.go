package addon

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RBACResources returns the ClusterRole and ClusterRoleBinding objects
// required by the HTTP add-on components (operator, scaler, interceptor).
// These supplement the RBAC shipped in the upstream manifest so the
// components can work in a locked-down cluster.
func RBACResources() []unstructured.Unstructured {
	var resources []unstructured.Unstructured

	// ── operator ────────────────────────────────────────────────────────
	// The http-add-on operator needs to manage ScaledObjects, HTTPScaledObjects,
	// and related core resources.
	operatorClusterRole := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRole",
			"metadata": map[string]interface{}{
				"name": "keda-add-ons-http-operator",
				"labels": map[string]interface{}{
					"app.kubernetes.io/component": "operator",
					"app.kubernetes.io/name":      "http-add-on",
					"app.kubernetes.io/part-of":   "keda",
				},
			},
			"rules": []interface{}{
				map[string]interface{}{
					"apiGroups": []interface{}{"keda.sh"},
					"resources": []interface{}{
						"scaledobjects",
						"scaledobjects/status",
						"scaledobjects/finalizers",
					},
					"verbs": []interface{}{"get", "list", "watch", "create", "update", "patch", "delete"},
				},
				map[string]interface{}{
					"apiGroups": []interface{}{"http.keda.sh"},
					"resources": []interface{}{
						"httpscaledobjects",
						"httpscaledobjects/status",
						"httpscaledobjects/finalizers",
					},
					"verbs": []interface{}{"get", "list", "watch", "create", "update", "patch", "delete"},
				},
				map[string]interface{}{
					"apiGroups": []interface{}{""},
					"resources": []interface{}{
						"configmaps",
						"configmaps/status",
						"events",
						"services",
						"endpoints",
						"pods",
					},
					"verbs": []interface{}{"get", "list", "watch", "create", "update", "patch"},
				},
				map[string]interface{}{
					"apiGroups": []interface{}{"apps"},
					"resources": []interface{}{
						"deployments",
					},
					"verbs": []interface{}{"get", "list", "watch", "create", "update", "patch"},
				},
				map[string]interface{}{
					"apiGroups": []interface{}{"coordination.k8s.io"},
					"resources": []interface{}{"leases"},
					"verbs":     []interface{}{"get", "list", "watch", "create", "update", "patch", "delete"},
				},
				map[string]interface{}{
					"apiGroups": []interface{}{"discovery.k8s.io"},
					"resources": []interface{}{"endpointslices"},
					"verbs":     []interface{}{"get", "list", "watch"},
				},
			},
		},
	}

	operatorClusterRoleBinding := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRoleBinding",
			"metadata": map[string]interface{}{
				"name": "keda-add-ons-http-operator",
				"labels": map[string]interface{}{
					"app.kubernetes.io/component": "operator",
					"app.kubernetes.io/name":      "http-add-on",
					"app.kubernetes.io/part-of":   "keda",
				},
			},
			"roleRef": map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     "keda-add-ons-http-operator",
			},
			"subjects": []interface{}{
				map[string]interface{}{
					"kind":      "ServiceAccount",
					"name":      "keda-add-ons-http-operator",
					"namespace": addonNamespace,
				},
			},
		},
	}

	// ── scaler (external-scaler) ────────────────────────────────────────
	// The scaler needs to watch EndpointSlices and HTTPScaledObjects to
	// provide scaling metrics back to the core KEDA operator.
	scalerClusterRole := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRole",
			"metadata": map[string]interface{}{
				"name": "keda-add-ons-http-scaler",
				"labels": map[string]interface{}{
					"app.kubernetes.io/component": "scaler",
					"app.kubernetes.io/name":      "http-add-on",
					"app.kubernetes.io/part-of":   "keda",
				},
			},
			"rules": []interface{}{
				map[string]interface{}{
					"apiGroups": []interface{}{"discovery.k8s.io"},
					"resources": []interface{}{"endpointslices"},
					"verbs":     []interface{}{"get", "list", "watch"},
				},
				map[string]interface{}{
					"apiGroups": []interface{}{"http.keda.sh"},
					"resources": []interface{}{
						"httpscaledobjects",
						"httpscaledobjects/status",
					},
					"verbs": []interface{}{"get", "list", "watch"},
				},
				map[string]interface{}{
					"apiGroups": []interface{}{""},
					"resources": []interface{}{
						"endpoints",
						"configmaps",
					},
					"verbs": []interface{}{"get", "list", "watch"},
				},
			},
		},
	}

	scalerClusterRoleBinding := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRoleBinding",
			"metadata": map[string]interface{}{
				"name": "keda-add-ons-http-scaler",
				"labels": map[string]interface{}{
					"app.kubernetes.io/component": "scaler",
					"app.kubernetes.io/name":      "http-add-on",
					"app.kubernetes.io/part-of":   "keda",
				},
			},
			"roleRef": map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     "keda-add-ons-http-scaler",
			},
			"subjects": []interface{}{
				map[string]interface{}{
					"kind":      "ServiceAccount",
					"name":      "keda-add-ons-http-scaler",
					"namespace": addonNamespace,
				},
			},
		},
	}

	// ── interceptor ─────────────────────────────────────────────────────
	// The interceptor watches HTTPScaledObjects and EndpointSlices so it
	// knows where to route incoming HTTP requests.
	interceptorClusterRole := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRole",
			"metadata": map[string]interface{}{
				"name": "keda-add-ons-http-interceptor",
				"labels": map[string]interface{}{
					"app.kubernetes.io/component": "interceptor",
					"app.kubernetes.io/name":      "http-add-on",
					"app.kubernetes.io/part-of":   "keda",
				},
			},
			"rules": []interface{}{
				map[string]interface{}{
					"apiGroups": []interface{}{"http.keda.sh"},
					"resources": []interface{}{
						"httpscaledobjects",
						"httpscaledobjects/status",
					},
					"verbs": []interface{}{"get", "list", "watch"},
				},
				map[string]interface{}{
					"apiGroups": []interface{}{"discovery.k8s.io"},
					"resources": []interface{}{"endpointslices"},
					"verbs":     []interface{}{"get", "list", "watch"},
				},
				map[string]interface{}{
					"apiGroups": []interface{}{""},
					"resources": []interface{}{
						"endpoints",
						"configmaps",
					},
					"verbs": []interface{}{"get", "list", "watch"},
				},
			},
		},
	}

	interceptorClusterRoleBinding := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRoleBinding",
			"metadata": map[string]interface{}{
				"name": "keda-add-ons-http-interceptor",
				"labels": map[string]interface{}{
					"app.kubernetes.io/component": "interceptor",
					"app.kubernetes.io/name":      "http-add-on",
					"app.kubernetes.io/part-of":   "keda",
				},
			},
			"roleRef": map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     "keda-add-ons-http-interceptor",
			},
			"subjects": []interface{}{
				map[string]interface{}{
					"kind":      "ServiceAccount",
					"name":      "keda-add-ons-http-interceptor",
					"namespace": addonNamespace,
				},
			},
		},
	}

	resources = append(resources,
		operatorClusterRole,
		operatorClusterRoleBinding,
		scalerClusterRole,
		scalerClusterRoleBinding,
		interceptorClusterRole,
		interceptorClusterRoleBinding,
	)

	return resources
}
