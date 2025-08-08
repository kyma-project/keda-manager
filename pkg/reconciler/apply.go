package reconciler

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/kyma-project/keda-manager/pkg/annotation"
	"go.uber.org/zap"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EnvOperatorImage  = "IMAGE_KEDA_OPERATOR"
	EnvMetricsImage   = "IMAGE_KEDA_METRICS_APISERVER"
	EnvAdmissionImage = "IMAGE_KEDA_ADMISSION_WEBHOOKS"
)

var (
	ErrInstallation = errors.New("installation error")
)

func sFnApply(ctx context.Context, r *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	var isError bool
	for _, obj := range r.Objs {
		r.log.
			With("gvk", obj.GetObjectKind().GroupVersionKind()).
			With("name", obj.GetName()).
			With("ns", obj.GetNamespace()).
			Debug("applying")

		obj = annotation.AddDoNotEditDisclaimer(obj)
		obj.SetLabels(setCommonLabels(obj.GetLabels()))
		err := r.Patch(ctx, &obj, client.Apply, &client.PatchOptions{
			Force:        ptr.To[bool](true),
			FieldManager: "keda-manager",
		})

		if err != nil {
			r.log.With("err", err).Error("apply error")
			isError = true
		}

		err = updateImagesInDeployments(&obj.Object, r.log)
		if err != nil {
			r.log.With("err", err).Error("update images error")
			isError = true
		}

		s.objs = append(s.objs, obj)
	}
	// no errors
	if !isError {
		return switchState(sFnVerify)
	}

	s.instance.UpdateStateFromErr(
		v1alpha1.ConditionTypeInstalled,
		v1alpha1.ConditionReasonApplyObjError,
		ErrInstallation,
	)
	return stopWithNoRequeue()
}

func updateImagesInDeployments(obj *map[string]interface{}, log *zap.SugaredLogger) error {
	if (*obj)["kind"] == "Deployment" {
		/// convert obj to Deployment
		var dep v1.Deployment
		err := fromUnstructured(*obj, &dep)
		if err != nil {
			return fmt.Errorf("convert from unstructured error: %w", err)
		}

		switch dep.ObjectMeta.Name {
		case "keda-operator":
			// "europe-docker.pkg.dev/kyma-project/prod/external/ghcr.io/kedacore/keda:2.17.0"
			updateImageIfOverride(EnvOperatorImage, &dep)

			// spec.template.spec.containers[0].image
		case "keda-operator-metrics-apiserver":
			// "europe-docker.pkg.dev/kyma-project/prod/external/ghcr.io/kedacore/keda-metrics-apiserver:2.17.0"
			updateImageIfOverride(EnvMetricsImage, &dep)
		case "keda-admission-webhooks":
			// "europe-docker.pkg.dev/kyma-project/prod/external/ghcr.io/kedacore/keda-admission-webhooks:2.17.0"
			updateImageIfOverride(EnvAdmissionImage, &dep)
		}

		converted, err := toUnstructed(dep)
		if err != nil {
			return fmt.Errorf("convert to unstructured error: %w", err)
		}
		obj = &converted
	}
	return nil
}

func updateImageIfOverride(envName string, dep *v1.Deployment) {
	imageName := os.Getenv(envName)
	if imageName != "" {
		dep.Spec.Template.Spec.Containers[0].Image = imageName
	}
}
