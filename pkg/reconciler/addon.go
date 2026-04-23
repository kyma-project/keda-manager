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
	if ann == nil {
		return addonCfg{}
	}
	return addonCfg{
		enabled:   strings.EqualFold(ann[v1alpha1.AnnotationAddonEnabled], "true"),
		version:   ann[v1alpha1.AnnotationAddonVersion],
		namespace: ann[v1alpha1.AnnotationAddonNamespace],
	}
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

// fetchAddonObjs downloads the add-on manifest, overrides namespaces, and
// appends network policies.
func fetchAddonObjs(r *fsm, version, namespace string) ([]unstructured.Unstructured, error) {
	objs, err := addon.FetchResources(r.HTTPClient, version)
	if err != nil {
		return nil, err
	}
	overrideNamespace(objs, namespace)
	objs = append(objs, addon.NetworkPolicies(namespace)...)
	return objs, nil
}

// applyObjects server-side-applies every object and returns a joined error.
func applyObjects(ctx context.Context, r *fsm, objs []unstructured.Unstructured) error {
	var applyErr error
	for i := range objs {
		if err := r.Patch(ctx, &objs[i], client.Apply, &client.PatchOptions{
			Force:        ptr.To(true),
			FieldManager: "keda-manager",
		}); err != nil {
			r.log.With("err", err).With("name", objs[i].GetName()).Error("addon apply error")
			applyErr = errors.Join(applyErr, err)
		}
	}
	return applyErr
}

// deleteObjects deletes every object, ignoring NotFound errors.
func deleteObjects(ctx context.Context, r *fsm, objs []unstructured.Unstructured) error {
	var delErr error
	for i := range objs {
		if err := r.Delete(ctx, &objs[i]); client.IgnoreNotFound(err) != nil {
			r.log.With("err", err).With("name", objs[i].GetName()).Error("addon delete error")
			delErr = errors.Join(delErr, err)
		}
	}
	return delErr
}

// sFnHandleAddon decides whether to apply or delete the HTTP add-on based on
// annotations on the Keda CR.
func sFnHandleAddon(_ context.Context, _ *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	cfg := readAddonCfg(&s.instance)

	if !cfg.enabled {
		return switchState(sFnDeleteAddon)
	}

	version := cfg.version
	if version == "" {
		return switchState(sFnResolveAddonVersion)
	}

	cleanVersion, err := addon.ValidateVersion(version)
	if err != nil {
		s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonVersionErr, err.Error())
		return stopWithNoRequeue()
	}

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

// sFnApplyAddon downloads and applies the add-on resources. If the version or
// namespace changed since the last install it first removes the old resources.
func sFnApplyAddon(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	cfg := readAddonCfg(&s.instance)
	version := cfg.version
	targetNS := cfg.effectiveNamespace()

	ann := s.instance.GetAnnotations()
	prevVersion, prevNS := "", ""
	if ann != nil {
		prevVersion = ann[v1alpha1.AnnotationAddonInstalledVersion]
		prevNS = ann[v1alpha1.AnnotationAddonInstalledNamespace]
	}

	if err := ensureNamespace(ctx, r, targetNS); err != nil {
		r.log.With("err", err).Error("failed to ensure addon namespace")
		s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, err.Error())
		return stopWithNoRequeue()
	}

	if prevVersion != "" {
		switch {
		case prevNS != "" && prevNS != targetNS:
			r.log.Infof("addon namespace changed %s → %s, removing old resources", prevNS, targetNS)
			cleanupOldAddon(ctx, r, prevVersion, prevNS)
		case prevVersion != version:
			r.log.Infof("addon version changed %s → %s, removing old resources", prevVersion, version)
			cleanupOldAddon(ctx, r, prevVersion, targetNS)
		}
	}

	r.log.Infof("fetching HTTP add-on resources for version %s", version)
	objs, err := fetchAddonObjs(r, version, targetNS)
	if err != nil {
		r.log.With("err", err).Error("failed to fetch addon resources")
		s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, err.Error())
		return stopWithNoRequeue()
	}

	if applyErr := applyObjects(ctx, r, objs); applyErr != nil {
		s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, applyErr.Error())
		return stopWithNoRequeue()
	}

	r.AddonObjs = objs
	setAnnotation(&s.instance, v1alpha1.AnnotationAddonInstalledVersion, version)
	setAnnotation(&s.instance, v1alpha1.AnnotationAddonInstalledNamespace, targetNS)
	s.instance.SetAddonCondition(metav1.ConditionTrue, v1alpha1.ConditionReasonAddonInstalled,
		fmt.Sprintf("HTTP add-on v%s installed in namespace %s", version, targetNS))
	r.log.Infof("HTTP add-on v%s installed in namespace %s", version, targetNS)
	return stopWithNoRequeue()
}

// sFnDeleteAddon removes all cluster resources that belong to the add-on.
// If AddonObjs is empty it re-fetches the manifest to delete.
func sFnDeleteAddon(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	objs := r.AddonObjs

	ann := s.instance.GetAnnotations()
	lastVersion, lastNS := "", v1alpha1.DefaultAddonNamespace
	if ann != nil {
		lastVersion = ann[v1alpha1.AnnotationAddonInstalledVersion]
		if ns := ann[v1alpha1.AnnotationAddonInstalledNamespace]; ns != "" {
			lastNS = ns
		}
	}

	if len(objs) == 0 {
		if lastVersion == "" {
			s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonDisabled, "HTTP add-on is disabled")
			return stopWithNoRequeue()
		}

		r.log.Infof("re-fetching manifest for version %s to delete from namespace %s", lastVersion, lastNS)
		var err error
		objs, err = fetchAddonObjs(r, lastVersion, lastNS)
		if err != nil {
			r.log.With("err", err).Error("failed to re-fetch addon manifest for deletion")
			s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, err.Error())
			return stopWithNoRequeue()
		}
	}

	if delErr := deleteObjects(ctx, r, objs); delErr != nil {
		s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, delErr.Error())
		return stopWithNoRequeue()
	}

	r.AddonObjs = nil
	setAnnotation(&s.instance, v1alpha1.AnnotationAddonInstalledVersion, "")
	setAnnotation(&s.instance, v1alpha1.AnnotationAddonInstalledNamespace, "")
	s.instance.SetAddonCondition(metav1.ConditionFalse, v1alpha1.ConditionReasonAddonDeleted, "HTTP add-on removed")
	r.log.Info("HTTP add-on removed")
	return stopWithNoRequeue()
}

// cleanupOldAddon fetches the manifest for an old version/namespace and deletes
// all resources. Errors are logged but not propagated.
func cleanupOldAddon(ctx context.Context, r *fsm, version, namespace string) {
	objs, err := fetchAddonObjs(r, version, namespace)
	if err != nil {
		r.log.With("err", err).Warn("failed to fetch old addon manifest for cleanup")
		return
	}
	_ = deleteObjects(ctx, r, objs)
}

// deleteAddonObjs is called during full Keda CR deletion to clean up addon resources.
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
