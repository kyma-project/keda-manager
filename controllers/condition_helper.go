package controllers

import (
	"github.com/kyma-project/keda-manager/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var cHelper ConditionHelper = &conditionHelper{}

type statusHelper v1alpha1.ConditionType

func (s *statusHelper) String() string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func (s *statusHelper) Unknown(reason v1alpha1.ConditionReason, msg string) metav1.Condition {
	return metav1.Condition{
		Type:               s.String(),
		Status:             metav1.ConditionUnknown,
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            msg,
	}
}

func (s *statusHelper) False(reason v1alpha1.ConditionReason, msg string) metav1.Condition {
	return metav1.Condition{
		Type:               s.String(),
		Status:             "False",
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            msg,
	}
}

func (s *statusHelper) True(reason v1alpha1.ConditionReason, msg string) metav1.Condition {
	return metav1.Condition{
		Type:               s.String(),
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            msg,
	}
}

var _ ConditionHelper = &conditionHelper{}

type conditionHelper struct{}

func (c *conditionHelper) Installed() StatusHelper {
	result := statusHelper(v1alpha1.ConditionTypeInstalled)
	return &result
}

type StatusHelper interface {
	Unknown(v1alpha1.ConditionReason, string) metav1.Condition
	False(v1alpha1.ConditionReason, string) metav1.Condition
	True(v1alpha1.ConditionReason, string) metav1.Condition
}

type ConditionHelper interface {
	Installed() StatusHelper
}
