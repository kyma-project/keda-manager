package reconciler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/kyma-project/keda-manager/pkg/addon"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ensureNamespace(ctx context.Context, r *fsm, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	err := r.Create(ctx, ns)
	if err == nil {
		r.log.Infof("created namespace %s", namespace)
		return nil
	}
	if !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
	}
	return nil
}

const (
	istioExcludeInboundPortsAnnotation = "traffic.sidecar.istio.io/excludeInboundPorts"
	istioExcludeInboundPortsValue      = "9090"
	istioSidecarInjectAnnotation       = "sidecar.istio.io/inject"
	kymaModuleLabel                    = "kyma-project.io/module"
	kymaModuleLabelValue               = "keda"

	httpScaledObjectGroup   = "http.keda.sh"
	httpScaledObjectVersion = "v1alpha1"
	httpScaledObjectKind    = "HTTPScaledObject"

	guardAddonInUseRequeue = 30 * time.Second
)

var namespaceEnvVars = map[string]struct{}{
	"KEDA_HTTP_SCALER_TARGET_ADMIN_NAMESPACE": {},
	"KEDA_HTTP_OPERATOR_NAMESPACE":            {},
}

func overrideNamespace(objs []unstructured.Unstructured, namespace string, istioInjection bool) {
	for i := range objs {
		obj := &objs[i]
		if obj.GetNamespace() != "" {
			obj.SetNamespace(namespace)
		}
		applyCommonMetadataLabels(obj)
		switch obj.GetKind() {
		case "ClusterRoleBinding", "RoleBinding":
			patchSubjectsNamespace(obj, namespace)
		case "Deployment":
			patchDeploymentEnvNamespace(obj, namespace)
			patchDeploymentPodTemplateLabels(obj)
			if istioInjection {
				patchDeploymentIstioExcludePortsAnnotation(obj)
				patchDeploymentIstioSidecarAnnotation(obj, "true")
			} else {
				patchDeploymentIstioSidecarAnnotation(obj, "false")
			}
		}
	}
}

// applyCommonMetadataLabels stamps the standard Kyma module labels
// (kyma-project.io/module=keda, app.kubernetes.io/part-of=keda-manager,
// app.kubernetes.io/managed-by=keda-manager) on the object's top-level
// metadata so add-on resources (Deployments, Services, etc.) are discoverable
// via the same label selectors as the rest of the Keda module.
func applyCommonMetadataLabels(obj *unstructured.Unstructured) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	obj.SetLabels(setCommonLabels(labels))
}

func patchDeploymentIstioExcludePortsAnnotation(obj *unstructured.Unstructured) {
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

func patchDeploymentIstioSidecarAnnotation(obj *unstructured.Unstructured, value string) {
	annotations, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
	if annotations == nil {
		annotations = map[string]string{}
	}
	if annotations[istioSidecarInjectAnnotation] == value {
		return
	}
	annotations[istioSidecarInjectAnnotation] = value
	_ = unstructured.SetNestedStringMap(obj.Object, annotations, "spec", "template", "metadata", "annotations")
}

// patchDeploymentPodTemplateLabels stamps `kyma-project.io/module=keda` on
// the Deployment's pod template so add-on Pods (interceptor, operator, scaler)
// are discoverable via the standard Kyma module label selector.
func patchDeploymentPodTemplateLabels(obj *unstructured.Unstructured) {
	labels, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "labels")
	if labels == nil {
		labels = map[string]string{}
	}
	if labels[kymaModuleLabel] == kymaModuleLabelValue {
		return
	}
	labels[kymaModuleLabel] = kymaModuleLabelValue
	_ = unstructured.SetNestedStringMap(obj.Object, labels, "spec", "template", "metadata", "labels")
}

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

func fetchAddonObjs(r *fsm, version, namespace string, istioInjection bool) ([]unstructured.Unstructured, error) {
	objs, err := addon.FetchResources(r.HTTPClient, version)
	if err != nil {
		return nil, err
	}
	overrideNamespace(objs, namespace, istioInjection)
	objs = append(objs, addon.NetworkPolicies(namespace)...)
	return objs, nil
}

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

