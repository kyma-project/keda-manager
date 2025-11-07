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
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/kyma-project/keda-manager/pkg/reconciler/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionReason string

type ConditionType string

const (
	StateReady      = "Ready"
	StateError      = "Error"
	StateWarning    = "Warning"
	StateProcessing = "Processing"
	StateDeleting   = "Deleting"

	ServedTrue  = "True"
	ServedFalse = "False"

	ConditionReasonDeploymentUpdateErr = ConditionReason("KedaDeploymentUpdateErr")
	ConditionReasonVerificationErr     = ConditionReason("VerificationErr")
	ConditionReasonVerified            = ConditionReason("Verified")
	ConditionReasonApplyObjError       = ConditionReason("ApplyObjError")
	ConditionReasonVerification        = ConditionReason("Verification")
	ConditionReasonInitialized         = ConditionReason("Initialized")
	ConditionReasonKedaDuplicated      = ConditionReason("KedaDuplicated")
	ConditionReasonDeletion            = ConditionReason("Deletion")
	ConditionReasonDeletionErr         = ConditionReason("DeletionErr")
	ConditionReasonDeleted             = ConditionReason("Deleted")

	ConditionTypeInstalled = ConditionType("Installed")
	ConditionTypeDeleted   = ConditionType("Deleted")

	OperatorLogLevelDebug = OperatorLogLevel("debug")
	OperatorLogLevelInfo  = OperatorLogLevel("info")
	OperatorLogLevelError = OperatorLogLevel("error")

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

	zapLogLevel           = "--zap-log-level"
	zapEncoder            = "--zap-encoder"
	zapTimeEncoding       = "--zap-time-encoding"
	vMetricServerLogLevel = "--v"
)

// +kubebuilder:validation:Enum=debug;info;error
type OperatorLogLevel string

func (l *OperatorLogLevel) zero() string {
	return string(OperatorLogLevelInfo)
}

func (l *OperatorLogLevel) String() string {
	value := l.zero()
	if l != nil {
		value = string(*l)
	}
	return fmt.Sprintf("%s=%s", zapLogLevel, value)
}

func (l *OperatorLogLevel) Match(s *string) bool {
	return strings.HasPrefix(*s, zapLogLevel)
}

// +kubebuilder:validation:Enum=json;console
type LogFormat string

func (f *LogFormat) zero() string {
	return string(LogFormatConsole)
}

func (f *LogFormat) String() string {
	value := f.zero()
	if f != nil {
		value = string(*f)
	}
	return fmt.Sprintf("%s=%s", zapEncoder, value)
}

func (f *LogFormat) Match(s *string) bool {
	return strings.HasPrefix(*s, zapEncoder)
}

// +kubebuilder:validation:Enum=epoch;millis;nano;iso8601;rfc3339;rfc3339nano
type LogTimeEncoding string

func (e *LogTimeEncoding) zero() string {
	return string(TimeEncodingRFC3339)
}

func (e *LogTimeEncoding) String() string {
	value := e.zero()
	if e != nil {
		value = string(*e)
	}
	return fmt.Sprintf("%s=%s", zapTimeEncoding, value)
}

func (e *LogTimeEncoding) Match(s *string) bool {
	return strings.HasPrefix(*s, zapTimeEncoding)
}

type LoggingOperatorCfg struct {
	Level        *OperatorLogLevel `json:"level,omitempty"`
	Format       *LogFormat        `json:"format,omitempty"`
	TimeEncoding *LogTimeEncoding  `json:"timeEncoding,omitempty"`
}

func (o *LoggingOperatorCfg) list() []api.MatchStringer {
	return []api.MatchStringer{
		o.Level,
		o.Format,
		o.TimeEncoding,
	}
}

func (o *LoggingOperatorCfg) UpdateArg(arg *string) {
	for _, cfgProp := range o.list() {
		if !cfgProp.Match(arg) {
			continue
		}
		*arg = cfgProp.String()
	}
}

// +kubebuilder:validation:Enum="0";"4"
type MetricsServerLogLevel string

func (l *MetricsServerLogLevel) zero() string {
	return string(MetricsServerLogLevelInfo)
}

func (l *MetricsServerLogLevel) String() string {
	value := l.zero()
	if l != nil {
		value = string(*l)
	}
	return fmt.Sprintf("%s=%s", vMetricServerLogLevel, value)
}

func (l *MetricsServerLogLevel) Match(s *string) bool {
	return strings.HasPrefix(*s, vMetricServerLogLevel)
}

type LoggingMetricsSrvCfg struct {
	Level *MetricsServerLogLevel `json:"level,omitempty"`
}

func (o *LoggingMetricsSrvCfg) list() []api.MatchStringer {
	return []api.MatchStringer{o.Level}
}

func (o *LoggingMetricsSrvCfg) UpdateArg(arg *string) {
	for _, cfgProp := range o.list() {
		if !cfgProp.Match(arg) {
			continue
		}
		*arg = cfgProp.String()
	}
}

type IstioCfg struct {
	EnabledSidecarInjection bool `json:"enabledSidecarInjection,omitempty"`
}

type Istio struct {
	Operator     *IstioCfg `json:"operator,omitempty"`
	MetricServer *IstioCfg `json:"metricServer,omitempty"`
}

type LoggingCfg struct {
	Operator      *LoggingOperatorCfg   `json:"operator,omitempty"`
	MetricsServer *LoggingMetricsSrvCfg `json:"metricServer,omitempty"`
}

