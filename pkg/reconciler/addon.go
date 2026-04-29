package reconciler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
