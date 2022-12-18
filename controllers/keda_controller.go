/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/kyma-project/module-manager/pkg/declarative"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"k8s.io/client-go/rest"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	chartNs = "kyma-system"
)

// KedaReconciler reconciles a Keda object
type KedaReconciler struct {
	declarative.ManifestReconciler
	client.Client
	Scheme *runtime.Scheme
	*rest.Config
	ChartPath string
}

//+kubebuilder:rbac:groups="*",resources="*",verbs=get
//+kubebuilder:rbac:groups=external.metrics.k8s.io,resources="*",verbs="*"
//+kubebuilder:rbac:groups="",resources=configmaps;configmaps/status;events;services,verbs="*"
//+kubebuilder:rbac:groups="",resources=external;pods;secrets;serviceaccounts,verbs=list;watch;create;delete;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=create;delete
//+kubebuilder:rbac:groups=apiregistration.k8s.io,resources=apiservices,verbs=create;delete;update;patch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings;clusterroles;rolebindings,verbs=create;delete;update;patch
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=create;delete;update;patch
//+kubebuilder:rbac:groups="*",resources="*/scale",verbs="*"
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;create;delete;update;patch
//+kubebuilder:rbac:groups=apps,resources=statefulsets;replicasets,verbs=list;watch
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs="*"
//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs="*"
//+kubebuilder:rbac:groups="keda.sh",resources=clustertriggerauthentications;clustertriggerauthentications/status;scaledjobs;scaledjobs/finalizers;scaledjobs/status;scaledobjects;scaledobjects/finalizers;scaledobjects/status;triggerauthentications;triggerauthentications/status,verbs="*"
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs="*"

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=kedas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=kedas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=kedas/finalizers,verbs=update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *KedaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Config = mgr.GetConfig()

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Keda{}).
		Complete(r)
}

type ReconciliationAction func(context.Context, client.Client, v1alpha1.Keda) (*ctrl.Result, error)

type result int

const (
	ommit result = iota
	patch
	apply
)

func buildInstallCRDs(expectedCRDs []apiextensionsv1.CustomResourceDefinition, name string) ReconciliationAction {
	crdReq, err := labels.NewRequirement("app.kubernetes.io/name", selection.Equals, []string{name})
	if err != nil {
		panic(err)
	}

	opts := &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(*crdReq),
	}

	return func(ctx context.Context, cli client.Client, instance v1alpha1.Keda) (*ctrl.Result, error) {
		var crdList apiextensionsv1.CustomResourceDefinitionList

		// list all labeled CRDs
		if err := cli.List(ctx, &crdList, opts); err != nil {
			return &defaultResult, err
		}

		for _, crd := range expectedCRDs {
			// check if CRD was already applied
			keyObj := client.ObjectKeyFromObject(&crd)
			err := cli.Get(ctx, keyObj, &crd)
			// the CRD is already applied, continue with next one
			if err != nil {
				continue
			}
			// error while getting CRD stop reconciliation and return error
			if !apierrors.IsNotFound(err) {
				return &defaultResult, err
			}
			// apply CRD
			if err := cli.Create(ctx, &crd); err != nil {
				return &defaultResult, err
			}
		}

		return nil, nil
	}
}

func buildDeleteAllResources(objKey client.ObjectKey) ReconciliationAction {
	return nil
}

func init() {
	file, err := os.Open("/tmp/templated.json")
	if err != nil {
		panic(err)
	}

	if err := json.NewDecoder(file).Decode(&clustertriggerauthentications_keda_sh); err != nil {
		panic(err)
	}
}

var (
	finalizer = "keda-manager.kyma-project.io/deletion-hook"

	clustertriggerauthentications_keda_sh apiextensionsv1.CustomResourceDefinition

	defaultResult = ctrl.Result{
		Requeue: true,
	}

	kedaObjKey = client.ObjectKey{
		Name:      "keda",
		Namespace: "kyma-system",
	}

	reconciliationActions = []ReconciliationAction{
		// handleIstio(..., installCRD),
		buildInstallCRDs([]apiextensionsv1.CustomResourceDefinition{clustertriggerauthentications_keda_sh}, "keda-manager"),
	}

	deleteResources = []ReconciliationAction{
		buildDeleteAllResources(kedaObjKey),
	}
)

func (r *KedaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var instance v1alpha1.Keda
	err := r.Get(ctx, req.NamespacedName, &instance)

	if err != nil {
		return defaultResult, client.IgnoreNotFound(err)
	}

	instanceIsBeingDeleted := !instance.DeletionTimestamp.IsZero()
	instanceHasFinalizer := !controllerutil.ContainsFinalizer(&instance, finalizer)

	// in case instance does not have finalizer - add it and update instance
	if !instanceIsBeingDeleted && !instanceHasFinalizer {
		controllerutil.AddFinalizer(&instance, finalizer)
		if err := r.Update(ctx, &instance); err != nil {
			return defaultResult, err
		}
		return defaultResult, nil
	}

	// in case instance has no finalizer and instance is being deleted - end reconciliation
	if instanceIsBeingDeleted && !controllerutil.ContainsFinalizer(&instance, finalizer) {
		return defaultResult, nil
	}

	// in case instance is being deleted and has finalizer - delete all resources
	if instanceIsBeingDeleted {
		reconciliationActions = deleteResources
	}

	// perform all the reconciliation actions; in case instance is being deleted,
	// the appropriate action(s) should be added be in reconciliationActions slice
	for _, f := range reconciliationActions {
		actionResult, err := f(ctx, r.Client, instance)
		// return reconcile action result if possible
		if actionResult != nil {
			return *actionResult, err
		}
		// stop reconciliation if reconcile action returned error
		if err != nil {
			return defaultResult, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}
