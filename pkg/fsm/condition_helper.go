package fsm

import "github.com/kyma-project/keda-manager/api/v1alpha1"

// TODO make builder e.g. s.SetCondition().Installed).False(...,...)

type StatusHelper interface {
    Unknown(conditionType v1alpha1.ConditionType)
}

type ConditionHelper interface {
    Installed() StatusHelper
}