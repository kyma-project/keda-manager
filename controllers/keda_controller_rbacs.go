package controllers

// Core API group permissions
//+kubebuilder:rbac:groups="",resources=configmaps;configmaps/status,verbs=create;delete;list;patch;update;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=services,verbs=create;delete;list;patch;update;watch
//+kubebuilder:rbac:groups="",resources=external;pods;secrets;serviceaccounts,verbs=create;delete;list;patch;update;watch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=create;delete
//+kubebuilder:rbac:groups="",resources=limitranges,verbs=list;watch

// API registration and extensions
//+kubebuilder:rbac:groups=apiregistration.k8s.io,resources=apiservices,verbs=create;delete;list;patch;update;watch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings;clusterroles;rolebindings;roles,verbs=bind;create;delete;escalate;list;patch;update;watch
//+kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
//+kubebuilder:rbac:groups=authentication.k8s.io,resources=tokenreviews,verbs=create
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=create;delete;list;patch;update;watch

// REQUIRED WILDCARD PERMISSIONS FOR KEDA FUNCTIONALITY:
// The following wildcard permissions are required because KEDA operator needs to:
// 1. Scale any scalable resource (Deployments, StatefulSets, custom resources, CRDs with /scale subresource)
// 2. Read various resources to gather metrics for scaling decisions (e.g., Prometheus, external metrics, custom resources)
// These permissions are delegated to the keda-operator ClusterRole which KEDA uses at runtime.
// Without these wildcards, KEDA would not be able to scale custom resources or read metrics from arbitrary sources.
//+kubebuilder:rbac:groups="*",resources="*/scale",verbs=get;list;patch;update;watch
//+kubebuilder:rbac:groups="*",resources="*",verbs=get

// Apps API group
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=create;delete;list;patch;update;watch
//+kubebuilder:rbac:groups=apps,resources=statefulsets;replicasets,verbs=list;watch

// Batch API group
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=create;delete;list;patch;update;watch

// Coordination API group
//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=create;delete;list;patch;update;watch

// KEDA resources
//+kubebuilder:rbac:groups="keda.sh",resources=clustertriggerauthentications;clustertriggerauthentications/status;scaledjobs;scaledjobs/finalizers;scaledjobs/status;scaledobjects;scaledobjects/finalizers;scaledobjects/status;triggerauthentications;triggerauthentications/status,verbs=create;delete;list;patch;update;watch
//+kubebuilder:rbac:groups="eventing.keda.sh",resources=cloudeventsources;cloudeventsources/status;clustercloudeventsources;clustercloudeventsources/status,verbs=list;patch;update;watch
//+kubebuilder:rbac:groups="discovery.k8s.io",resources="endpointslices",verbs=list;watch

// HTTP add-on resources – the keda-manager must hold at least the same
// permissions it delegates to the add-on service-accounts via ClusterRoles
// shipped in the upstream manifest (RBAC escalation prevention).
//+kubebuilder:rbac:groups="http.keda.sh",resources=httpscaledobjects;httpscaledobjects/status;httpscaledobjects/finalizers,verbs=create;delete;get;list;patch;update;watch

// External metrics API
//+kubebuilder:rbac:groups=external.metrics.k8s.io,resources=externalmetrics,verbs=list;watch

// Autoscaling
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=create;delete;list;patch;update;watch

// Webhooks
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=create;delete;list;patch;update;watch

// Kyma Keda operator resources
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=kedas,verbs=create;delete;list;patch;update;watch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=kedas/status,verbs=patch;update
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=kedas/finalizers,verbs=patch;update

// Network policies
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=create;delete;list;patch;update;watch
