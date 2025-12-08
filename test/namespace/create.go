package namespace

import (
	"github.com/kyma-project/keda-manager/test/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Create(utils *utils.TestUtils) error {
	namespace := fixNamespace(utils)

	return utils.Client.Create(utils.Ctx, namespace)
}

func fixNamespace(utils *utils.TestUtils) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: utils.Namespace,
		},
	}
}