func sFnHandleAddon(_ context.Context, _ *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	cfg := v1alpha1.ReadAddonCfg(&s.instance)
	if !cfg.Enabled {
		return switchState(sFnGuardAddonInUse)
	}
	return switchState(sFnApplyAddon)
}

// httpScaledObjectsInUse returns the number of HTTPScaledObjects on the
// cluster. A missing CRD (NoKindMatchError / IsNotFound) returns (0, nil) —
// there is nothing to block because the type doesn't exist. RBAC / API errors
// are returned to the caller so they can surface a Warning and requeue.
func httpScaledObjectsInUse(ctx context.Context, c client.Client) (int, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   httpScaledObjectGroup,
		Version: httpScaledObjectVersion,
		Kind:    httpScaledObjectKind + "List",
	})
	if err := c.List(ctx, list); err != nil {
		if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("list HTTPScaledObjects: %w", err)
	}
	return len(list.Items), nil
}

// sFnGuardAddonInUse blocks the disable path of the HTTP add-on while at least
// one HTTPScaledObject exists on the cluster. The user must remove their
// HTTPScaledObjects before the add-on can be uninstalled — otherwise removing
// the CRD would leave orphaned resources of an unknown type.
func sFnGuardAddonInUse(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	count, err := httpScaledObjectsInUse(ctx, r.Client)
	if err != nil {
		msg := fmt.Sprintf("Cannot verify HTTPScaledObject usage: %v", err)
		v1alpha1.SetAddonCondition(&s.instance, metav1.ConditionFalse,
			v1alpha1.ConditionReasonAddonInUse, msg)
		s.instance.Status.State = v1alpha1.StateWarning
		return stopWithRequeueAfter(guardAddonInUseRequeue)
	}
	if count == 0 {
		return switchState(sFnDeleteAddon)
	}
	msg := fmt.Sprintf(
		"%d HTTPScaledObject(s) still exist on the cluster; delete them before disabling the HTTP add-on",
		count)
	v1alpha1.SetAddonCondition(&s.instance, metav1.ConditionFalse,
		v1alpha1.ConditionReasonAddonInUse, msg)
	s.instance.Status.State = v1alpha1.StateWarning
	return stopWithRequeueAfter(guardAddonInUseRequeue)
}

