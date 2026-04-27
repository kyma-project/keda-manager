package reconciler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/kyma-project/keda-manager/pkg/addon"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	conditionTypeAddon = "Addon"

	conditionReasonAddonInstalled  = "AddonInstalled"
	conditionReasonAddonDeleted    = "AddonDeleted"
	conditionReasonAddonInstallErr = "AddonInstallErr"
	conditionReasonAddonDisabled   = "AddonDisabled"
	conditionReasonAddonVersionErr = "AddonVersionErr"

	annotationAddonEnabled   = "keda.kyma-project.io/addon-enabled"
	annotationAddonVersion   = "keda.kyma-project.io/addon-version"
	annotationAddonNamespace = "keda.kyma-project.io/addon-namespace"

	annotationAddonInstalledVersion   = "keda.kyma-project.io/addon-installed-version"
	annotationAddonInstalledNamespace = "keda.kyma-project.io/addon-installed-namespace"

	defaultAddonNamespace = "keda"
)

// setAddonCondition sets an addon-specific condition on the Keda CR status.
func setAddonCondition(instance *v1alpha1.Keda, status metav1.ConditionStatus, reason, msg string) {
	condition := metav1.Condition{
		Type:               conditionTypeAddon,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            msg,
	}
	meta.SetStatusCondition(&instance.Status.Conditions, condition)
}

// addonCfg holds the addon configuration read from the Keda CR annotations.
type addonCfg struct {
	enabled   bool
	version   string
	namespace string
}

func readAddonCfg(instance *v1alpha1.Keda) addonCfg {
	ann := instance.GetAnnotations()
	if ann == nil {
		return addonCfg{}
	}
	return addonCfg{
		enabled:   strings.EqualFold(ann[annotationAddonEnabled], "true"),
		version:   ann[annotationAddonVersion],
		namespace: ann[annotationAddonNamespace],
	}
}

func (a addonCfg) effectiveNamespace() string {
	if a.namespace == "" {
		return defaultAddonNamespace
	}
	return a.namespace
}

// setAnnotation updates (or removes when value is empty) an annotation on the Keda CR in-memory.
func setAnnotation(instance *v1alpha1.Keda, key, value string) {
	ann := instance.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	if value == "" {
		delete(ann, key)
	} else {
		ann[key] = value
	}
	instance.SetAnnotations(ann)
}

// ensureNamespace creates the target namespace if it does not exist and labels
// it with istio-injection=enabled.
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
