package reconciler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ensureNamespace creates the target namespace if it does not exist and labels
// it with istio-injection=enabled.
//
//nolint:unused
func ensureNamespace(ctx context.Context, r *fsm, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: map[string]string{"istio-injection": "enabled"},
		},
	}
	err := r.Create(ctx, ns)
	if err == nil {
		r.log.Infof("created namespace %s with istio-injection=enabled", namespace)
		return nil
	}
	if !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
	}

	existing := &corev1.Namespace{}
	if getErr := r.Get(ctx, client.ObjectKeyFromObject(ns), existing); getErr != nil {
		return fmt.Errorf("failed to get existing namespace %s: %w", namespace, getErr)
	}
	if existing.Labels == nil {
		existing.Labels = map[string]string{}
	}
	if existing.Labels["istio-injection"] != "enabled" {
		existing.Labels["istio-injection"] = "enabled"
		if updateErr := r.Update(ctx, existing); updateErr != nil {
			return fmt.Errorf("failed to label namespace %s with istio-injection: %w", namespace, updateErr)
		}
	}
	return nil
}

const (
	istioExcludeInboundPortsAnnotation = "traffic.sidecar.istio.io/excludeInboundPorts"
	istioExcludeInboundPortsValue      = "9090"
)

var namespaceEnvVars = map[string]struct{}{
	"KEDA_HTTP_SCALER_TARGET_ADMIN_NAMESPACE": {},
	"KEDA_HTTP_OPERATOR_NAMESPACE":            {},
}

// overrideNamespace sets the namespace on all namespaced resources, patches
// subjects[].namespace on bindings, and patches Deployment env vars and Istio annotations.
func overrideNamespace(objs []unstructured.Unstructured, namespace string) {
	for i := range objs {
		obj := &objs[i]
		if obj.GetNamespace() != "" {
			obj.SetNamespace(namespace)
		}

		switch obj.GetKind() {
		case "ClusterRoleBinding", "RoleBinding":
			patchSubjectsNamespace(obj, namespace)
		case "Deployment":
			patchDeploymentEnvNamespace(obj, namespace)
			patchDeploymentIstioAnnotation(obj)
		}
	}
}

// patchDeploymentIstioAnnotation adds excludeInboundPorts="9090" so Istio
// does not intercept gRPC traffic on that port.
func patchDeploymentIstioAnnotation(obj *unstructured.Unstructured) {
	annotations, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
	if annotations == nil {
		annotations = map[string]string{}
	}
	if annotations[istioExcludeInboundPortsAnnotation] == istioExcludeInboundPortsValue {
		return
	}
	annotations[istioExcludeInboundPortsAnnotation] = istioExcludeInboundPortsValue
	_ = unstructured.SetNestedStringMap(obj.Object, annotations, "spec", "template", "metadata", "annotations")
}

// patchDeploymentEnvNamespace overrides namespace-referencing env vars in all
// containers of a Deployment.
func patchDeploymentEnvNamespace(obj *unstructured.Unstructured, namespace string) {
	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	if err != nil || !found {
		return
	}

	changed := false
	for ci, rawC := range containers {
		container, ok := rawC.(map[string]interface{})
		if !ok {
			continue
		}
		envList, ok := container["env"].([]interface{})
		if !ok {
			continue
		}
		for ei, rawE := range envList {
			envVar, ok := rawE.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := envVar["name"].(string)
			if _, match := namespaceEnvVars[name]; match && envVar["value"] != namespace {
				envVar["value"] = namespace
				envList[ei] = envVar
				changed = true
			}
		}
		container["env"] = envList
		containers[ci] = container
	}
	if changed {
		_ = unstructured.SetNestedSlice(obj.Object, containers, "spec", "template", "spec", "containers")
	}
}

// patchSubjectsNamespace updates ServiceAccount subject namespaces in bindings.
func patchSubjectsNamespace(obj *unstructured.Unstructured, namespace string) {
	subjects, found, err := unstructured.NestedSlice(obj.Object, "subjects")
	if err != nil || !found {
		return
	}

	changed := false
	for j, raw := range subjects {
		subj, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if kind, _, _ := unstructured.NestedString(subj, "kind"); kind != "ServiceAccount" {
			continue
		}
		if ns, _, _ := unstructured.NestedString(subj, "namespace"); ns != "" && ns != namespace {
			subj["namespace"] = namespace
			subjects[j] = subj
			changed = true
		}
	}
	if changed {
		_ = unstructured.SetNestedSlice(obj.Object, subjects, "subjects")
	}
}
