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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// addonCfg holds the addon configuration read from the Keda CR annotations.
type addonCfg struct {
	enabled   bool
	version   string
	namespace string
}

func readAddonCfg(instance *v1alpha1.Keda) addonCfg {
	ann := instance.GetAnnotations()
	cfg := addonCfg{}
	if ann == nil {
		return cfg
	}
	cfg.enabled = strings.EqualFold(ann[v1alpha1.AnnotationAddonEnabled], "true")
	cfg.version = ann[v1alpha1.AnnotationAddonVersion]
	cfg.namespace = ann[v1alpha1.AnnotationAddonNamespace]
	return cfg
}

func (a addonCfg) effectiveNamespace() string {
	if a.namespace == "" {
		return v1alpha1.DefaultAddonNamespace
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

// sFnHandleAddon is entered after sFnVerify. It decides whether to apply or
// delete the HTTP add-on based on annotations on the Keda CR.
func sFnHandleAddon(_ context.Context, _ *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	cfg := readAddonCfg(&s.instance)

	// Add-on disabled (annotation absent or "false") → delete any existing addon resources.
	if !cfg.enabled {
		return switchState(sFnDeleteAddon)
	}

	version := cfg.version
	// No version specified → resolve the latest one at runtime.
	if version == "" {
		return switchState(sFnResolveAddonVersion)
	}

	cleanVersion, err := addon.ValidateVersion(version)
	if err != nil {
		s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonVersionErr, err.Error())
		return stopWithNoRequeue()
	}

	// Store the clean (trimmed) version back in the annotation so sFnApplyAddon can read it.
	setAnnotation(&s.instance, v1alpha1.AnnotationAddonVersion, cleanVersion)
	return switchState(sFnApplyAddon)
}

// sFnResolveAddonVersion fetches the latest tag from GitHub and then applies it.
func sFnResolveAddonVersion(_ context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	version, err := addon.LatestVersion(r.HTTPClient)
	if err != nil {
		r.log.With("err", err).Error("failed to resolve latest addon version")
		s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonVersionErr, err.Error())
		return stopWithNoRequeue()
	}
	r.log.Infof("resolved latest HTTP add-on version: %s", version)
	setAnnotation(&s.instance, v1alpha1.AnnotationAddonVersion, version)
	return switchState(sFnApplyAddon)
}

// ensureNamespace creates the target namespace if it does not exist and labels it
// with istio-injection=enabled so that Istio sidecars are injected into add-on pods.
func ensureNamespace(ctx context.Context, r *fsm, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"istio-injection": "enabled",
			},
		},
	}
	if err := r.Create(ctx, ns); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// Namespace exists — ensure the istio label is present.
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
		return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
	}
	r.log.Infof("created namespace %s with istio-injection=enabled", namespace)
	return nil
}

const istioExcludeInboundPortsAnnotation = "traffic.sidecar.istio.io/excludeInboundPorts"
const istioExcludeInboundPortsValue = "9090"

// patchDeploymentIstioAnnotation adds the Istio excludeInboundPorts="9090"
// annotation to every addon Deployment so that the Istio sidecar does not
// intercept gRPC traffic on port 9090, which would break health checks.
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

// namespaceEnvVars is the set of environment variable names in the HTTP add-on
// Deployments that contain a hardcoded namespace reference and must be patched
// when the add-on is installed into a non-default namespace.
var namespaceEnvVars = map[string]struct{}{
	"KEDA_HTTP_SCALER_TARGET_ADMIN_NAMESPACE": {},
	"KEDA_HTTP_OPERATOR_NAMESPACE":            {},
}

// overrideNamespace sets the namespace on all namespaced resources to the target
// namespace. For cluster-scoped resources (ClusterRoleBinding, ClusterRole) it
// patches subjects[].namespace so that ServiceAccount references point to the
// correct namespace. For Deployments it also patches environment variables that
// contain hardcoded namespace references (e.g. KEDA_HTTP_SCALER_TARGET_ADMIN_NAMESPACE).
func overrideNamespace(objs []unstructured.Unstructured, namespace string) {
	for i := range objs {
		// Namespaced resources — override the metadata.namespace.
		if objs[i].GetNamespace() != "" {
			objs[i].SetNamespace(namespace)
		}

		// ClusterRoleBindings / RoleBindings — patch subjects[].namespace.
		kind := objs[i].GetKind()
		if kind == "ClusterRoleBinding" || kind == "RoleBinding" {
			patchSubjectsNamespace(&objs[i], namespace)
		}

		// Deployments — patch env vars that reference the namespace and add Istio annotation.
		if kind == "Deployment" {
			patchDeploymentEnvNamespace(&objs[i], namespace)
			patchDeploymentIstioAnnotation(&objs[i])
		}
	}
}

