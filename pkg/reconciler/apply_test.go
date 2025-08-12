package reconciler

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	envName = "IMAGE_TEST"
)

func TestUpdateImagesInDeployments(t *testing.T) {
	t.Run("update operator image", func(t *testing.T) {
		t.Setenv(EnvOperatorImage, "newOperatorImage")
		dep := &v1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind: "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "keda-operator",
			},
			Spec: v1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "keda-operator",
								Image: "oldOperatorImage",
							},
						},
					},
				},
			},
		}
		unstructuredDep, err := toUnstructed(dep)
		require.NoError(t, err)

		newUnstructuredDep, err := updateImagesInDeployments(unstructuredDep)
		require.NoError(t, err)
		var changedDep v1.Deployment
		err = fromUnstructured(newUnstructuredDep, &changedDep)
		require.NoError(t, err)
		require.Equal(t, "newOperatorImage", changedDep.Spec.Template.Spec.Containers[0].Image)
	})

	t.Run("don't update other deployments", func(t *testing.T) {
		t.Setenv(EnvOperatorImage, "newOperatorImage")
		dep := &v1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind: "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "not-interesting",
			},
			Spec: v1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "keda-operator",
								Image: "unchanged",
							},
						},
					},
				},
			},
		}
		unstructuredDep, err := toUnstructed(dep)
		require.NoError(t, err)

		newUnstructuredDep, err := updateImagesInDeployments(unstructuredDep)
		require.NoError(t, err)
		var changedDep v1.Deployment
		err = fromUnstructured(newUnstructuredDep, &changedDep)
		require.NoError(t, err)
		require.Equal(t, "unchanged", changedDep.Spec.Template.Spec.Containers[0].Image)
	})

	t.Run("don't update other deployments", func(t *testing.T) {
		t.Setenv(EnvOperatorImage, "newOperatorImage")
		dep := &v1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				Kind: "StatefulSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "keda-operator",
			},
			Spec: v1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "keda-operator",
								Image: "unchanged",
							},
						},
					},
				},
			},
		}
		unstructuredSet, err := toUnstructed(dep)
		require.NoError(t, err)

		newUnstructuredSet, err := updateImagesInDeployments(unstructuredSet)
		require.NoError(t, err)
		var changedDep v1.Deployment
		err = fromUnstructured(newUnstructuredSet, &changedDep)
		require.NoError(t, err)
		require.Equal(t, "unchanged", changedDep.Spec.Template.Spec.Containers[0].Image)
	})
}

func TestUpdateImageIfOverride(t *testing.T) {
	t.Run("Override image", func(t *testing.T) {
		t.Setenv(envName, "newImage")
		dep := &v1.Deployment{
			Spec: v1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Image: "oldImage",
							},
						},
					},
				},
			},
		}

		updateImageIfOverride(envName, dep)
		require.Equal(t, "newImage", dep.Spec.Template.Spec.Containers[0].Image)
	})
	t.Run("Don't override image when empty env", func(t *testing.T) {
		dep := &v1.Deployment{
			Spec: v1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Image: "oldImage",
							},
						},
					},
				},
			},
		}

		updateImageIfOverride(envName, dep)
		require.Equal(t, "oldImage", dep.Spec.Template.Spec.Containers[0].Image)
	})
}
