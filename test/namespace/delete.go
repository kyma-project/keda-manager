package namespace

import "github.com/kyma-project/keda-manager/test/utils"

func Delete(utils *utils.TestUtils) error {
	namespace := fixNamespace(utils)

	return utils.Client.Delete(utils.Ctx, namespace)
}