// patchDeploymentEnvNamespace walks all containers in a Deployment and overrides
// environment variables whose names are in namespaceEnvVars to point to the
// given namespace. This is required because the upstream HTTP add-on manifest
// hardcodes the namespace "keda" in env vars like KEDA_HTTP_SCALER_TARGET_ADMIN_NAMESPACE.
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
		envRaw, exists := container["env"]
		if !exists {
			continue
		}
		envList, ok := envRaw.([]interface{})
		if !ok {
			continue
		}
		for ei, rawE := range envList {
			envVar, ok := rawE.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := envVar["name"].(string)
			if _, match := namespaceEnvVars[name]; match {
				if envVar["value"] != namespace {
					envVar["value"] = namespace
					envList[ei] = envVar
					changed = true
				}
			}
		}
		container["env"] = envList
		containers[ci] = container
	}
	if changed {
		_ = unstructured.SetNestedSlice(obj.Object, containers, "spec", "template", "spec", "containers")
	}
}

// patchSubjectsNamespace updates every subject's namespace field to the given
// namespace for ServiceAccount subjects.
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
		subjKind, _, _ := unstructured.NestedString(subj, "kind")
		if subjKind != "ServiceAccount" {
			continue
		}
		existingNS, _, _ := unstructured.NestedString(subj, "namespace")
		if existingNS != "" && existingNS != namespace {
			subj["namespace"] = namespace
			subjects[j] = subj
			changed = true
		}
	}
	if changed {
		_ = unstructured.SetNestedSlice(obj.Object, subjects, "subjects")
	}
}

// deleteOldAddonResources fetches the manifest for the given version, overrides
// namespaces to oldNS, and deletes each resource. It also deletes the
// NetworkPolicy objects that were added by us.
func deleteOldAddonResources(ctx context.Context, r *fsm, version, oldNS string) {
	oldObjs, err := addon.FetchResources(r.HTTPClient, version)
	if err != nil {
		r.log.With("err", err).Warn("failed to fetch old addon manifest for cleanup, proceeding with apply anyway")
		return
	}
	overrideNamespace(oldObjs, oldNS)
	// Also delete NetworkPolicies that we appended.
	oldObjs = append(oldObjs, addon.NetworkPolicies(oldNS)...)

	for i := range oldObjs {
		obj := oldObjs[i]
		if err := r.Delete(ctx, &obj); client.IgnoreNotFound(err) != nil {
			r.log.With("err", err).With("name", obj.GetName()).Warn("failed to delete old addon resource")
		}
	}
}