type Resources struct {
	Operator         *corev1.ResourceRequirements `json:"operator,omitempty"`
	MetricsServer    *corev1.ResourceRequirements `json:"metricServer,omitempty"`
	AdmissionWebhook *corev1.ResourceRequirements `json:"admissionWebhook,omitempty"`
}

type PodAnnotations struct {
	Operator         map[string]string `json:"operator,omitempty"`
	MetricsServer    map[string]string `json:"metricServer,omitempty"`
	AdmissionWebhook map[string]string `json:"admissionWebhook,omitempty"`
}

// KedaSpec defines the desired state of Keda
type KedaSpec struct {
	// +kubebuilder:validation:Required
	EnableNetworkPolicies bool            `json:"enableNetworkPolicies"`
	Istio                 *Istio          `json:"istio,omitempty"`
	Logging               *LoggingCfg     `json:"logging,omitempty"`
	Resources             *Resources      `json:"resources,omitempty"`
	Env                   EnvVars         `json:"env,omitempty"`
	PodAnnotations        *PodAnnotations `json:"podAnnotations,omitempty"`
}

type EnvVars []corev1.EnvVar

var (
	watchNamespace = corev1.EnvVar{
		Name:  "WATCH_NAMESPACE",
		Value: "",
	}
	podName = corev1.EnvVar{
		Name: "POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	}
	podNamespace = corev1.EnvVar{
		Name: "POD_NAMESPACE",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		},
	}
	operatorName = corev1.EnvVar{
		Name:  "OPERATOR_NAME",
		Value: "keda-operator",
	}
	kedaHTTPdefaultTimeout = corev1.EnvVar{
		Name:  "KEDA_HTTP_DEFAULT_TIMEOUT",
		Value: "3000",
	}
	kedaHTTPMinTLSVersion = corev1.EnvVar{
		Name:  "KEDA_HTTP_MIN_TLS_VERSION",
		Value: "TLS12",
	}
)

func (v *EnvVars) zero() []corev1.EnvVar {
	return []corev1.EnvVar{
		watchNamespace,
		podName,
		podNamespace,
		operatorName,
		kedaHTTPdefaultTimeout,
		kedaHTTPMinTLSVersion,
	}
}

func contains(envs []corev1.EnvVar, e corev1.EnvVar) bool {
	for _, env := range envs {
		if env.Name == e.Name {
			return true
		}
	}
	return false
}

func (v *EnvVars) Sanitize() {
	if v == nil {
		result := v.zero()
		*v = result
	}

	var required []corev1.EnvVar
	for _, env := range v.zero() {
		if !contains(*v, env) {
			required = append(required, env)
		}
	}
	*v = append(*v, required...)
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:categories={kyma-modules,kyma-keda}
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

func (k *Keda) UpdateStateFromErr(c ConditionType, r ConditionReason, err error) {
	k.Status.State = StateError
	condition := metav1.Condition{
		Type:               string(c),
		Status:             "False",
		LastTransitionTime: metav1.Now(),
		Reason:             string(r),
		Message:            err.Error(),
	}
	meta.SetStatusCondition(&k.Status.Conditions, condition)
}

func (k *Keda) UpdateStateFromWarning(c ConditionType, r ConditionReason, err error) {
	k.Status.State = StateWarning
	condition := metav1.Condition{
		Type:               string(c),
		Status:             "False",
		LastTransitionTime: metav1.Now(),
		Reason:             string(r),
		Message:            err.Error(),
	}
	meta.SetStatusCondition(&k.Status.Conditions, condition)
}

func (k *Keda) UpdateStateReady(c ConditionType, r ConditionReason, msg string) {
	k.Status.State = StateReady
	condition := metav1.Condition{
		Type:               string(c),
		Status:             "True",
		LastTransitionTime: metav1.Now(),
		Reason:             string(r),
		Message:            msg,
	}
	meta.SetStatusCondition(&k.Status.Conditions, condition)
}

func (k *Keda) UpdateStateProcessing(c ConditionType, r ConditionReason, msg string) {
	k.Status.State = StateProcessing
	condition := metav1.Condition{
		Type:               string(c),
		Status:             "Unknown",
		LastTransitionTime: metav1.Now(),
		Reason:             string(r),
		Message:            msg,
	}
	meta.SetStatusCondition(&k.Status.Conditions, condition)
}

func (k *Keda) UpdateStateDeletion(c ConditionType, r ConditionReason, msg string) {
	k.Status.State = StateDeleting
	condition := metav1.Condition{
		Type:               string(c),
		Status:             "Unknown",
		LastTransitionTime: metav1.Now(),
		Reason:             string(r),
		Message:            msg,
	}
	meta.SetStatusCondition(&k.Status.Conditions, condition)
}

func (k *Keda) UpdateStateDeletionTrue(c ConditionType, r ConditionReason, msg string) {
	k.Status.State = StateDeleting
	condition := metav1.Condition{
		Type:               string(c),
		Status:             "True",
		LastTransitionTime: metav1.Now(),
		Reason:             string(r),
		Message:            msg,
	}
	meta.SetStatusCondition(&k.Status.Conditions, condition)
}

func (k *Keda) UpdateServed(served string) {
	k.Status.Served = served
}

func (k *Keda) IsServedEmpty() bool {
	return k.Status.Served == ""
}

type Status struct {
	State                  string             `json:"state"`
	Served                 string             `json:"served"`
	EnabledNetworkPolicies bool               `json:"enabledNetworkPolicies"`
	KedaVersion            string             `json:"kedaVersion,omitempty"`
	Conditions             []metav1.Condition `json:"conditions,omitempty"`
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
