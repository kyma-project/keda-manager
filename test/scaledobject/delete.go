package scaledobject

import (
	"github.com/kyma-project/keda-manager/test/utils"
)

func Delete(utils *utils.TestUtils) error {
	scaledobject := fixScaledObject(utils)

	return utils.Client.Delete(utils.Ctx, scaledobject)
}