// sFnApplyAddon downloads and applies the add-on resources for the version stored in the annotation.
// If the version or namespace changed since the last install, it first removes the old resources.
func sFnApplyAddon(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	cfg := readAddonCfg(&s.instance)
	version := cfg.version
	targetNS := cfg.effectiveNamespace()

	ann := s.instance.GetAnnotations()
	previousVersion := ""
	previousNS := ""
	if ann != nil {
		previousVersion = ann[v1alpha1.AnnotationAddonInstalledVersion]
		previousNS = ann[v1alpha1.AnnotationAddonInstalledNamespace]
	}

	// Ensure the target namespace exists and is labelled for Istio sidecar injection.
	if err := ensureNamespace(ctx, r, targetNS); err != nil {
		r.log.With("err", err).Error("failed to ensure addon namespace")
		s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, err.Error())
		return stopWithNoRequeue()
	}

	// Namespace changed → delete old resources from old namespace first.
	if previousNS != "" && previousNS != targetNS && previousVersion != "" {
		r.log.Infof("addon namespace changed %s → %s, removing old resources from %s", previousNS, targetNS, previousNS)
		deleteOldAddonResources(ctx, r, previousVersion, previousNS)
	} else if previousVersion != "" && previousVersion != version {
		// Version changed (same namespace) → delete old manifest's resources.
		r.log.Infof("addon version changed %s → %s, removing old resources first", previousVersion, version)
		deleteOldAddonResources(ctx, r, previousVersion, targetNS)
	}

	r.log.Infof("fetching HTTP add-on resources for version %s", version)
	objs, err := addon.FetchResources(r.HTTPClient, version)
	if err != nil {
		r.log.With("err", err).Error("failed to fetch addon resources")
		s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, err.Error())
		return stopWithNoRequeue()
	}

	overrideNamespace(objs, targetNS)
	objs = append(objs, addon.NetworkPolicies(targetNS)...)

	var applyErr error
	for i := range objs {
		obj := &objs[i]
		err := r.Patch(ctx, obj, client.Apply, &client.PatchOptions{
			Force:        ptr.To(true),
			FieldManager: "keda-manager",
		})
		if err != nil {
			r.log.With("err", err).With("name", obj.GetName()).Error("addon apply error")
			applyErr = errors.Join(applyErr, err)
		}
	}

	if applyErr != nil {
		s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, applyErr.Error())
		return stopWithNoRequeue()
	}

	r.AddonObjs = objs
	// Record what was successfully installed in tracking annotations.
	setAnnotation(&s.instance, v1alpha1.AnnotationAddonInstalledVersion, version)
	setAnnotation(&s.instance, v1alpha1.AnnotationAddonInstalledNamespace, targetNS)
	s.instance.SetAddonCondition(metav1.ConditionTrue, v1alpha1.ConditionReasonAddonInstalled,
		fmt.Sprintf("HTTP add-on v%s installed in namespace %s", version, targetNS))
	r.log.Infof("HTTP add-on v%s installed in namespace %s", version, targetNS)
	return stopWithNoRequeue()
}

// sFnDeleteAddon removes all cluster resources that belong to the add-on.
// If AddonObjs is empty (e.g. after a controller restart) it re-fetches the
// manifest for the last known version recorded in the tracking annotation so it can still delete.
func sFnDeleteAddon(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	objs := r.AddonObjs

	ann := s.instance.GetAnnotations()
	lastVersion := ""
	lastNS := v1alpha1.DefaultAddonNamespace
	if ann != nil {
		lastVersion = ann[v1alpha1.AnnotationAddonInstalledVersion]
		if ns := ann[v1alpha1.AnnotationAddonInstalledNamespace]; ns != "" {
			lastNS = ns
		}
	}

	if len(objs) == 0 {
		if lastVersion == "" {
			// Nothing was ever installed or already cleaned up.
			s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonDisabled, "HTTP add-on is disabled")
			return stopWithNoRequeue()
		}

		r.log.Infof("AddonObjs empty after restart, re-fetching manifest for version %s to delete from namespace %s", lastVersion, lastNS)
		var fetchErr error
		objs, fetchErr = addon.FetchResources(r.HTTPClient, lastVersion)
		if fetchErr != nil {
			r.log.With("err", fetchErr).Error("failed to re-fetch addon manifest for deletion")
			s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, fetchErr.Error())
			return stopWithNoRequeue()
		}

		overrideNamespace(objs, lastNS)
		objs = append(objs, addon.NetworkPolicies(lastNS)...)
	}

	var delErr error
	for i := range objs {
		obj := objs[i]
		if err := r.Delete(ctx, &obj); client.IgnoreNotFound(err) != nil {
			r.log.With("err", err).With("name", obj.GetName()).Error("addon delete error")
			delErr = errors.Join(delErr, err)
		}
	}

	if delErr != nil {
		s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, delErr.Error())
		return stopWithNoRequeue()
	}

	r.AddonObjs = nil
	// Clear tracking annotations.
	setAnnotation(&s.instance, v1alpha1.AnnotationAddonInstalledVersion, "")
	setAnnotation(&s.instance, v1alpha1.AnnotationAddonInstalledNamespace, "")
	s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonDeleted, "HTTP add-on removed")
	r.log.Info("HTTP add-on removed")
	return stopWithNoRequeue()
}

// deleteAddonObjs is a helper called during full Keda CR deletion to clean up any addon resources.
func deleteAddonObjs(ctx context.Context, r *fsm) error {
	var delErr error
	for _, obj := range r.AddonObjs {
		o := unstructured.Unstructured{}
		o.SetGroupVersionKind(obj.GroupVersionKind())
		o.SetName(obj.GetName())
		o.SetNamespace(obj.GetNamespace())
		if err := r.Delete(ctx, &o); client.IgnoreNotFound(err) != nil {
			delErr = errors.Join(delErr, err)
		}
	}
	return delErr
}
