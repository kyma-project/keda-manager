package reconciler

import (
	"context"
	"errors"
	"fmt"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/kyma-project/keda-manager/pkg/addon"
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

// sFnApplyAddon downloads and applies the add-on resources for the version stored in spec.addon.version.
// If the version changed since the last install, it first removes the old version's resources.
func sFnApplyAddon(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	version := s.instance.Spec.Addon.Version
	previousVersion := s.instance.Status.AddonVersion

	// Version changed → delete the old manifest's resources before applying the new one.
	if previousVersion != "" && previousVersion != version {
		r.log.Infof("addon version changed %s → %s, removing old resources first", previousVersion, version)
		oldObjs, err := addon.FetchResources(r.HTTPClient, previousVersion)
		if err != nil {
			r.log.With("err", err).Warn("failed to fetch old addon manifest for cleanup, proceeding with apply anyway")
		} else {
			for i := range oldObjs {
				obj := oldObjs[i]
				if err := r.Delete(ctx, &obj); client.IgnoreNotFound(err) != nil {
					r.log.With("err", err).With("name", obj.GetName()).Warn("failed to delete old addon resource")
				}
			}
		}
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
		fmt.Sprintf("HTTP add-on v%s installed", version),
	)
	s.instance.Status.AddonVersion = version
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

		r.log.Infof("AddonObjs empty after restart, re-fetching manifest for version %s to delete", lastVersion)
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
