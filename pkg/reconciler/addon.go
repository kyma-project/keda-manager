package reconciler

import (
	"context"
	"errors"
	"fmt"

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

// sFnHandleAddon is entered after sFnVerify. It decides whether to apply or
// delete the HTTP add-on based on spec.addon.
func sFnHandleAddon(_ context.Context, _ *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	addonCfg := s.instance.Spec.Addon

	// Add-on disabled (nil, or enabled=false) → delete any existing addon resources.
	if addonCfg == nil || !addonCfg.Enabled {
		return switchState(sFnDeleteAddon)
	}

	version := addonCfg.Version
	// No version specified → resolve the latest one at runtime.
	if version == "" {
		return switchState(sFnResolveAddonVersion)
	}

	version, err := addon.ValidateVersion(version)
	if err != nil {
		s.instance.UpdateAddonStatus(
			v1alpha1.AddonStateError,
			v1alpha1.ConditionReasonAddonVersionErr,
			err.Error(),
		)
		return stopWithNoRequeue()
	}

	// Store the clean version back so sFnApplyAddon can read it.
	s.instance.Spec.Addon.Version = version
	return switchState(sFnApplyAddon)
}

// sFnResolveAddonVersion fetches the latest tag from GitHub and then applies it.
func sFnResolveAddonVersion(_ context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	version, err := addon.LatestVersion(r.HTTPClient)
	if err != nil {
		r.log.With("err", err).Error("failed to resolve latest addon version")
		s.instance.UpdateAddonStatus(
			v1alpha1.AddonStateError,
			v1alpha1.ConditionReasonAddonVersionErr,
			err.Error(),
		)
		return stopWithNoRequeue()
	}
	r.log.Infof("resolved latest HTTP add-on version: %s", version)
	s.instance.Spec.Addon.Version = version
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

// istioExcludePortsDeployments is the set of HTTP add-on Deployment names that
// require the Istio sidecar exclude-inbound-ports annotation so that gRPC on
// port 9090 is not intercepted (which would break health checks).
var istioExcludePortsDeployments = map[string]struct{}{
	"keda-add-ons-http-interceptor": {},
	"keda-add-ons-http-operator":    {},
	"keda-add-ons-http-scaler":      {},
}

const istioExcludeInboundPortsAnnotation = "traffic.sidecar.istio.io/excludeInboundPorts"
const istioExcludeInboundPortsValue = "9090"

// patchDeploymentIstioAnnotation adds the Istio excludeInboundPorts annotation
// to a Deployment if its name is in istioExcludePortsDeployments.
func patchDeploymentIstioAnnotation(obj *unstructured.Unstructured) {
	if _, ok := istioExcludePortsDeployments[obj.GetName()]; !ok {
		return
	}
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

// sFnApplyAddon downloads and applies the add-on resources for the version stored in spec.addon.version.
// If the version or namespace changed since the last install, it first removes the old resources.
func sFnApplyAddon(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	version := s.instance.Spec.Addon.Version
	previousVersion := s.instance.Status.AddonVersion
	previousNS := s.instance.Status.AddonNamespace
	targetNS := s.instance.Spec.Addon.EffectiveNamespace()

	// Ensure the target namespace exists and is labelled for Istio sidecar injection.
	if err := ensureNamespace(ctx, r, targetNS); err != nil {
		r.log.With("err", err).Error("failed to ensure addon namespace")
		s.instance.UpdateAddonStatus(
			v1alpha1.AddonStateError,
			v1alpha1.ConditionReasonAddonInstallErr,
			err.Error(),
		)
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
		s.instance.UpdateAddonStatus(
			v1alpha1.AddonStateError,
			v1alpha1.ConditionReasonAddonInstallErr,
			err.Error(),
		)
		return stopWithNoRequeue()
	}

	// Override namespace on all resources (namespaced + ClusterRoleBinding subjects).
	overrideNamespace(objs, targetNS)

	// Append NetworkPolicies so the add-on components can reach the API server
	// and communicate with each other even when a default-deny policy is in place.
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
		s.instance.UpdateAddonStatus(
			v1alpha1.AddonStateError,
			v1alpha1.ConditionReasonAddonInstallErr,
			applyErr.Error(),
		)
		return stopWithNoRequeue()
	}

	r.AddonObjs = objs
	s.instance.UpdateAddonStatus(
		v1alpha1.AddonStateInstalled,
		v1alpha1.ConditionReasonAddonInstalled,
		fmt.Sprintf("HTTP add-on v%s installed in namespace %s", version, targetNS),
	)
	s.instance.Status.AddonVersion = version
	s.instance.Status.AddonNamespace = targetNS
	return stopWithNoRequeue()
}

// sFnDeleteAddon removes all cluster resources that belong to the add-on.
// If AddonObjs is empty (e.g. after a controller restart) it re-fetches the
// manifest for the last known version so it can still delete the resources.
func sFnDeleteAddon(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	objs := r.AddonObjs

	// AddonObjs is empty after a controller restart. Re-fetch from the last
	// known version recorded in the status so we can still delete.
	if len(objs) == 0 {
		lastVersion := s.instance.Status.AddonVersion
		if lastVersion == "" || s.instance.Status.Addon != v1alpha1.AddonStateInstalled {
			// Nothing was ever installed or already cleaned up.
			if s.instance.Status.Addon != v1alpha1.AddonStateNotInstalled {
				s.instance.UpdateAddonStatus(
					v1alpha1.AddonStateNotInstalled,
					v1alpha1.ConditionReasonAddonDisabled,
					"HTTP add-on is disabled",
				)
			}
			return stopWithNoRequeue()
		}

		// Use the namespace recorded in status (where the addon was actually installed).
		lastNS := s.instance.Status.AddonNamespace
		if lastNS == "" {
			lastNS = v1alpha1.DefaultAddonNamespace
		}

		r.log.Infof("AddonObjs empty after restart, re-fetching manifest for version %s to delete from namespace %s", lastVersion, lastNS)
		var fetchErr error
		objs, fetchErr = addon.FetchResources(r.HTTPClient, lastVersion)
		if fetchErr != nil {
			r.log.With("err", fetchErr).Error("failed to re-fetch addon manifest for deletion")
			s.instance.UpdateAddonStatus(
				v1alpha1.AddonStateError,
				v1alpha1.ConditionReasonAddonInstallErr,
				fetchErr.Error(),
			)
			return stopWithNoRequeue()
		}

		// Override namespace on re-fetched resources so we delete from the correct namespace.
		overrideNamespace(objs, lastNS)
		// Also include NetworkPolicies for deletion.
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
		s.instance.UpdateAddonStatus(
			v1alpha1.AddonStateError,
			v1alpha1.ConditionReasonAddonInstallErr,
			delErr.Error(),
		)
		return stopWithNoRequeue()
	}

	r.AddonObjs = nil
	s.instance.UpdateAddonStatus(
		v1alpha1.AddonStateNotInstalled,
		v1alpha1.ConditionReasonAddonDeleted,
		"HTTP add-on removed",
	)
	s.instance.Status.AddonVersion = ""
	s.instance.Status.AddonNamespace = ""
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