func sFnApplyAddon(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	cfg := v1alpha1.ReadAddonCfg(&s.instance)
	version := v1alpha1.DefaultAddonVersion
	targetNS := cfg.EffectiveNamespace()

	ann := s.instance.GetAnnotations()
	prevVersion, prevNS := "", ""
	if ann != nil {
		prevVersion = ann[v1alpha1.AnnotationAddonInstalledVersion]
		prevNS = ann[v1alpha1.AnnotationAddonInstalledNamespace]
	}

	if err := ensureNamespace(ctx, r, targetNS); err != nil {
		r.log.With("err", err).Error("failed to ensure addon namespace")
		v1alpha1.SetAddonCondition(&s.instance, metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, err.Error())
		return stopWithNoRequeue()
	}

	if prevVersion != "" {
		switch {
		case prevNS != "" && prevNS != targetNS:
			r.log.Infof("addon namespace changed %s -> %s, removing old resources", prevNS, targetNS)
			cleanupOldAddon(ctx, r, prevVersion, prevNS)
		case prevVersion != version:
			r.log.Infof("addon version changed %s -> %s, removing old resources", prevVersion, version)
			cleanupOldAddon(ctx, r, prevVersion, targetNS)
		}
	}

	r.log.Infof("fetching HTTP add-on resources for version %s", version)
	objs, err := fetchAddonObjs(r, version, targetNS, cfg.IstioInjection)
	if err != nil {
		r.log.With("err", err).Error("failed to fetch addon resources")
		v1alpha1.SetAddonCondition(&s.instance, metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, err.Error())
		return stopWithNoRequeue()
	}

	if applyErr := applyObjects(ctx, r, objs); applyErr != nil {
		v1alpha1.SetAddonCondition(&s.instance, metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, applyErr.Error())
		return stopWithNoRequeue()
	}

	r.AddonObjs = objs
	v1alpha1.SetAnnotation(&s.instance, v1alpha1.AnnotationAddonInstalledVersion, version)
	v1alpha1.SetAnnotation(&s.instance, v1alpha1.AnnotationAddonInstalledNamespace, targetNS)
	v1alpha1.SetAddonCondition(&s.instance, metav1.ConditionTrue, v1alpha1.ConditionReasonAddonInstalled,
		fmt.Sprintf("HTTP add-on v%s installed in namespace %s", version, targetNS))

	// Save desired status before r.Update, because r.Update overwrites s.instance
	// with the server response which contains the OLD status from the server
	// (status subresource is not updated by a regular Update call).
	desiredStatus := s.instance.Status.DeepCopy()

	// Persist annotations so next reconcile knows the installed version/namespace.
	if err := r.Update(ctx, &s.instance); err != nil {
		r.log.With("err", err).Error("failed to persist addon annotations after install")
	}

	// Restore desired status after Update (server response overwrites in-memory status).
	s.instance.Status = *desiredStatus

	r.log.Infof("HTTP add-on v%s installed in namespace %s", version, targetNS)
	return stopWithNoRequeue()
}

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
			v1alpha1.SetAddonCondition(&s.instance, metav1.ConditionFalse, v1alpha1.ConditionReasonAddonDisabled, "HTTP add-on is disabled")
			return stopWithNoRequeue()
		}
		r.log.Infof("re-fetching manifest for version %s to delete from namespace %s", lastVersion, lastNS)
		var err error
		objs, err = fetchAddonObjs(r, lastVersion, lastNS, false)
		if err != nil {
			r.log.With("err", err).Error("failed to re-fetch addon manifest for deletion")
			v1alpha1.SetAddonCondition(&s.instance, metav1.ConditionFalse, v1alpha1.ConditionReasonAddonDeleted, err.Error())
			return stopWithNoRequeue()
		}
	}

	if delErr := deleteObjects(ctx, r, objs); delErr != nil {
		v1alpha1.SetAddonCondition(&s.instance, metav1.ConditionFalse, v1alpha1.ConditionReasonAddonInstallErr, delErr.Error())
		return stopWithNoRequeue()
	}

	r.AddonObjs = nil
	v1alpha1.SetAnnotation(&s.instance, v1alpha1.AnnotationAddonInstalledVersion, "")
	v1alpha1.SetAnnotation(&s.instance, v1alpha1.AnnotationAddonInstalledNamespace, "")
	v1alpha1.SetAddonCondition(&s.instance, metav1.ConditionFalse, v1alpha1.ConditionReasonAddonDeleted, "HTTP add-on removed")

	// Save desired status before r.Update, because r.Update overwrites s.instance
	// with the server response which contains the OLD status from the server
	// (status subresource is not updated by a regular Update call).
	desiredStatus := s.instance.Status.DeepCopy()

	// Persist annotation removal on the CR.
	if err := r.Update(ctx, &s.instance); err != nil {
		r.log.With("err", err).Error("failed to persist addon annotations after delete")
	}

	// Restore desired status after Update (server response overwrites in-memory status).
	s.instance.Status = *desiredStatus

	r.log.Info("HTTP add-on removed")
	return stopWithNoRequeue()
}

func cleanupOldAddon(ctx context.Context, r *fsm, version, namespace string) {
	objs, err := fetchAddonObjs(r, version, namespace, false)
	if err != nil {
		r.log.With("err", err).Warn("failed to fetch old addon manifest for cleanup")
		return
	}
	_ = deleteObjects(ctx, r, objs)
}

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
