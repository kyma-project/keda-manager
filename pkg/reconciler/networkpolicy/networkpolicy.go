// Package networkpolicy implements utilities for creating Kubernetes NetworkPolicy resources
// based on: https://github.com/kyma-project/kyma/issues/18818#issue-3375929050
package networkpolicy

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func New(name, namespace string, podSelector map[string]string) *v1.NetworkPolicy {
	return &v1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: podSelector,
			},
			PolicyTypes: []v1.PolicyType{
				v1.PolicyTypeIngress,
				v1.PolicyTypeEgress,
			},
			Ingress: newMetricsScrappingIngressRule(),
			Egress: append(
				newAPIServerEgressRule(),
				newDNSEgressRule()...,
			),
		},
	}
}

// newMetricsScrappingIngressRule creates a NetworkPolicyIngressRule that allows
// incoming traffic from pods in the 'kyma-system' namespace with the label
// 'networking.kyma-project.io/metrics-scraping=allowed' on port 8080 (standard
// metrics port for KEDA workloads).
// This rule is used to enable metrics scraping for KEDA components.
func newMetricsScrappingIngressRule() []v1.NetworkPolicyIngressRule {
	return []v1.NetworkPolicyIngressRule{
		{
			From: []v1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "kyma-system",
						},
					},
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"networking.kyma-project.io/metrics-scraping": "allowed",
						},
					},
				},
			},
			Ports: []v1.NetworkPolicyPort{
				{
					Protocol: ptr.To(corev1.ProtocolTCP),
					// standard metrics port for all KEDA workloads
					Port: ptr.To(intstr.FromInt(8080)),
				},
			},
		},
	}
}

// newAPIServerEgressRule creates a NetworkPolicyEgressRule that allows
// outgoing traffic to the Kubernetes API server on port 443.
// This rule is necessary for KEDA components to communicate with the API server.
func newAPIServerEgressRule() []v1.NetworkPolicyEgressRule {
	return []v1.NetworkPolicyEgressRule{
		{
			Ports: []v1.NetworkPolicyPort{
				{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt(443)),
				},
			},
		},
	}
}

// newDNSEgressRule creates NetworkPolicyEgressRules that allow
// outgoing DNS traffic (both TCP and UDP on port 53) to any IP address
// as well as to specific DNS resolver pods in the 'kube-system' namespace.
// This rule is necessary for KEDA components to resolve domain names.
func newDNSEgressRule() []v1.NetworkPolicyEgressRule {
	return []v1.NetworkPolicyEgressRule{
		{
			To: []v1.NetworkPolicyPeer{
				{
					IPBlock: &v1.IPBlock{
						CIDR: "0.0.0.0/0",
					},
				},
			},
			Ports: []v1.NetworkPolicyPort{
				{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt(53)),
				},
				{
					Protocol: ptr.To(corev1.ProtocolUDP),
					Port:     ptr.To(intstr.FromInt(53)),
				},
			},
		},
		{
			To: []v1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"gardener.cloud/purpose": "kube-system",
						},
					},
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"k8s-app": "node-local-dns",
						},
					},
				},
			},
			Ports: []v1.NetworkPolicyPort{
				{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt(53)),
				},
				{
					Protocol: ptr.To(corev1.ProtocolUDP),
					Port:     ptr.To(intstr.FromInt(53)),
				},
			},
		},
		{
			To: []v1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"gardener.cloud/purpose": "kube-system",
						},
					},
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"k8s-app": "kube-dns",
						},
					},
				},
			},
			Ports: []v1.NetworkPolicyPort{
				{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt(8053)),
				},
				{
					Protocol: ptr.To(corev1.ProtocolUDP),
					Port:     ptr.To(intstr.FromInt(8053)),
				},
			},
		},
	}
}
