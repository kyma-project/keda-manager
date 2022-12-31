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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionReason string

type ConditionType string

const (
	ConditionReasonCrdError      = ConditionReason("CrdError")
	ConditionReasonApplyObjError = ConditionReason("ApplyObjError")
	ConditionReasonVerification  = ConditionReason("Verification")

	ConditionTypeInstalled = ConditionType("Installed")
	OperatorLogLevelDebug  = OperatorLogLevel("debug")
	OperatorLogLevelInfo   = OperatorLogLevel("info")
	OperatorLogLevelError  = OperatorLogLevel("error")

	LogFormatJSON    = LogFormat("json")
	LogFormatConsole = LogFormat("console")

	TimeEncodingEpoch       = LogTimeEncoding("epoch")
	TimeEncodingMillis      = LogTimeEncoding("millis")
	TimeEncodingNano        = LogTimeEncoding("nano")
	TimeEncodingISO8601     = LogTimeEncoding("iso8601")
	TimeEncodingRFC3339     = LogTimeEncoding("rfc3339")
	TimeEncodingRFC3339Nano = LogTimeEncoding("rfc3339nano")

	MetricsServerLogLevelInfo  = MetricsServerLogLevel("0")
	MetricsServerLogLevelDebug = MetricsServerLogLevel("4")

	Finalizer = "keda-manager.kyma-project.io/deletion-hook"
)

// +kubebuilder:validation:Enum=debug;info;error
type OperatorLogLevel string

func (l *OperatorLogLevel) String() string {
	if l == nil {
		return ""
	}
	return string(*l)
}

// +kubebuilder:validation:Enum=json;console
type LogFormat string

func (f *LogFormat) String() string {
	if f == nil {
		return ""
	}
	return string(*f)
}

// +kubebuilder:validation:Enum=epoch;millis;nano;iso8601;rfc3339;rfc3339nano
type LogTimeEncoding string

func (e *LogTimeEncoding) String() string {
	if e == nil {
		return ""
	}
	return string(*e)
}

type LoggingOperatorCfg struct {
	Level        *OperatorLogLevel `json:"level,omitempty"`
	Format       *LogFormat        `json:"format,omitempty"`
	TimeEncoding *LogTimeEncoding  `json:"timeEncoding,omitempty"`
}

// +kubebuilder:validation:Enum="0";"4"
type MetricsServerLogLevel string

type LoggingMetricsSrvCfg struct {
	Level *MetricsServerLogLevel `json:"level,omitempty"`
}

type LoggingCfg struct {
	Operator      *LoggingOperatorCfg   `json:"operator,omitempty"`
	MetricsServer *LoggingMetricsSrvCfg `json:"metricServer,omitempty"`
}

type Resources struct {
	Operator      *corev1.ResourceRequirements `json:"operator,omitempty"`
	MetricsServer *corev1.ResourceRequirements `json:"metricServer,omitempty"`
}

type NameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// KedaSpec defines the desired state of Keda
type KedaSpec struct {
	Logging   *LoggingCfg `json:"logging,omitempty"`
	Resources *Resources  `json:"resources,omitempty"`
	Env       []NameValue `json:"env,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="generation",type="integer",JSONPath=".metadata.generation"
//+kubebuilder:printcolumn:name="age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="state",type="string",JSONPath=".status.state"

// Keda is the Schema for the kedas API
type Keda struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KedaSpec `json:"spec,omitempty"`
	Status Status   `json:"status,omitempty"`
}

type Status struct {
	State      string             `json:"state"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true

// KedaList contains a list of Keda
type KedaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Keda `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Keda{}, &KedaList{})
}
